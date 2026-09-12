package auth

import (
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func makeJWTForClaims(t *testing.T, claims map[string]any) string {
	t.Helper()

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	payload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	return header + "." + payload + ".sig"
}

func TestBuildAuthorizeURL(t *testing.T) {
	cfg := OAuthProviderConfig{
		Issuer:     "https://auth.example.com",
		ClientID:   "test-client-id",
		Scopes:     "openid profile",
		Originator: "codex_cli_rs",
		Port:       1455,
	}
	pkce := PKCECodes{
		CodeVerifier:  "test-verifier",
		CodeChallenge: "test-challenge",
	}

	u := BuildAuthorizeURL(cfg, pkce, "test-state", "http://localhost:1455/auth/callback")

	if !strings.HasPrefix(u, "https://auth.example.com/oauth/authorize?") {
		t.Errorf("URL does not start with expected prefix: %s", u)
	}
	if !strings.Contains(u, "client_id=test-client-id") {
		t.Error("URL missing client_id")
	}
	if !strings.Contains(u, "code_challenge=test-challenge") {
		t.Error("URL missing code_challenge")
	}
	if !strings.Contains(u, "code_challenge_method=S256") {
		t.Error("URL missing code_challenge_method")
	}
	if !strings.Contains(u, "state=test-state") {
		t.Error("URL missing state")
	}
	if !strings.Contains(u, "response_type=code") {
		t.Error("URL missing response_type")
	}
	if !strings.Contains(u, "id_token_add_organizations=true") {
		t.Error("URL missing id_token_add_organizations")
	}
	if !strings.Contains(u, "codex_cli_simplified_flow=true") {
		t.Error("URL missing codex_cli_simplified_flow")
	}
	if !strings.Contains(u, "originator=codex_cli_rs") {
		t.Error("URL missing originator")
	}
}

func TestBuildAuthorizeURLOpenAIExtras(t *testing.T) {
	cfg := OpenAIOAuthConfig()
	pkce := PKCECodes{CodeVerifier: "test-verifier", CodeChallenge: "test-challenge"}

	u := BuildAuthorizeURL(cfg, pkce, "test-state", "http://localhost:1455/auth/callback")
	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatalf("url.Parse() error: %v", err)
	}
	q := parsed.Query()

	if q.Get("id_token_add_organizations") != "true" {
		t.Errorf("id_token_add_organizations = %q, want true", q.Get("id_token_add_organizations"))
	}
	if q.Get("codex_cli_simplified_flow") != "true" {
		t.Errorf("codex_cli_simplified_flow = %q, want true", q.Get("codex_cli_simplified_flow"))
	}
	if q.Get("originator") != "codex_cli_rs" {
		t.Errorf("originator = %q, want codex_cli_rs", q.Get("originator"))
	}
}

func TestParseTokenResponse(t *testing.T) {
	resp := map[string]any{
		"access_token":  "test-access-token",
		"refresh_token": "test-refresh-token",
		"expires_in":    3600,
		"id_token":      "test-id-token",
	}
	body, _ := json.Marshal(resp)

	cred, err := parseTokenResponse(body, "openai")
	if err != nil {
		t.Fatalf("parseTokenResponse() error: %v", err)
	}

	if cred.AccessToken != "test-access-token" {
		t.Errorf("AccessToken = %q, want %q", cred.AccessToken, "test-access-token")
	}
	if cred.RefreshToken != "test-refresh-token" {
		t.Errorf("RefreshToken = %q, want %q", cred.RefreshToken, "test-refresh-token")
	}
	if cred.Provider != "openai" {
		t.Errorf("Provider = %q, want %q", cred.Provider, "openai")
	}
	if cred.AuthMethod != "oauth" {
		t.Errorf("AuthMethod = %q, want %q", cred.AuthMethod, "oauth")
	}
	if cred.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should not be zero")
	}
}

func TestParseTokenResponseExtractsAccountIDFromIDToken(t *testing.T) {
	idToken := makeJWTForClaims(t, map[string]any{"chatgpt_account_id": "acc-id-from-id-token"})
	resp := map[string]any{
		"access_token":  "opaque-access-token",
		"refresh_token": "test-refresh-token",
		"expires_in":    3600,
		"id_token":      idToken,
	}
	body, _ := json.Marshal(resp)

	cred, err := parseTokenResponse(body, "openai")
	if err != nil {
		t.Fatalf("parseTokenResponse() error: %v", err)
	}
	if cred.AccountID != "acc-id-from-id-token" {
		t.Errorf("AccountID = %q, want %q", cred.AccountID, "acc-id-from-id-token")
	}
}

func TestExtractAccountIDFromOrganizationsFallback(t *testing.T) {
	token := makeJWTForClaims(t, map[string]any{
		"organizations": []any{
			map[string]any{"id": "org_from_orgs"},
		},
	})

	if got := extractAccountID(token); got != "org_from_orgs" {
		t.Errorf("extractAccountID() = %q, want %q", got, "org_from_orgs")
	}
}

func TestParseTokenResponseNoAccessToken(t *testing.T) {
	body := []byte(`{"refresh_token": "test"}`)
	_, err := parseTokenResponse(body, "openai")
	if err == nil {
		t.Error("expected error for missing access_token")
	}
}

func TestParseTokenResponseAccountIDFromIDToken(t *testing.T) {
	idToken := makeJWTWithAccountID("acc-from-id")
	resp := map[string]any{
		"access_token":  "not-a-jwt",
		"refresh_token": "test-refresh-token",
		"expires_in":    3600,
		"id_token":      idToken,
	}
	body, _ := json.Marshal(resp)

	cred, err := parseTokenResponse(body, "openai")
	if err != nil {
		t.Fatalf("parseTokenResponse() error: %v", err)
	}

	if cred.AccountID != "acc-from-id" {
		t.Errorf("AccountID = %q, want %q", cred.AccountID, "acc-from-id")
	}
}

func makeJWTWithAccountID(accountID string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString(
		[]byte(`{"https://api.openai.com/auth":{"chatgpt_account_id":"` + accountID + `"}}`),
	)
	return header + "." + payload + ".sig"
}

func TestExchangeCodeForTokens(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		r.ParseForm()
		if r.FormValue("grant_type") != "authorization_code" {
			http.Error(w, "invalid grant_type", http.StatusBadRequest)
			return
		}

		resp := map[string]any{
			"access_token":  "mock-access-token",
			"refresh_token": "mock-refresh-token",
			"expires_in":    3600,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := OAuthProviderConfig{
		Issuer:   server.URL,
		ClientID: "test-client",
		Scopes:   "openid",
		Port:     1455,
	}

	cred, err := ExchangeCodeForTokens(cfg, "test-code", "test-verifier", "http://localhost:1455/auth/callback")
	if err != nil {
		t.Fatalf("ExchangeCodeForTokens() error: %v", err)
	}

	if cred.AccessToken != "mock-access-token" {
		t.Errorf("AccessToken = %q, want %q", cred.AccessToken, "mock-access-token")
	}
}

func TestRefreshAccessToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		r.ParseForm()
		if r.FormValue("grant_type") != "refresh_token" {
			http.Error(w, "invalid grant_type", http.StatusBadRequest)
			return
		}

		resp := map[string]any{
			"access_token":  "refreshed-access-token",
			"refresh_token": "refreshed-refresh-token",
			"expires_in":    3600,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := OAuthProviderConfig{
		Issuer:   server.URL,
		ClientID: "test-client",
	}

	cred := &AuthCredential{
		AccessToken:  "old-token",
		RefreshToken: "old-refresh-token",
		Provider:     "openai",
		AuthMethod:   "oauth",
	}

	refreshed, err := RefreshAccessToken(cred, cfg)
	if err != nil {
		t.Fatalf("RefreshAccessToken() error: %v", err)
	}

	if refreshed.AccessToken != "refreshed-access-token" {
		t.Errorf("AccessToken = %q, want %q", refreshed.AccessToken, "refreshed-access-token")
	}
	if refreshed.RefreshToken != "refreshed-refresh-token" {
		t.Errorf("RefreshToken = %q, want %q", refreshed.RefreshToken, "refreshed-refresh-token")
	}
}

func TestRefreshAccessTokenNoRefreshToken(t *testing.T) {
	cfg := OpenAIOAuthConfig()
	cred := &AuthCredential{
		AccessToken: "old-token",
		Provider:    "openai",
		AuthMethod:  "oauth",
	}

	_, err := RefreshAccessToken(cred, cfg)
	if err == nil {
		t.Error("expected error for missing refresh token")
	}
}

func TestRefreshAccessTokenPreservesRefreshAndAccountID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"access_token": "new-access-token-only",
			"expires_in":   3600,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := OAuthProviderConfig{Issuer: server.URL, ClientID: "test-client"}
	cred := &AuthCredential{
		AccessToken:  "old-access",
		RefreshToken: "existing-refresh",
		AccountID:    "acc_existing",
		Provider:     "openai",
		AuthMethod:   "oauth",
	}

	refreshed, err := RefreshAccessToken(cred, cfg)
	if err != nil {
		t.Fatalf("RefreshAccessToken() error: %v", err)
	}
	if refreshed.RefreshToken != "existing-refresh" {
		t.Errorf("RefreshToken = %q, want %q", refreshed.RefreshToken, "existing-refresh")
	}
	if refreshed.AccountID != "acc_existing" {
		t.Errorf("AccountID = %q, want %q", refreshed.AccountID, "acc_existing")
	}
}

func TestOpenAIOAuthConfig(t *testing.T) {
	cfg := OpenAIOAuthConfig()
	if cfg.Issuer != "https://auth.openai.com" {
		t.Errorf("Issuer = %q, want %q", cfg.Issuer, "https://auth.openai.com")
	}
	if cfg.ClientID == "" {
		t.Error("ClientID is empty")
	}
	if cfg.Port != 1455 {
		t.Errorf("Port = %d, want 1455", cfg.Port)
	}
}

func TestParseDeviceCodeResponseIntervalAsNumber(t *testing.T) {
	body := []byte(`{"device_auth_id":"abc","user_code":"DEF-1234","interval":5}`)

	resp, err := parseDeviceCodeResponse(body)
	if err != nil {
		t.Fatalf("parseDeviceCodeResponse() error: %v", err)
	}

	if resp.DeviceAuthID != "abc" {
		t.Errorf("DeviceAuthID = %q, want %q", resp.DeviceAuthID, "abc")
	}
	if resp.UserCode != "DEF-1234" {
		t.Errorf("UserCode = %q, want %q", resp.UserCode, "DEF-1234")
	}
	if resp.Interval != 5 {
		t.Errorf("Interval = %d, want %d", resp.Interval, 5)
	}
}

func TestParseDeviceCodeResponseIntervalAsString(t *testing.T) {
	body := []byte(`{"device_auth_id":"abc","user_code":"DEF-1234","interval":"5"}`)

	resp, err := parseDeviceCodeResponse(body)
	if err != nil {
		t.Fatalf("parseDeviceCodeResponse() error: %v", err)
	}

	if resp.Interval != 5 {
		t.Errorf("Interval = %d, want %d", resp.Interval, 5)
	}
}

func TestParseDeviceCodeResponseInvalidInterval(t *testing.T) {
	body := []byte(`{"device_auth_id":"abc","user_code":"DEF-1234","interval":"abc"}`)

	if _, err := parseDeviceCodeResponse(body); err == nil {
		t.Fatal("expected error for invalid interval")
	}
}

func TestLoginBrowserWithOptionsNoBrowserDoesNotRequireCallbackPort(t *testing.T) {
	server := newMockOAuthTokenServer()
	defer server.Close()
	reservedListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error: %v", err)
	}
	defer reservedListener.Close()

	reservedPort := reservedListener.Addr().(*net.TCPAddr).Port
	origOpenBrowserFunc := openBrowserFunc
	origBrowserLoginInput := browserLoginInput
	t.Cleanup(func() {
		openBrowserFunc = origOpenBrowserFunc
		browserLoginInput = origBrowserLoginInput
	})

	var openCalls int
	openBrowserFunc = func(string) error {
		openCalls++
		return nil
	}
	browserLoginInput = strings.NewReader("manual-code\n")

	cfg := OAuthProviderConfig{
		Issuer:   server.URL,
		ClientID: "test-client",
		Scopes:   "openid",
		Port:     reservedPort,
	}

	cred, err := LoginBrowserWithOptions(cfg, LoginBrowserOptions{NoBrowser: true})
	if err != nil {
		t.Fatalf("LoginBrowserWithOptions() error: %v", err)
	}

	if openCalls != 0 {
		t.Fatalf("openBrowserFunc call count = %d, want 0", openCalls)
	}
	if cred.AccessToken != "mock-access-token" {
		t.Fatalf("AccessToken = %q, want %q", cred.AccessToken, "mock-access-token")
	}
}

func TestLoginBrowserWithOptionsAutoOpensByDefault(t *testing.T) {
	server := newMockOAuthTokenServer()
	defer server.Close()

	origOpenBrowserFunc := openBrowserFunc
	origBrowserLoginInput := browserLoginInput
	t.Cleanup(func() {
		openBrowserFunc = origOpenBrowserFunc
		browserLoginInput = origBrowserLoginInput
	})

	var (
		openCalls  int
		browserURL string
	)
	openBrowserFunc = func(url string) error {
		openCalls++
		browserURL = url
		return nil
	}
	browserLoginInput = strings.NewReader("manual-code\n")

	cfg := OAuthProviderConfig{
		Issuer:   server.URL,
		ClientID: "test-client",
		Scopes:   "openid",
		Port:     0,
	}

	_, err := LoginBrowserWithOptions(cfg, LoginBrowserOptions{})
	if err != nil {
		t.Fatalf("LoginBrowserWithOptions() error: %v", err)
	}

	if openCalls != 1 {
		t.Fatalf("openBrowserFunc call count = %d, want 1", openCalls)
	}

	parsedBrowserURL, err := url.Parse(browserURL)
	if err != nil {
		t.Fatalf("url.Parse(browserURL) error: %v", err)
	}

	redirectURI, err := url.Parse(parsedBrowserURL.Query().Get("redirect_uri"))
	if err != nil {
		t.Fatalf("url.Parse(redirectURI) error: %v", err)
	}
	if redirectURI.Port() == "" {
		t.Fatal("redirectURI port is empty")
	}
	if redirectURI.Port() == "0" {
		t.Fatalf("redirectURI port = %q, want dynamically assigned port", redirectURI.Port())
	}
}

func newMockOAuthTokenServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		resp := map[string]any{
			"access_token":  "mock-access-token",
			"refresh_token": "mock-refresh-token",
			"expires_in":    3600,
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func TestGenerateState_Length(t *testing.T) {
	state, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState() error: %v", err)
	}
	// 32 bytes hex encoded = 64 chars
	if len(state) != 64 {
		t.Errorf("GenerateState() len = %d, want 64", len(state))
	}
}

func TestGenerateState_Unique(t *testing.T) {
	s1, _ := GenerateState()
	s2, _ := GenerateState()
	if s1 == s2 {
		t.Error("GenerateState() should produce unique values")
	}
}

func TestParseFlexibleInt_Null(t *testing.T) {
	v, err := parseFlexibleInt(json.RawMessage("null"))
	if err != nil || v != 0 {
		t.Errorf("parseFlexibleInt null = %d, %v, want 0, nil", v, err)
	}
}

func TestParseFlexibleInt_Empty(t *testing.T) {
	v, err := parseFlexibleInt(json.RawMessage(""))
	if err != nil || v != 0 {
		t.Errorf("parseFlexibleInt empty = %d, %v, want 0, nil", v, err)
	}
}

func TestParseFlexibleInt_Integer(t *testing.T) {
	v, err := parseFlexibleInt(json.RawMessage("42"))
	if err != nil || v != 42 {
		t.Errorf("parseFlexibleInt int = %d, %v, want 42, nil", v, err)
	}
}

func TestParseFlexibleInt_StringInteger(t *testing.T) {
	v, err := parseFlexibleInt(json.RawMessage(`"30"`))
	if err != nil || v != 30 {
		t.Errorf("parseFlexibleInt string int = %d, %v, want 30, nil", v, err)
	}
}

func TestParseFlexibleInt_EmptyString(t *testing.T) {
	v, err := parseFlexibleInt(json.RawMessage(`""`))
	if err != nil || v != 0 {
		t.Errorf("parseFlexibleInt empty string = %d, %v, want 0, nil", v, err)
	}
}

func TestParseFlexibleInt_Invalid(t *testing.T) {
	_, err := parseFlexibleInt(json.RawMessage(`true`))
	if err == nil {
		t.Error("parseFlexibleInt invalid value should return error")
	}
}

func TestGoogleAntigravityOAuthConfig_Fields(t *testing.T) {
	t.Setenv("KHUNQUANT_GOOGLE_ANTIGRAVITY_CLIENT_ID", "test-client-id")
	t.Setenv("KHUNQUANT_GOOGLE_ANTIGRAVITY_CLIENT_SECRET", "test-client-secret")

	cfg := GoogleAntigravityOAuthConfig()
	if cfg.ClientID != "test-client-id" {
		t.Errorf("ClientID = %q, want test-client-id", cfg.ClientID)
	}
	if cfg.ClientSecret != "test-client-secret" {
		t.Errorf("ClientSecret = %q, want test-client-secret", cfg.ClientSecret)
	}
	if !strings.Contains(cfg.Issuer, "google.com") {
		t.Errorf("Issuer should contain google.com, got %q", cfg.Issuer)
	}
	if cfg.Port == 0 {
		t.Error("Port should be set")
	}
}

func TestGoogleOAuthConfigsDoNotEmbedClientCredentials(t *testing.T) {
	t.Setenv("KHUNQUANT_GOOGLE_ANTIGRAVITY_CLIENT_ID", "")
	t.Setenv("KHUNQUANT_GOOGLE_ANTIGRAVITY_CLIENT_SECRET", "")
	t.Setenv("KHUNQUANT_GOOGLE_GEMINI_CLIENT_ID", "")
	t.Setenv("KHUNQUANT_GOOGLE_GEMINI_CLIENT_SECRET", "")

	for name, cfg := range map[string]OAuthProviderConfig{
		"antigravity": GoogleAntigravityOAuthConfig(),
		"gemini":      GoogleGeminiOAuthConfig(),
	} {
		if cfg.ClientID != "" || cfg.ClientSecret != "" {
			t.Fatalf("%s config contains embedded client credentials", name)
		}
	}
}

func TestOAuthCallbackHandler_StateMismatch(t *testing.T) {
	resultCh := make(chan callbackResult, 1)
	handler := oauthCallbackHandler("expected-state", resultCh)

	req, _ := http.NewRequest("GET", "/auth/callback?state=wrong-state&code=abc", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("state mismatch: status = %d, want 400", rr.Code)
	}
	result := <-resultCh
	if result.err == nil || !strings.Contains(result.err.Error(), "state mismatch") {
		t.Errorf("expected state mismatch error, got %v", result.err)
	}
}

func TestOAuthCallbackHandler_MissingCode(t *testing.T) {
	resultCh := make(chan callbackResult, 1)
	handler := oauthCallbackHandler("my-state", resultCh)

	req, _ := http.NewRequest("GET", "/auth/callback?state=my-state&error=access_denied", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("missing code: status = %d, want 400", rr.Code)
	}
	result := <-resultCh
	if result.err == nil {
		t.Error("expected error for missing code")
	}
	if result.code != "" {
		t.Errorf("expected empty code, got %q", result.code)
	}
}

func TestOAuthCallbackHandler_Success(t *testing.T) {
	resultCh := make(chan callbackResult, 1)
	handler := oauthCallbackHandler("good-state", resultCh)

	req, _ := http.NewRequest("GET", "/auth/callback?state=good-state&code=auth-code-123", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("success: status = %d, want 200", rr.Code)
	}
	result := <-resultCh
	if result.err != nil {
		t.Errorf("unexpected error: %v", result.err)
	}
	if result.code != "auth-code-123" {
		t.Errorf("code = %q, want auth-code-123", result.code)
	}
}

func TestRequestDeviceCode_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/accounts/deviceauth/usercode" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"device_auth_id":"did-123","user_code":"ABCD-1234","interval":5}`))
	}))
	defer srv.Close()

	cfg := OAuthProviderConfig{Issuer: srv.URL, ClientID: "test-client"}
	info, err := RequestDeviceCode(cfg)
	if err != nil {
		t.Fatalf("RequestDeviceCode unexpected error: %v", err)
	}
	if info.DeviceAuthID != "did-123" {
		t.Errorf("DeviceAuthID = %q, want did-123", info.DeviceAuthID)
	}
	if info.UserCode != "ABCD-1234" {
		t.Errorf("UserCode = %q, want ABCD-1234", info.UserCode)
	}
}

func TestRequestDeviceCode_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := OAuthProviderConfig{Issuer: srv.URL, ClientID: "test-client"}
	_, err := RequestDeviceCode(cfg)
	if err == nil {
		t.Error("RequestDeviceCode with 500 should return error")
	}
}

func TestRequestDeviceCode_InvalidBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer srv.Close()

	cfg := OAuthProviderConfig{Issuer: srv.URL, ClientID: "test-client"}
	_, err := RequestDeviceCode(cfg)
	if err == nil {
		t.Error("RequestDeviceCode with invalid JSON body should return error")
	}
}

func TestRequestDeviceCode_ZeroInterval(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"device_auth_id":"did-zero","user_code":"ZERO","interval":0}`))
	}))
	defer srv.Close()

	cfg := OAuthProviderConfig{Issuer: srv.URL, ClientID: "test-client"}
	info, err := RequestDeviceCode(cfg)
	if err != nil {
		t.Fatalf("RequestDeviceCode unexpected error: %v", err)
	}
	if info.Interval != 5 {
		t.Errorf("zero interval should default to 5, got %d", info.Interval)
	}
}

func TestExchangeCodeForTokens_WithGoogleTokenURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"google-token","expires_in":3600}`))
	}))
	defer srv.Close()

	cfg := OAuthProviderConfig{
		Issuer:       "https://accounts.google.com",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		TokenURL:     srv.URL + "?googleapis.com=1",
	}
	cred, err := ExchangeCodeForTokens(cfg, "code", "verifier", "http://localhost/callback")
	if err != nil {
		t.Fatalf("ExchangeCodeForTokens: %v", err)
	}
	if cred.Provider != "google-antigravity" {
		t.Errorf("provider = %q, want google-antigravity", cred.Provider)
	}
}

func TestExchangeCodeForTokens_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer srv.Close()

	cfg := OAuthProviderConfig{Issuer: srv.URL, ClientID: "test-client"}
	_, err := ExchangeCodeForTokens(cfg, "code", "verifier", "http://localhost/callback")
	if err == nil {
		t.Error("ExchangeCodeForTokens 403 should return error")
	}
}

func TestExchangeCodeForTokens_NoAccessToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"refresh_token":"rtoken"}`))
	}))
	defer srv.Close()

	cfg := OAuthProviderConfig{Issuer: srv.URL, ClientID: "test-client"}
	_, err := ExchangeCodeForTokens(cfg, "code", "verifier", "http://localhost/callback")
	if err == nil {
		t.Error("ExchangeCodeForTokens with no access_token should return error")
	}
}

func TestPollDeviceCodeOnce_Pending(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "pending", http.StatusAccepted)
	}))
	defer srv.Close()

	cfg := OAuthProviderConfig{Issuer: srv.URL, ClientID: "test-client"}
	cred, err := PollDeviceCodeOnce(cfg, "device-id", "USER-CODE")
	if err == nil {
		t.Error("pending response should return error")
	}
	if cred != nil {
		t.Error("pending response should return nil credential")
	}
}

func TestPollDeviceCodeOnce_NetworkError(t *testing.T) {
	cfg := OAuthProviderConfig{Issuer: "http://localhost:19998", ClientID: "test-client"}
	cred, err := PollDeviceCodeOnce(cfg, "device-id", "USER-CODE")
	if err == nil {
		t.Error("network error should return error")
	}
	if cred != nil {
		t.Error("network error should return nil credential")
	}
}

func TestPollDeviceCodeOnce_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/accounts/deviceauth/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"authorization_code":"auth-code","code_verifier":"verifier","code_challenge":"challenge"}`))
		case "/oauth/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"poll-token","expires_in":3600}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cfg := OAuthProviderConfig{Issuer: srv.URL, ClientID: "test-client"}
	cred, err := PollDeviceCodeOnce(cfg, "device-id", "USER-CODE")
	if err != nil {
		t.Fatalf("PollDeviceCodeOnce unexpected error: %v", err)
	}
	if cred == nil {
		t.Fatal("PollDeviceCodeOnce should return credential on success")
	}
	if cred.AccessToken != "poll-token" {
		t.Errorf("AccessToken = %q, want poll-token", cred.AccessToken)
	}
}

func TestBuildAuthorizeURL_GoogleIssuer(t *testing.T) {
	cfg := OAuthProviderConfig{
		Issuer:   "https://accounts.google.com/o/oauth2/v2",
		ClientID: "google-client-id",
		Scopes:   "openid profile email",
	}
	pkce := PKCECodes{CodeVerifier: "verifier", CodeChallenge: "challenge"}
	u := BuildAuthorizeURL(cfg, pkce, "state-val", "http://localhost:51121/auth/callback")

	if !strings.Contains(u, "accounts.google.com") {
		t.Errorf("Google URL should contain accounts.google.com, got: %s", u)
	}
	if !strings.Contains(u, "access_type=offline") {
		t.Errorf("Google URL should have access_type=offline, got: %s", u)
	}
	if !strings.Contains(u, "prompt=consent") {
		t.Errorf("Google URL should have prompt=consent, got: %s", u)
	}
	if !strings.Contains(u, "/auth?") {
		t.Errorf("Google URL should use /auth path, got: %s", u)
	}
}

func TestParseJWTClaims_NotJWT(t *testing.T) {
	_, err := parseJWTClaims("notajwt")
	if err == nil {
		t.Error("parseJWTClaims with non-JWT should return error")
	}
}

func TestRefreshAccessToken_WithCustomTokenURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"refreshed-token","expires_in":3600}`))
	}))
	defer srv.Close()

	cfg := OAuthProviderConfig{
		Issuer:       "https://auth.example.com",
		ClientID:     "client",
		ClientSecret: "secret",
		TokenURL:     srv.URL,
	}
	cred := &AuthCredential{
		AccessToken:  "old",
		RefreshToken: "rtoken",
		Provider:     "openai",
	}
	refreshed, err := RefreshAccessToken(cred, cfg)
	if err != nil {
		t.Fatalf("RefreshAccessToken with custom TokenURL: %v", err)
	}
	if refreshed.AccessToken != "refreshed-token" {
		t.Errorf("AccessToken = %q, want refreshed-token", refreshed.AccessToken)
	}
	if refreshed.RefreshToken != "rtoken" {
		t.Errorf("RefreshToken should carry over original, got %q", refreshed.RefreshToken)
	}
}

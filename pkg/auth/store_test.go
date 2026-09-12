package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func setTestAuthHome(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	t.Setenv("KHUNQUANT_HOME", filepath.Join(tmpDir, ".khunquant"))
	return tmpDir
}

func TestAuthCredentialIsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{"zero time", time.Time{}, false},
		{"future", time.Now().Add(time.Hour), false},
		{"past", time.Now().Add(-time.Hour), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &AuthCredential{ExpiresAt: tt.expiresAt}
			if got := c.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthCredentialNeedsRefresh(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{"zero time", time.Time{}, false},
		{"far future", time.Now().Add(time.Hour), false},
		{"within 5 min", time.Now().Add(3 * time.Minute), true},
		{"already expired", time.Now().Add(-time.Minute), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &AuthCredential{ExpiresAt: tt.expiresAt}
			if got := c.NeedsRefresh(); got != tt.want {
				t.Errorf("NeedsRefresh() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStoreRoundtrip(t *testing.T) {
	setTestAuthHome(t)

	cred := &AuthCredential{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		AccountID:    "acct-123",
		ExpiresAt:    time.Now().Add(time.Hour).Truncate(time.Second),
		Provider:     "openai",
		AuthMethod:   "oauth",
	}

	if err := SetCredential("openai", cred); err != nil {
		t.Fatalf("SetCredential() error: %v", err)
	}

	loaded, err := GetCredential("openai")
	if err != nil {
		t.Fatalf("GetCredential() error: %v", err)
	}
	if loaded == nil {
		t.Fatal("GetCredential() returned nil")
	}
	if loaded.AccessToken != cred.AccessToken {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, cred.AccessToken)
	}
	if loaded.RefreshToken != cred.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", loaded.RefreshToken, cred.RefreshToken)
	}
	if loaded.Provider != cred.Provider {
		t.Errorf("Provider = %q, want %q", loaded.Provider, cred.Provider)
	}
}

func TestStoreFilePermissions(t *testing.T) {
	tmpDir := setTestAuthHome(t)

	cred := &AuthCredential{
		AccessToken: "secret-token",
		Provider:    "openai",
		AuthMethod:  "oauth",
	}
	if err := SetCredential("openai", cred); err != nil {
		t.Fatalf("SetCredential() error: %v", err)
	}

	path := filepath.Join(tmpDir, ".khunquant", "auth.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error: %v", err)
	}
	perm := info.Mode().Perm()
	if runtime.GOOS == "windows" {
		return
	}
	if perm != 0o600 {
		t.Errorf("file permissions = %o, want 0600", perm)
	}
}

func TestStoreMultiProvider(t *testing.T) {
	setTestAuthHome(t)

	openaiCred := &AuthCredential{AccessToken: "openai-token", Provider: "openai", AuthMethod: "oauth"}
	anthropicCred := &AuthCredential{AccessToken: "anthropic-token", Provider: "anthropic", AuthMethod: "token"}

	if err := SetCredential("openai", openaiCred); err != nil {
		t.Fatalf("SetCredential(openai) error: %v", err)
	}
	if err := SetCredential("anthropic", anthropicCred); err != nil {
		t.Fatalf("SetCredential(anthropic) error: %v", err)
	}

	loaded, err := GetCredential("openai")
	if err != nil {
		t.Fatalf("GetCredential(openai) error: %v", err)
	}
	if loaded.AccessToken != "openai-token" {
		t.Errorf("openai token = %q, want %q", loaded.AccessToken, "openai-token")
	}

	loaded, err = GetCredential("anthropic")
	if err != nil {
		t.Fatalf("GetCredential(anthropic) error: %v", err)
	}
	if loaded.AccessToken != "anthropic-token" {
		t.Errorf("anthropic token = %q, want %q", loaded.AccessToken, "anthropic-token")
	}
}

func TestDeleteCredential(t *testing.T) {
	setTestAuthHome(t)

	cred := &AuthCredential{AccessToken: "to-delete", Provider: "openai", AuthMethod: "oauth"}
	if err := SetCredential("openai", cred); err != nil {
		t.Fatalf("SetCredential() error: %v", err)
	}

	if err := DeleteCredential("openai"); err != nil {
		t.Fatalf("DeleteCredential() error: %v", err)
	}

	loaded, err := GetCredential("openai")
	if err != nil {
		t.Fatalf("GetCredential() error: %v", err)
	}
	if loaded != nil {
		t.Error("expected nil after delete")
	}
}

func TestLoadStoreEmpty(t *testing.T) {
	setTestAuthHome(t)

	store, err := LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() error: %v", err)
	}
	if store == nil {
		t.Fatal("LoadStore() returned nil")
	}
	if len(store.Credentials) != 0 {
		t.Errorf("expected empty credentials, got %d", len(store.Credentials))
	}
}

func TestGetCredentialCanonicalizesLegacyAntigravityProvider(t *testing.T) {
	tmpDir := setTestAuthHome(t)

	expiresAt := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	store := map[string]any{
		"credentials": map[string]any{
			"antigravity": map[string]any{
				"access_token": "legacy-token",
				"expires_at":   expiresAt.Format(time.RFC3339),
				"provider":     "antigravity",
				"auth_method":  "oauth",
				"project_id":   "project-1",
			},
		},
	}
	data, err := json.Marshal(store)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}
	path := filepath.Join(tmpDir, ".khunquant", "auth.json")
	err = os.MkdirAll(filepath.Dir(path), 0o755)
	if err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	err = os.WriteFile(path, data, 0o600)
	if err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	cred, err := GetCredential("google-antigravity")
	if err != nil {
		t.Fatalf("GetCredential() error: %v", err)
	}
	if cred == nil {
		t.Fatal("GetCredential() returned nil")
	}
	if cred.Provider != "google-antigravity" {
		t.Fatalf("Provider = %q, want %q", cred.Provider, "google-antigravity")
	}
	if !cred.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("ExpiresAt = %v, want %v", cred.ExpiresAt, expiresAt)
	}
}

func TestLoadStoreMergesAntigravityAliasesPreferringNewerExpiry(t *testing.T) {
	tmpDir := setTestAuthHome(t)

	legacyExpiry := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	refreshedExpiry := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	store := map[string]any{
		"credentials": map[string]any{
			"antigravity": map[string]any{
				"access_token":  "legacy-token",
				"refresh_token": "legacy-refresh",
				"expires_at":    legacyExpiry.Format(time.RFC3339),
				"provider":      "antigravity",
				"auth_method":   "oauth",
				"email":         "legacy@example.com",
			},
			"google-antigravity": map[string]any{
				"access_token": "fresh-token",
				"expires_at":   refreshedExpiry.Format(time.RFC3339),
				"provider":     "google-antigravity",
				"auth_method":  "oauth",
				"project_id":   "project-2",
			},
		},
	}
	data, err := json.Marshal(store)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}
	path := filepath.Join(tmpDir, ".khunquant", "auth.json")
	err = os.MkdirAll(filepath.Dir(path), 0o755)
	if err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	err = os.WriteFile(path, data, 0o600)
	if err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	loaded, err := LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() error: %v", err)
	}
	if len(loaded.Credentials) != 1 {
		t.Fatalf("credential count = %d, want 1", len(loaded.Credentials))
	}

	cred := loaded.Credentials["google-antigravity"]
	if cred == nil {
		t.Fatal("google-antigravity credential missing")
	}
	if cred.AccessToken != "fresh-token" {
		t.Fatalf("AccessToken = %q, want %q", cred.AccessToken, "fresh-token")
	}
	if cred.RefreshToken != "legacy-refresh" {
		t.Fatalf("RefreshToken = %q, want %q", cred.RefreshToken, "legacy-refresh")
	}
	if cred.Email != "legacy@example.com" {
		t.Fatalf("Email = %q, want %q", cred.Email, "legacy@example.com")
	}
	if cred.ProjectID != "project-2" {
		t.Fatalf("ProjectID = %q, want %q", cred.ProjectID, "project-2")
	}
	if !cred.ExpiresAt.Equal(refreshedExpiry) {
		t.Fatalf("ExpiresAt = %v, want %v", cred.ExpiresAt, refreshedExpiry)
	}
}

func TestLoadStorePrefersCanonicalKeyWhenExpiryMatchesAlias(t *testing.T) {
	tmpDir := setTestAuthHome(t)

	expiresAt := time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)
	store := map[string]any{
		"credentials": map[string]any{
			"antigravity": map[string]any{
				"access_token":  "legacy-token",
				"refresh_token": "legacy-refresh",
				"expires_at":    expiresAt.Format(time.RFC3339),
				"provider":      "antigravity",
				"auth_method":   "oauth",
				"email":         "legacy@example.com",
			},
			" Google-Antigravity ": map[string]any{
				"access_token": "fresh-token",
				"expires_at":   expiresAt.Format(time.RFC3339),
				"provider":     " Google-Antigravity ",
				"auth_method":  "oauth",
				"project_id":   "project-2",
			},
		},
	}
	data, err := json.Marshal(store)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}
	path := filepath.Join(tmpDir, ".khunquant", "auth.json")
	err = os.MkdirAll(filepath.Dir(path), 0o755)
	if err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	err = os.WriteFile(path, data, 0o600)
	if err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	loaded, err := LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() error: %v", err)
	}
	if len(loaded.Credentials) != 1 {
		t.Fatalf("credential count = %d, want 1", len(loaded.Credentials))
	}

	cred := loaded.Credentials["google-antigravity"]
	if cred == nil {
		t.Fatal("google-antigravity credential missing")
	}
	if cred.AccessToken != "fresh-token" {
		t.Fatalf("AccessToken = %q, want %q", cred.AccessToken, "fresh-token")
	}
	if cred.RefreshToken != "legacy-refresh" {
		t.Fatalf("RefreshToken = %q, want %q", cred.RefreshToken, "legacy-refresh")
	}
	if cred.Email != "legacy@example.com" {
		t.Fatalf("Email = %q, want %q", cred.Email, "legacy@example.com")
	}
	if cred.ProjectID != "project-2" {
		t.Fatalf("ProjectID = %q, want %q", cred.ProjectID, "project-2")
	}
}

func TestSetCredentialReplacesLegacyAntigravityEntry(t *testing.T) {
	tmpDir := setTestAuthHome(t)

	legacyStore := map[string]any{
		"credentials": map[string]any{
			"antigravity": map[string]any{
				"access_token": "legacy-token",
				"expires_at":   time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC).Format(time.RFC3339),
				"provider":     "antigravity",
				"auth_method":  "oauth",
			},
		},
	}
	data, err := json.Marshal(legacyStore)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}
	path := filepath.Join(tmpDir, ".khunquant", "auth.json")
	err = os.MkdirAll(filepath.Dir(path), 0o755)
	if err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	err = os.WriteFile(path, data, 0o600)
	if err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	refreshedExpiry := time.Date(2026, 4, 16, 12, 30, 0, 0, time.UTC)
	err = SetCredential("google-antigravity", &AuthCredential{
		AccessToken: "fresh-token",
		ExpiresAt:   refreshedExpiry,
		Provider:    "google-antigravity",
		AuthMethod:  "oauth",
	})
	if err != nil {
		t.Fatalf("SetCredential() error: %v", err)
	}

	loaded, err := LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() error: %v", err)
	}
	if len(loaded.Credentials) != 1 {
		t.Fatalf("credential count = %d, want 1", len(loaded.Credentials))
	}

	cred := loaded.Credentials["google-antigravity"]
	if cred == nil {
		t.Fatal("google-antigravity credential missing")
	}
	if cred.AccessToken != "fresh-token" {
		t.Fatalf("AccessToken = %q, want %q", cred.AccessToken, "fresh-token")
	}
	if !cred.ExpiresAt.Equal(refreshedExpiry) {
		t.Fatalf("ExpiresAt = %v, want %v", cred.ExpiresAt, refreshedExpiry)
	}
}

func TestDeleteCredentialRemovesLegacyAntigravityAlias(t *testing.T) {
	tmpDir := setTestAuthHome(t)

	legacyStore := map[string]any{
		"credentials": map[string]any{
			"antigravity": map[string]any{
				"access_token": "legacy-token",
				"provider":     "antigravity",
				"auth_method":  "oauth",
			},
		},
	}
	data, err := json.Marshal(legacyStore)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}
	path := filepath.Join(tmpDir, ".khunquant", "auth.json")
	err = os.MkdirAll(filepath.Dir(path), 0o755)
	if err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	err = os.WriteFile(path, data, 0o600)
	if err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	err = DeleteCredential(" google-antigravity ")
	if err != nil {
		t.Fatalf("DeleteCredential() error: %v", err)
	}

	loaded, err := LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() error: %v", err)
	}
	if len(loaded.Credentials) != 0 {
		t.Fatalf("credential count = %d, want 0", len(loaded.Credentials))
	}
}

func TestSetCredentialCanonicalizesTrimmedMixedCaseProvider(t *testing.T) {
	setTestAuthHome(t)

	expiresAt := time.Date(2026, 4, 16, 13, 0, 0, 0, time.UTC)
	if err := SetCredential("  AnTiGrAvItY  ", &AuthCredential{
		AccessToken: "fresh-token",
		ExpiresAt:   expiresAt,
		Provider:    "  AnTiGrAvItY  ",
		AuthMethod:  "oauth",
	}); err != nil {
		t.Fatalf("SetCredential() error: %v", err)
	}

	loaded, err := LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() error: %v", err)
	}
	if len(loaded.Credentials) != 1 {
		t.Fatalf("credential count = %d, want 1", len(loaded.Credentials))
	}

	cred := loaded.Credentials["google-antigravity"]
	if cred == nil {
		t.Fatal("google-antigravity credential missing")
	}
	if cred.Provider != "google-antigravity" {
		t.Fatalf("Provider = %q, want %q", cred.Provider, "google-antigravity")
	}
	if !cred.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("ExpiresAt = %v, want %v", cred.ExpiresAt, expiresAt)
	}

	got, err := GetCredential("  GoOgLe-AnTiGrAvItY ")
	if err != nil {
		t.Fatalf("GetCredential() error: %v", err)
	}
	if got == nil {
		t.Fatal("GetCredential() returned nil")
	}
	if got.Provider != "google-antigravity" {
		t.Fatalf("GetCredential provider = %q, want %q", got.Provider, "google-antigravity")
	}
}

// TestDeleteAllCredentials tests the DeleteAllCredentials function
func TestDeleteAllCredentials(t *testing.T) {
	tmpDir := setTestAuthHome(t)

	cred := &AuthCredential{AccessToken: "to-delete", Provider: "openai", AuthMethod: "oauth"}
	if err := SetCredential("openai", cred); err != nil {
		t.Fatalf("SetCredential() error: %v", err)
	}

	// Verify file exists
	path := filepath.Join(tmpDir, ".khunquant", "auth.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("auth.json should exist before DeleteAllCredentials")
	}

	// Delete all credentials
	if err := DeleteAllCredentials(); err != nil {
		t.Fatalf("DeleteAllCredentials() error: %v", err)
	}

	// Verify file no longer exists
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("auth.json should not exist after DeleteAllCredentials")
	}

	// Verify store is empty after reload
	store, err := LoadStore()
	if err != nil {
		t.Fatalf("LoadStore() error: %v", err)
	}
	if len(store.Credentials) != 0 {
		t.Errorf("expected empty credentials after DeleteAllCredentials, got %d", len(store.Credentials))
	}
}

// TestDeleteAllCredentials_NoFile tests DeleteAllCredentials when no file exists
func TestDeleteAllCredentials_NoFile(t *testing.T) {
	setTestAuthHome(t)

	// Should not error if file doesn't exist
	if err := DeleteAllCredentials(); err != nil {
		t.Fatalf("DeleteAllCredentials() should not error when file missing: %v", err)
	}
}

// TestAuthFilePath tests authFilePath with and without KHUNQUANT_HOME
func TestAuthFilePath(t *testing.T) {
	t.Run("with KHUNQUANT_HOME", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("KHUNQUANT_HOME", tmpDir)

		path := authFilePath()
		expected := filepath.Join(tmpDir, "auth.json")
		if path != expected {
			t.Errorf("authFilePath() = %q, want %q", path, expected)
		}
	})

	t.Run("without KHUNQUANT_HOME", func(t *testing.T) {
		t.Setenv("KHUNQUANT_HOME", "")

		path := authFilePath()
		if !strings.Contains(path, ".khunquant") || !strings.HasSuffix(path, "auth.json") {
			t.Errorf("authFilePath() = %q, should contain .khunquant and end with auth.json", path)
		}
	})
}

// TestShouldPreferCredential tests the shouldPreferCredential function
func TestShouldPreferCredential(t *testing.T) {
	tests := []struct {
		name              string
		candidate         *AuthCredential
		candidateCanonical bool
		current           *AuthCredential
		currentCanonical  bool
		want              bool
	}{
		{
			name:              "candidate nil",
			candidate:         nil,
			candidateCanonical: true,
			current:           &AuthCredential{},
			currentCanonical:  false,
			want:              false,
		},
		{
			name:              "current nil",
			candidate:         &AuthCredential{},
			candidateCanonical: false,
			current:           nil,
			currentCanonical:  false,
			want:              true,
		},
		{
			name:              "candidate newer expiry",
			candidate:         &AuthCredential{ExpiresAt: time.Now().Add(2 * time.Hour)},
			candidateCanonical: false,
			current:           &AuthCredential{ExpiresAt: time.Now().Add(1 * time.Hour)},
			currentCanonical:  false,
			want:              true,
		},
		{
			name:              "current newer expiry",
			candidate:         &AuthCredential{ExpiresAt: time.Now().Add(1 * time.Hour)},
			candidateCanonical: false,
			current:           &AuthCredential{ExpiresAt: time.Now().Add(2 * time.Hour)},
			currentCanonical:  false,
			want:              false,
		},
		{
			name:              "same expiry, candidate canonical",
			candidate:         &AuthCredential{ExpiresAt: time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)},
			candidateCanonical: true,
			current:           &AuthCredential{ExpiresAt: time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)},
			currentCanonical:  false,
			want:              true,
		},
		{
			name:              "same expiry, current canonical",
			candidate:         &AuthCredential{ExpiresAt: time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)},
			candidateCanonical: false,
			current:           &AuthCredential{ExpiresAt: time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)},
			currentCanonical:  true,
			want:              false,
		},
		{
			name:              "both canonical and same expiry",
			candidate:         &AuthCredential{ExpiresAt: time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)},
			candidateCanonical: true,
			current:           &AuthCredential{ExpiresAt: time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)},
			currentCanonical:  true,
			want:              false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldPreferCredential(tt.candidate, tt.candidateCanonical, tt.current, tt.currentCanonical)
			if got != tt.want {
				t.Errorf("shouldPreferCredential() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCloneCredential_Nil(t *testing.T) {
	if cloneCredential(nil) != nil {
		t.Error("cloneCredential(nil) should return nil")
	}
}

func TestCloneCredential_NonNil(t *testing.T) {
	cred := &AuthCredential{Provider: "openai", AccessToken: "tok"}
	got := cloneCredential(cred)
	if got == cred {
		t.Error("cloneCredential should return a copy, not the same pointer")
	}
	if got.Provider != "openai" || got.AccessToken != "tok" {
		t.Errorf("cloneCredential = %+v, want same values", got)
	}
}

func TestMergeCredentials_BothNil(t *testing.T) {
	if mergeCredentials(nil, nil) != nil {
		t.Error("mergeCredentials(nil, nil) should return nil")
	}
}

func TestMergeCredentials_PrimaryNil(t *testing.T) {
	secondary := &AuthCredential{Provider: "openai", AccessToken: "secondary-tok"}
	got := mergeCredentials(nil, secondary)
	if got == nil {
		t.Fatal("mergeCredentials(nil, secondary) should not be nil")
	}
	if got.AccessToken != "secondary-tok" {
		t.Errorf("mergeCredentials nil primary: AccessToken = %q, want secondary-tok", got.AccessToken)
	}
}

func TestMergeCredentials_SecondaryNil(t *testing.T) {
	primary := &AuthCredential{Provider: "openai", AccessToken: "primary-tok"}
	got := mergeCredentials(primary, nil)
	if got == nil {
		t.Fatal("mergeCredentials(primary, nil) should not be nil")
	}
	if got.AccessToken != "primary-tok" {
		t.Errorf("mergeCredentials nil secondary: AccessToken = %q, want primary-tok", got.AccessToken)
	}
}

func TestMergeCredentials_FillsFromSecondary(t *testing.T) {
	primary := &AuthCredential{Provider: "openai", AccessToken: "primary-tok"}
	secondary := &AuthCredential{Provider: "openai", RefreshToken: "refresh-tok", AccountID: "acct"}
	got := mergeCredentials(primary, secondary)
	if got.AccessToken != "primary-tok" {
		t.Errorf("mergeCredentials should keep primary AccessToken, got %q", got.AccessToken)
	}
	if got.RefreshToken != "refresh-tok" {
		t.Errorf("mergeCredentials should fill empty RefreshToken from secondary, got %q", got.RefreshToken)
	}
	if got.AccountID != "acct" {
		t.Errorf("mergeCredentials should fill empty AccountID from secondary, got %q", got.AccountID)
	}
}

func TestNormalizeStore_Nil(t *testing.T) {
	normalizeStore(nil) // should not panic
}

func TestNormalizeStore_NilCredentials(t *testing.T) {
	store := &AuthStore{Credentials: nil}
	normalizeStore(store)
	if store.Credentials == nil {
		t.Error("normalizeStore should initialize nil Credentials to empty map")
	}
}

func TestNormalizeStore_CanonicalizesKeys(t *testing.T) {
	store := &AuthStore{
		Credentials: map[string]*AuthCredential{
			"OpenAI": {Provider: "OpenAI", AccessToken: "tok"},
		},
	}
	normalizeStore(store)
	// canonical should be lowercase
	var found bool
	for k := range store.Credentials {
		if k == "openai" {
			found = true
		}
	}
	if !found {
		t.Errorf("normalizeStore should canonicalize key to lowercase, got keys: %v", store.Credentials)
	}
}

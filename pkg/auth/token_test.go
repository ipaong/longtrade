package auth

import (
	"strings"
	"testing"
)

func TestLoginSetupToken(t *testing.T) {
	// A valid token: correct prefix + at least 80 chars
	validToken := "sk-ant-oat01-" + strings.Repeat("a", 80)

	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{"valid token", validToken, ""},
		{"empty input", "", "expected prefix sk-ant-oat01-"},
		{"wrong prefix", "sk-ant-api-" + strings.Repeat("a", 80), "expected prefix sk-ant-oat01-"},
		{"too short", "sk-ant-oat01-short", "too short"},
		{"whitespace only", "   ", "expected prefix sk-ant-oat01-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input + "\n")
			cred, err := LoginSetupToken(r)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cred.AccessToken != validToken {
				t.Errorf("AccessToken = %q, want %q", cred.AccessToken, validToken)
			}
			if cred.Provider != "anthropic" {
				t.Errorf("Provider = %q, want %q", cred.Provider, "anthropic")
			}
			if cred.AuthMethod != "oauth" {
				t.Errorf("AuthMethod = %q, want %q", cred.AuthMethod, "oauth")
			}
		})
	}
}

func TestLoginSetupToken_EmptyReader(t *testing.T) {
	r := strings.NewReader("")
	_, err := LoginSetupToken(r)
	if err == nil {
		t.Fatal("expected error for empty reader, got nil")
	}
}

// TestLoginPasteToken tests the LoginPasteToken function
func TestLoginPasteToken_Valid(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		input    string
		wantErr  bool
	}{
		{
			name:     "anthropic provider",
			provider: "anthropic",
			input:    "sk-ant-valid-token-123",
			wantErr:  false,
		},
		{
			name:     "openai provider",
			provider: "openai",
			input:    "sk-proj-valid-token-456",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input + "\n")
			cred, err := LoginPasteToken(tt.provider, r)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cred.AccessToken != tt.input {
				t.Errorf("AccessToken = %q, want %q", cred.AccessToken, tt.input)
			}
			if cred.Provider != tt.provider {
				t.Errorf("Provider = %q, want %q", cred.Provider, tt.provider)
			}
			if cred.AuthMethod != "token" {
				t.Errorf("AuthMethod = %q, want %q", cred.AuthMethod, "token")
			}
		})
	}
}

// TestLoginPasteToken_EmptyToken tests LoginPasteToken with empty input
func TestLoginPasteToken_EmptyToken(t *testing.T) {
	r := strings.NewReader("\n")
	_, err := LoginPasteToken("anthropic", r)
	if err == nil {
		t.Fatal("expected error for empty token, got nil")
	}
	if !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("expected 'cannot be empty' error, got: %v", err)
	}
}

// TestLoginPasteToken_NoInput tests LoginPasteToken with EOF
func TestLoginPasteToken_NoInput(t *testing.T) {
	r := strings.NewReader("")
	_, err := LoginPasteToken("anthropic", r)
	if err == nil {
		t.Fatal("expected error for no input, got nil")
	}
	if !strings.Contains(err.Error(), "no input received") {
		t.Errorf("expected 'no input received' error, got: %v", err)
	}
}

// TestProviderDisplayName tests the providerDisplayName function
func TestProviderDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		want     string
	}{
		{
			name:     "anthropic",
			provider: "anthropic",
			want:     "console.anthropic.com",
		},
		{
			name:     "openai",
			provider: "openai",
			want:     "platform.openai.com",
		},
		{
			name:     "unknown provider",
			provider: "unknown",
			want:     "unknown",
		},
		{
			name:     "custom provider",
			provider: "custom-api",
			want:     "custom-api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := providerDisplayName(tt.provider)
			if got != tt.want {
				t.Errorf("providerDisplayName(%q) = %q, want %q", tt.provider, got, tt.want)
			}
		})
	}
}

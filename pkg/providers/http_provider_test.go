package providers

import "testing"

func TestNewHTTPProvider_NotNil(t *testing.T) {
	p := NewHTTPProvider("key", "http://localhost:8080", "")
	if p == nil {
		t.Fatal("NewHTTPProvider returned nil")
	}
}

func TestNewHTTPProviderWithMaxTokensField_NotNil(t *testing.T) {
	p := NewHTTPProviderWithMaxTokensField("key", "http://localhost:8080", "", "max_completion_tokens")
	if p == nil {
		t.Fatal("NewHTTPProviderWithMaxTokensField returned nil")
	}
}

func TestHTTPProvider_GetDefaultModel_Empty(t *testing.T) {
	p := NewHTTPProvider("key", "http://localhost:8080", "")
	if got := p.GetDefaultModel(); got != "" {
		t.Errorf("GetDefaultModel() = %q, want empty string", got)
	}
}

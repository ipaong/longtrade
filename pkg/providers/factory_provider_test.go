// KhunQuant - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 KhunQuant contributors

package providers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cryptoquantumwave/khunquant/pkg/config"
)

func TestExtractProtocol(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		wantProtocol string
		wantModelID  string
	}{
		{
			name:         "openai with prefix",
			model:        "openai/gpt-4o",
			wantProtocol: "openai",
			wantModelID:  "gpt-4o",
		},
		{
			name:         "anthropic with prefix",
			model:        "anthropic/claude-sonnet-4.6",
			wantProtocol: "anthropic",
			wantModelID:  "claude-sonnet-4.6",
		},
		{
			name:         "no prefix - defaults to openai",
			model:        "gpt-4o",
			wantProtocol: "openai",
			wantModelID:  "gpt-4o",
		},
		{
			name:         "groq with prefix",
			model:        "groq/llama-3.1-70b",
			wantProtocol: "groq",
			wantModelID:  "llama-3.1-70b",
		},
		{
			name:         "empty string",
			model:        "",
			wantProtocol: "openai",
			wantModelID:  "",
		},
		{
			name:         "with whitespace",
			model:        "  openai/gpt-4  ",
			wantProtocol: "openai",
			wantModelID:  "gpt-4",
		},
		{
			name:         "multiple slashes",
			model:        "nvidia/meta/llama-3.1-8b",
			wantProtocol: "nvidia",
			wantModelID:  "meta/llama-3.1-8b",
		},
		{
			name:         "azure with prefix",
			model:        "azure/my-gpt5-deployment",
			wantProtocol: "azure",
			wantModelID:  "my-gpt5-deployment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			protocol, modelID := ExtractProtocol(tt.model)
			if protocol != tt.wantProtocol {
				t.Errorf("ExtractProtocol(%q) protocol = %q, want %q", tt.model, protocol, tt.wantProtocol)
			}
			if modelID != tt.wantModelID {
				t.Errorf("ExtractProtocol(%q) modelID = %q, want %q", tt.model, modelID, tt.wantModelID)
			}
		})
	}
}

func TestCreateProviderFromConfig_OpenAI(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-openai",
		Model:     "openai/gpt-4o",
		APIKey:    *config.NewSecureString("test-key"),
		APIBase:   "https://api.example.com/v1",
	}

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "gpt-4o" {
		t.Errorf("modelID = %q, want %q", modelID, "gpt-4o")
	}
}

func TestCreateProviderFromConfig_DefaultAPIBase(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
	}{
		{"openai", "openai"},
		{"groq", "groq"},
		{"openrouter", "openrouter"},
		{"cerebras", "cerebras"},
		{"vivgrid", "vivgrid"},
		{"qwen", "qwen"},
		{"vllm", "vllm"},
		{"deepseek", "deepseek"},
		{"ollama", "ollama"},
		{"llamacpp", "llamacpp"},
		{"longcat", "longcat"},
		{"modelscope", "modelscope"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.ModelConfig{
				ModelName: "test-" + tt.protocol,
				Model:     tt.protocol + "/test-model",
				APIKey:    *config.NewSecureString("test-key"),
			}

			provider, _, err := CreateProviderFromConfig(cfg)
			if err != nil {
				t.Fatalf("CreateProviderFromConfig() error = %v", err)
			}

			// Verify we got an HTTPProvider for all these protocols
			if _, ok := provider.(*HTTPProvider); !ok {
				t.Fatalf("expected *HTTPProvider, got %T", provider)
			}
		})
	}
}

func TestGetDefaultAPIBase_LiteLLM(t *testing.T) {
	if got := getDefaultAPIBase("litellm"); got != "http://localhost:4000/v1" {
		t.Fatalf("getDefaultAPIBase(%q) = %q, want %q", "litellm", got, "http://localhost:4000/v1")
	}
}

func TestCreateProviderFromConfig_LiteLLM(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-litellm",
		Model:     "litellm/my-proxy-alias",
		APIKey:    *config.NewSecureString("test-key"),
		APIBase:   "http://localhost:4000/v1",
	}

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "my-proxy-alias" {
		t.Errorf("modelID = %q, want %q", modelID, "my-proxy-alias")
	}
}

func TestCreateProviderFromConfig_LongCat(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-longcat",
		Model:     "longcat/LongCat-Flash-Thinking",
		APIKey:    *config.NewSecureString("test-key"),
		APIBase:   "https://api.longcat.chat/openai",
	}

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "LongCat-Flash-Thinking" {
		t.Errorf("modelID = %q, want %q", modelID, "LongCat-Flash-Thinking")
	}
	if _, ok := provider.(*HTTPProvider); !ok {
		t.Fatalf("expected *HTTPProvider, got %T", provider)
	}
}

func TestCreateProviderFromConfig_ModelScope(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-modelscope",
		Model:     "modelscope/Qwen/Qwen3-235B-A22B-Instruct-2507",
		APIKey:    *config.NewSecureString("test-key"),
		APIBase:   "https://api-inference.modelscope.cn/v1",
	}

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "Qwen/Qwen3-235B-A22B-Instruct-2507" {
		t.Errorf("modelID = %q, want %q", modelID, "Qwen/Qwen3-235B-A22B-Instruct-2507")
	}
	if _, ok := provider.(*HTTPProvider); !ok {
		t.Fatalf("expected *HTTPProvider, got %T", provider)
	}
}

func TestGetDefaultAPIBase_ModelScope(t *testing.T) {
	if got := getDefaultAPIBase("modelscope"); got != "https://api-inference.modelscope.cn/v1" {
		t.Fatalf("getDefaultAPIBase(%q) = %q, want %q", "modelscope", got, "https://api-inference.modelscope.cn/v1")
	}
}

func TestCreateProviderFromConfig_Anthropic(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-anthropic",
		Model:     "anthropic/claude-sonnet-4.6",
		APIKey:    *config.NewSecureString("test-key"),
	}

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "claude-sonnet-4.6" {
		t.Errorf("modelID = %q, want %q", modelID, "claude-sonnet-4.6")
	}
}

func TestCreateProviderFromConfig_Antigravity(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-antigravity",
		Model:     "antigravity/gemini-2.0-flash",
	}

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "gemini-2.0-flash" {
		t.Errorf("modelID = %q, want %q", modelID, "gemini-2.0-flash")
	}
}

func TestCreateProviderFromConfig_ClaudeCLI(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-claude-cli",
		Model:     "claude-cli/claude-sonnet-4.6",
	}

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "claude-sonnet-4.6" {
		t.Errorf("modelID = %q, want %q", modelID, "claude-sonnet-4.6")
	}
}

func TestCreateProviderFromConfig_CodexCLI(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-codex-cli",
		Model:     "codex-cli/codex",
	}

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "codex" {
		t.Errorf("modelID = %q, want %q", modelID, "codex")
	}
}

func TestCreateProviderFromConfig_MissingAPIKey(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-no-key",
		Model:     "openai/gpt-4o",
	}

	_, _, err := CreateProviderFromConfig(cfg)
	if err == nil {
		t.Fatal("CreateProviderFromConfig() expected error for missing API key")
	}
}

func TestCreateProviderFromConfig_UnknownProtocol(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-unknown",
		Model:     "unknown-protocol/model",
		APIKey:    *config.NewSecureString("test-key"),
	}

	_, _, err := CreateProviderFromConfig(cfg)
	if err == nil {
		t.Fatal("CreateProviderFromConfig() expected error for unknown protocol")
	}
}

func TestCreateProviderFromConfig_NilConfig(t *testing.T) {
	_, _, err := CreateProviderFromConfig(nil)
	if err == nil {
		t.Fatal("CreateProviderFromConfig(nil) expected error")
	}
}

func TestCreateProviderFromConfig_EmptyModel(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-empty",
		Model:     "",
	}

	_, _, err := CreateProviderFromConfig(cfg)
	if err == nil {
		t.Fatal("CreateProviderFromConfig() expected error for empty model")
	}
}

func TestCreateProviderFromConfig_RequestTimeoutPropagation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1500 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()

	cfg := &config.ModelConfig{
		ModelName:      "test-timeout",
		Model:          "openai/gpt-4o",
		APIBase:        server.URL,
		RequestTimeout: 1,
	}

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if modelID != "gpt-4o" {
		t.Fatalf("modelID = %q, want %q", modelID, "gpt-4o")
	}

	_, err = provider.Chat(
		t.Context(),
		[]Message{{Role: "user", Content: "hi"}},
		nil,
		modelID,
		nil,
	)
	if err == nil {
		t.Fatal("Chat() expected timeout error, got nil")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "context deadline exceeded") && !strings.Contains(errMsg, "Client.Timeout exceeded") {
		t.Fatalf("Chat() error = %q, want timeout-related error", errMsg)
	}
}

func TestCreateProviderFromConfig_Azure(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "azure-gpt5",
		Model:     "azure/my-gpt5-deployment",
		APIKey:    *config.NewSecureString("test-azure-key"),
		APIBase:   "https://my-resource.openai.azure.com",
	}

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "my-gpt5-deployment" {
		t.Errorf("modelID = %q, want %q", modelID, "my-gpt5-deployment")
	}
}

func TestCreateProviderFromConfig_AzureOpenAIAlias(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "azure-gpt4",
		Model:     "azure-openai/my-deployment",
		APIKey:    *config.NewSecureString("test-azure-key"),
		APIBase:   "https://my-resource.openai.azure.com",
	}

	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "my-deployment" {
		t.Errorf("modelID = %q, want %q", modelID, "my-deployment")
	}
}

func TestCreateProviderFromConfig_AzureMissingAPIKey(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "azure-gpt5",
		Model:     "azure/my-gpt5-deployment",
		APIBase:   "https://my-resource.openai.azure.com",
	}

	_, _, err := CreateProviderFromConfig(cfg)
	if err == nil {
		t.Fatal("CreateProviderFromConfig() expected error for missing API key")
	}
}

func TestCreateProviderFromConfig_AzureMissingAPIBase(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "azure-gpt5",
		Model:     "azure/my-gpt5-deployment",
		APIKey:    *config.NewSecureString("test-azure-key"),
	}

	_, _, err := CreateProviderFromConfig(cfg)
	if err == nil {
		t.Fatal("CreateProviderFromConfig() expected error for missing API base")
	}
}

func TestGetDefaultAPIBase_RemainingProtocols(t *testing.T) {
	cases := []struct {
		protocol string
		want     string
	}{
		{"zhipu", "https://open.bigmodel.cn/api/paas/v4"},
		{"gemini", "https://generativelanguage.googleapis.com/v1beta"},
		{"nvidia", "https://integrate.api.nvidia.com/v1"},
		{"moonshot", "https://api.moonshot.cn/v1"},
		{"shengsuanyun", "https://router.shengsuanyun.com/api/v1"},
		{"volcengine", "https://ark.cn-beijing.volces.com/api/v3"},
		{"mistral", "https://api.mistral.ai/v1"},
		{"avian", "https://api.avian.io/v1"},
		{"minimax", "https://api.minimaxi.com/v1"},
		{"mlx_lm", "http://localhost:8080/v1"},
		{"unknown-proto", ""},
	}
	for _, tc := range cases {
		got := getDefaultAPIBase(tc.protocol)
		if got != tc.want {
			t.Errorf("getDefaultAPIBase(%q) = %q, want %q", tc.protocol, got, tc.want)
		}
	}
}

func TestCreateProviderFromConfig_MLXlm(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-mlx",
		Model:     "mlx_lm/~/models/gemma",
		APIKey:    *config.NewSecureString(""),
		APIBase:   "http://localhost:8080/v1",
	}
	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "~/models/gemma" {
		t.Errorf("modelID = %q, want %q", modelID, "~/models/gemma")
	}
}

func TestCreateProviderFromConfig_AnthropicMessages(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-anthropic-msg",
		Model:     "anthropic-messages/claude-sonnet-4.6",
		APIKey:    *config.NewSecureString("test-key"),
	}
	provider, modelID, err := CreateProviderFromConfig(cfg)
	if err != nil {
		t.Fatalf("CreateProviderFromConfig() error = %v", err)
	}
	if provider == nil {
		t.Fatal("CreateProviderFromConfig() returned nil provider")
	}
	if modelID != "claude-sonnet-4.6" {
		t.Errorf("modelID = %q, want %q", modelID, "claude-sonnet-4.6")
	}
}

func TestCreateProviderFromConfig_AnthropicMessagesMissingKey(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName: "test-anthropic-msg-nokey",
		Model:     "anthropic-messages/claude-sonnet-4.6",
	}
	_, _, err := CreateProviderFromConfig(cfg)
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestCreateProviderFromConfig_OpenAIOAuth(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName:  "test-openai-oauth",
		Model:      "openai/gpt-4o",
		AuthMethod: "oauth",
	}
	// createCodexAuthProvider will fail (no credentials on disk), but
	// the codepath before the auth call must be exercised without panic.
	_, _, err := CreateProviderFromConfig(cfg)
	// expected error: "loading auth credentials: ..."
	if err == nil {
		t.Log("createCodexAuthProvider succeeded unexpectedly (credentials present)")
	}
}

func TestCreateProviderFromConfig_AnthropicOAuth(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName:  "test-anthropic-oauth",
		Model:      "anthropic/claude-sonnet-4.6",
		AuthMethod: "token",
	}
	// createClaudeAuthProvider will fail (no credentials on disk).
	_, _, err := CreateProviderFromConfig(cfg)
	if err == nil {
		t.Log("createClaudeAuthProvider succeeded unexpectedly (credentials present)")
	}
}

func TestCreateProviderFromConfig_GitHubCopilot(t *testing.T) {
	cfg := &config.ModelConfig{
		ModelName:   "test-copilot",
		Model:       "github-copilot/gpt-4o",
		APIBase:     "localhost:4321",
		ConnectMode: "grpc",
	}
	// NewGitHubCopilotProvider may fail if grpc dial fails, but the
	// code path to that call must be reached without panic.
	_, _, _ = CreateProviderFromConfig(cfg)
}

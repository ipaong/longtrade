package providers

import (
	"strings"
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/auth"
	"github.com/cryptoquantumwave/khunquant/pkg/config"
)

func TestResolveProviderSelection(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(*config.Config)
		wantType      providerType
		wantAPIBase   string
		wantProxy     string
		wantErrSubstr string
	}{
		{
			name: "explicit litellm provider uses configured base",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Provider = "litellm"
				cfg.Providers.LiteLLM.APIKey = "litellm-key"
				cfg.Providers.LiteLLM.APIBase = "http://localhost:4000/v1"
				cfg.Providers.LiteLLM.Proxy = "http://127.0.0.1:7890"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "http://localhost:4000/v1",
			wantProxy:   "http://127.0.0.1:7890",
		},
		{
			name: "explicit litellm provider defaults base when only key is configured",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Provider = "litellm"
				cfg.Providers.LiteLLM.APIKey = "litellm-key"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "http://localhost:4000/v1",
		},
		{
			name: "explicit claude-cli provider routes to cli provider type",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Provider = "claude-cli"
				cfg.Agents.Defaults.Workspace = "/tmp/ws"
			},
			wantType: providerTypeClaudeCLI,
		},
		{
			name: "explicit copilot provider routes to github copilot type",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Provider = "copilot"
			},
			wantType:    providerTypeGitHubCopilot,
			wantAPIBase: "localhost:4321",
		},
		{
			name: "explicit deepseek provider uses deepseek defaults",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Provider = "deepseek"
				cfg.Agents.Defaults.Model = "deepseek/deepseek-chat"
				cfg.Providers.DeepSeek.APIKey = "deepseek-key"
				cfg.Providers.DeepSeek.Proxy = "http://127.0.0.1:7890"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "https://api.deepseek.com/v1",
			wantProxy:   "http://127.0.0.1:7890",
		},
		{
			name: "explicit shengsuanyun provider uses defaults",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Provider = "shengsuanyun"
				cfg.Providers.ShengSuanYun.APIKey = "ssy-key"
				cfg.Providers.ShengSuanYun.Proxy = "http://127.0.0.1:7890"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "https://router.shengsuanyun.com/api/v1",
			wantProxy:   "http://127.0.0.1:7890",
		},
		{
			name: "explicit nvidia provider uses defaults",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Provider = "nvidia"
				cfg.Providers.Nvidia.APIKey = "nvapi-test"
				cfg.Providers.Nvidia.Proxy = "http://127.0.0.1:7890"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "https://integrate.api.nvidia.com/v1",
			wantProxy:   "http://127.0.0.1:7890",
		},
		{
			name: "explicit vivgrid provider uses defaults",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Provider = "vivgrid"
				cfg.Providers.Vivgrid.APIKey = "vivgrid-key"
				cfg.Providers.Vivgrid.Proxy = "http://127.0.0.1:7890"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "https://api.vivgrid.com/v1",
			wantProxy:   "http://127.0.0.1:7890",
		},
		{
			name: "openrouter model uses openrouter defaults",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Model = "openrouter/auto"
				cfg.Providers.OpenRouter.APIKey = "sk-or-test"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "https://openrouter.ai/api/v1",
		},
		{
			name: "anthropic oauth routes to claude auth provider",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Model = "claude-sonnet-4.6"
				cfg.Providers.Anthropic.AuthMethod = "oauth"
			},
			wantType: providerTypeClaudeAuth,
		},
		{
			name: "openai oauth routes to codex auth provider",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Model = "gpt-4o"
				cfg.Providers.OpenAI.AuthMethod = "oauth"
			},
			wantType: providerTypeCodexAuth,
		},
		{
			name: "openai codex-cli auth routes to codex cli token provider",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Model = "gpt-4o"
				cfg.Providers.OpenAI.AuthMethod = "codex-cli"
			},
			wantType: providerTypeCodexCLIToken,
		},
		{
			name: "explicit codex-code provider routes to codex cli provider type",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Provider = "codex-code"
				cfg.Agents.Defaults.Workspace = "/tmp/ws"
			},
			wantType: providerTypeCodexCLI,
		},
		{
			name: "zhipu model uses zhipu base default",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Model = "glm-4.7"
				cfg.Providers.Zhipu.APIKey = "zhipu-key"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "https://open.bigmodel.cn/api/paas/v4",
		},
		{
			name: "groq model uses groq base default",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Model = "groq/llama-3.3-70b"
				cfg.Providers.Groq.APIKey = "gsk-key"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "https://api.groq.com/openai/v1",
		},
		{
			name: "ollama model uses ollama base default",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Model = "ollama/qwen2.5:14b"
				cfg.Providers.Ollama.APIKey = "ollama-key"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "http://localhost:11434/v1",
		},
		{
			name: "moonshot model keeps proxy and default base",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Model = "moonshot/kimi-k2.5"
				cfg.Providers.Moonshot.APIKey = "moonshot-key"
				cfg.Providers.Moonshot.Proxy = "http://127.0.0.1:7890"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "https://api.moonshot.cn/v1",
			wantProxy:   "http://127.0.0.1:7890",
		},
		{
			name: "explicit longcat provider uses defaults",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Provider = "longcat"
				cfg.Providers.LongCat.APIKey = "longcat-key"
				cfg.Providers.LongCat.Proxy = "http://127.0.0.1:7890"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "https://api.longcat.chat/openai",
			wantProxy:   "http://127.0.0.1:7890",
		},
		{
			name: "longcat model fallback uses longcat base default",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Model = "longcat/LongCat-Flash-Thinking"
				cfg.Providers.LongCat.APIKey = "longcat-key"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "https://api.longcat.chat/openai",
		},
		{
			name: "missing keys returns model config error",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Model = "custom-model"
			},
			wantErrSubstr: "no API key configured for model",
		},
		{
			name: "openrouter prefix without key returns provider key error",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Model = "openrouter/auto"
			},
			wantErrSubstr: "no API key configured for provider",
		},
		{
			name: "explicit gemini provider uses gemini defaults",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Provider = "gemini"
				cfg.Providers.Gemini.APIKey = "gemini-key"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "https://generativelanguage.googleapis.com/v1beta",
		},
		{
			name: "explicit google alias routes to gemini defaults",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Provider = "google"
				cfg.Providers.Gemini.APIKey = "google-key"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "https://generativelanguage.googleapis.com/v1beta",
		},
		{
			name: "explicit vllm provider uses configured base",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Provider = "vllm"
				cfg.Providers.VLLM.APIBase = "http://localhost:8000/v1"
				cfg.Providers.VLLM.APIKey = "vllm-key"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "http://localhost:8000/v1",
		},
		{
			name: "explicit llamacpp provider uses configured base",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Provider = "llamacpp"
				cfg.Providers.LlamaCpp.APIBase = "http://localhost:8080/v1"
				cfg.Providers.LlamaCpp.APIKey = "none"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "http://localhost:8080/v1",
		},
		{
			name: "explicit mlx_lm provider uses configured base",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Provider = "mlx_lm"
				cfg.Providers.MLXLM.APIBase = "http://localhost:8080/v1"
				cfg.Providers.MLXLM.APIKey = "none"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "http://localhost:8080/v1",
		},
		{
			name: "explicit avian provider uses avian defaults",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Provider = "avian"
				cfg.Providers.Avian.APIKey = "avian-key"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "https://api.avian.io/v1",
		},
		{
			name: "explicit mistral provider uses mistral defaults",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Provider = "mistral"
				cfg.Providers.Mistral.APIKey = "mistral-key"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "https://api.mistral.ai/v1",
		},
		{
			name: "explicit minimax provider uses minimax defaults",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Provider = "minimax"
				cfg.Providers.Minimax.APIKey = "minimax-key"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "https://api.minimaxi.com/v1",
		},
		{
			name: "glm alias routes to zhipu provider",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Provider = "glm"
				cfg.Providers.Zhipu.APIKey = "zhipu-key"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "https://open.bigmodel.cn/api/paas/v4",
		},
		{
			name: "claude-code alias routes to claude-cli provider type",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Provider = "claude-code"
				cfg.Agents.Defaults.Workspace = "/tmp/ws"
			},
			wantType: providerTypeClaudeCLI,
		},
		{
			name: "gemini model inferred from model prefix",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Model = "gemini-1.5-pro"
				cfg.Providers.Gemini.APIKey = "gemini-key"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "https://generativelanguage.googleapis.com/v1beta",
		},
		{
			name: "mistral model inferred from model name",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Model = "mistral-large"
				cfg.Providers.Mistral.APIKey = "mistral-key"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "https://api.mistral.ai/v1",
		},
		{
			name: "minimax model inferred from model name",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Model = "minimax-text-01"
				cfg.Providers.Minimax.APIKey = "minimax-key"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "https://api.minimaxi.com/v1",
		},
		{
			name: "avian model prefix inferred",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Model = "avian/gpt-4o"
				cfg.Providers.Avian.APIKey = "avian-key"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "https://api.avian.io/v1",
		},
		{
			name: "nvidia model inferred from model name",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Model = "nvidia/llama-3.1-nemotron-70b"
				cfg.Providers.Nvidia.APIKey = "nvapi-test"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "https://integrate.api.nvidia.com/v1",
		},
		{
			name: "vivgrid model prefix inferred",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Model = "vivgrid/claude-3.5-sonnet"
				cfg.Providers.Vivgrid.APIKey = "vivgrid-key"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "https://api.vivgrid.com/v1",
		},
		{
			name: "vllm fallback when no explicit provider but base configured",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Model = "local-model"
				cfg.Providers.VLLM.APIBase = "http://localhost:8000/v1"
				cfg.Providers.VLLM.APIKey = "vllm-key"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "http://localhost:8000/v1",
		},
		{
			name: "llamacpp fallback when no explicit provider but base configured",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Model = "local-model"
				cfg.Providers.LlamaCpp.APIBase = "http://localhost:8080/v1"
				cfg.Providers.LlamaCpp.APIKey = "none"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "http://localhost:8080/v1",
		},
		{
			name: "mlx_lm fallback when no explicit provider but base configured",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Model = "local-model"
				cfg.Providers.MLXLM.APIBase = "http://localhost:8080/v1"
				cfg.Providers.MLXLM.APIKey = "none"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "http://localhost:8080/v1",
		},
		{
			name: "no model no provider returns error",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Model = ""
				cfg.Agents.Defaults.Provider = ""
			},
			wantErrSubstr: "no model configured",
		},
		{
			name: "openai explicit provider routes to openai defaults",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Provider = "openai"
				cfg.Providers.OpenAI.APIKey = "sk-test"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "https://api.openai.com/v1",
		},
		{
			name: "gpt alias routes to openai defaults",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Provider = "gpt"
				cfg.Providers.OpenAI.APIKey = "sk-test"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "https://api.openai.com/v1",
		},
		{
			name: "anthropic explicit provider with api key",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Provider = "anthropic"
				cfg.Providers.Anthropic.APIKey = "sk-ant-test"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: defaultAnthropicAPIBase,
		},
		{
			name: "claude alias with oauth routes to claude auth",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Provider = "claude"
				cfg.Providers.Anthropic.AuthMethod = "oauth"
			},
			wantType: providerTypeClaudeAuth,
		},
		{
			name: "groq explicit provider uses groq defaults",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Provider = "groq"
				cfg.Providers.Groq.APIKey = "gsk-test"
			},
			wantType:    providerTypeHTTPCompat,
			wantAPIBase: "https://api.groq.com/openai/v1",
		},
		{
			name: "openai explicit codex-cli auth method returns codex cli token",
			setup: func(cfg *config.Config) {
				cfg.Agents.Defaults.Provider = "openai"
				cfg.Providers.OpenAI.AuthMethod = "codex-cli"
			},
			wantType: providerTypeCodexCLIToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			tt.setup(cfg)

			got, err := resolveProviderSelection(cfg)
			if tt.wantErrSubstr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErrSubstr)
				}
				if !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("error = %q, want substring %q", err.Error(), tt.wantErrSubstr)
				}
				return
			}

			if err != nil {
				t.Fatalf("resolveProviderSelection() error = %v", err)
			}
			if got.providerType != tt.wantType {
				t.Fatalf("providerType = %v, want %v", got.providerType, tt.wantType)
			}
			if tt.wantAPIBase != "" && got.apiBase != tt.wantAPIBase {
				t.Fatalf("apiBase = %q, want %q", got.apiBase, tt.wantAPIBase)
			}
			if tt.wantProxy != "" && got.proxy != tt.wantProxy {
				t.Fatalf("proxy = %q, want %q", got.proxy, tt.wantProxy)
			}
		})
	}
}

func TestCreateProviderReturnsHTTPProviderForOpenRouter(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "test-openrouter"
	cfg.ModelList = []config.ModelConfig{
		{
			ModelName: "test-openrouter",
			Model:     "openrouter/auto",
			APIKey:    *config.NewSecureString("sk-or-test"),
			APIBase:   "https://openrouter.ai/api/v1",
		},
	}

	provider, _, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}

	if _, ok := provider.(*HTTPProvider); !ok {
		t.Fatalf("provider type = %T, want *HTTPProvider", provider)
	}
}

func TestCreateProviderReturnsCodexCliProviderForCodexCode(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "test-codex"
	cfg.ModelList = []config.ModelConfig{
		{
			ModelName: "test-codex",
			Model:     "codex-cli/codex-model",
			Workspace: "/tmp/workspace",
		},
	}

	provider, _, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}

	if _, ok := provider.(*CodexCliProvider); !ok {
		t.Fatalf("provider type = %T, want *CodexCliProvider", provider)
	}
}

func TestCreateProviderReturnsClaudeCliProviderForClaudeCli(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "test-claude-cli"
	cfg.ModelList = []config.ModelConfig{
		{
			ModelName: "test-claude-cli",
			Model:     "claude-cli/claude-sonnet",
			Workspace: "/tmp/workspace",
		},
	}

	provider, _, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}

	if _, ok := provider.(*ClaudeCliProvider); !ok {
		t.Fatalf("provider type = %T, want *ClaudeCliProvider", provider)
	}
}

func TestCreateProviderReturnsClaudeProviderForAnthropicOAuth(t *testing.T) {
	originalGetCredential := getCredential
	t.Cleanup(func() { getCredential = originalGetCredential })

	getCredential = func(provider string) (*auth.AuthCredential, error) {
		if provider != "anthropic" {
			t.Fatalf("provider = %q, want anthropic", provider)
		}
		return &auth.AuthCredential{
			AccessToken: "anthropic-token",
		}, nil
	}

	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.Model = "test-claude-oauth"
	cfg.ModelList = []config.ModelConfig{
		{
			ModelName:  "test-claude-oauth",
			Model:      "anthropic/claude-sonnet-4.6",
			AuthMethod: "oauth",
		},
	}

	provider, _, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}

	if _, ok := provider.(*ClaudeProvider); !ok {
		t.Fatalf("provider type = %T, want *ClaudeProvider", provider)
	}
	// TODO: Test custom APIBase when createClaudeAuthProvider supports it
}

func TestCreateProviderReturnsCodexProviderForOpenAIOAuth(t *testing.T) {
	// TODO: This test requires openai protocol to support auth_method: "oauth"
	// which is not yet implemented in the new factory_provider.go
	t.Skip("OpenAI OAuth via model_list not yet implemented")
}

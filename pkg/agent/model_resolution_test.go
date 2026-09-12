package agent

import (
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/config"
	"github.com/cryptoquantumwave/khunquant/pkg/providers"
)

func TestResolvedCandidateModel_WithCandidates(t *testing.T) {
	candidates := []providers.FallbackCandidate{{Model: "openai/gpt-4o", Provider: "openai"}}
	got := resolvedCandidateModel(candidates, "fallback-model")
	if got != "openai/gpt-4o" {
		t.Errorf("resolvedCandidateModel = %q, want openai/gpt-4o", got)
	}
}

func TestResolvedCandidateModel_Empty(t *testing.T) {
	got := resolvedCandidateModel(nil, "fallback-model")
	if got != "fallback-model" {
		t.Errorf("resolvedCandidateModel with nil = %q, want fallback-model", got)
	}
}

func TestResolvedCandidateModel_EmptyModelField(t *testing.T) {
	candidates := []providers.FallbackCandidate{{Model: "  ", Provider: "openai"}}
	got := resolvedCandidateModel(candidates, "fallback-model")
	if got != "fallback-model" {
		t.Errorf("resolvedCandidateModel with blank model = %q, want fallback-model", got)
	}
}

func TestResolvedCandidateProvider_WithCandidates(t *testing.T) {
	candidates := []providers.FallbackCandidate{{Model: "gpt-4o", Provider: "openai"}}
	got := resolvedCandidateProvider(candidates, "fallback-provider")
	if got != "openai" {
		t.Errorf("resolvedCandidateProvider = %q, want openai", got)
	}
}

func TestResolvedCandidateProvider_Empty(t *testing.T) {
	got := resolvedCandidateProvider(nil, "fallback-provider")
	if got != "fallback-provider" {
		t.Errorf("resolvedCandidateProvider with nil = %q, want fallback-provider", got)
	}
}

func TestResolvedCandidateProvider_EmptyProviderField(t *testing.T) {
	candidates := []providers.FallbackCandidate{{Model: "gpt-4o", Provider: ""}}
	got := resolvedCandidateProvider(candidates, "fallback-provider")
	if got != "fallback-provider" {
		t.Errorf("resolvedCandidateProvider with blank provider = %q, want fallback-provider", got)
	}
}

func TestBuildModelListResolver_NilConfig(t *testing.T) {
	resolve := buildModelListResolver(nil)
	_, ok := resolve("any-model")
	if ok {
		t.Error("resolver with nil config should return false")
	}
}

func TestBuildModelListResolver_EmptyModel(t *testing.T) {
	resolve := buildModelListResolver(config.DefaultConfig())
	_, ok := resolve("")
	if ok {
		t.Error("resolver with empty model should return false")
	}
}

func TestBuildModelListResolver_ModelWithProtocol(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ModelList = []config.ModelConfig{
		{ModelName: "my-gpt4", Model: "openai/gpt-4o"},
	}
	resolve := buildModelListResolver(cfg)
	model, ok := resolve("my-gpt4")
	if !ok {
		t.Fatal("resolver should find model by alias")
	}
	if model != "openai/gpt-4o" {
		t.Errorf("resolved model = %q, want openai/gpt-4o", model)
	}
}

func TestBuildModelListResolver_ModelWithoutProtocol(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ModelList = []config.ModelConfig{
		{ModelName: "my-local", Model: "local-model"},
	}
	resolve := buildModelListResolver(cfg)
	model, ok := resolve("my-local")
	if !ok {
		t.Fatal("resolver should find model by alias")
	}
	// Models without protocol get "openai/" prepended
	if model != "openai/local-model" {
		t.Errorf("resolved model = %q, want openai/local-model", model)
	}
}

func TestResolvedModelConfig_NilConfig(t *testing.T) {
	_, err := resolvedModelConfig(nil, "my-model", "/ws")
	if err == nil {
		t.Error("resolvedModelConfig with nil config should return error")
	}
}

func TestResolvedModelConfig_NotFound(t *testing.T) {
	cfg := config.DefaultConfig()
	_, err := resolvedModelConfig(cfg, "nonexistent-model", "/ws")
	if err == nil {
		t.Error("resolvedModelConfig for nonexistent model should return error")
	}
}

func TestResolvedModelConfig_Found(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ModelList = []config.ModelConfig{
		{ModelName: "my-model", Model: "openai/gpt-4o"},
	}
	mc, err := resolvedModelConfig(cfg, "my-model", "/ws")
	if err != nil {
		t.Fatalf("resolvedModelConfig error: %v", err)
	}
	if mc.Model != "openai/gpt-4o" {
		t.Errorf("mc.Model = %q, want openai/gpt-4o", mc.Model)
	}
	if mc.Workspace != "/ws" {
		t.Errorf("mc.Workspace = %q, want /ws", mc.Workspace)
	}
}

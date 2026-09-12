package tools

import (
	"context"
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/config"
)

func TestGetMarkets_MissingProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetMarketsTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{})
	if !result.IsError {
		t.Fatal("expected error when provider is missing")
	}
}

func TestGetMarkets_EmptyProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetMarketsTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "",
	})
	if !result.IsError {
		t.Fatal("expected error when provider is empty")
	}
}

func TestGetMarkets_NonexistentProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetMarketsTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetMarkets_WithAccount(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetMarketsTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"account":  "myaccount",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetMarkets_WithBaseFilter(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetMarketsTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"base":     "BTC",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetMarkets_WithQuoteFilter(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetMarketsTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"quote":    "USDT",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetMarkets_WithTypeFilter(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetMarketsTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"type":     "spot",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetMarkets_AllFilters(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetMarketsTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"base":     "BTC",
		"quote":    "USDT",
		"type":     "spot",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetMarkets_CaseSensitivity(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetMarketsTool(cfg)

	// Filters should be case-insensitive, tool will uppercase/lowercase them
	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"base":     "btc",
		"quote":    "usdt",
		"type":     "SPOT",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetMarkets_ParametersSchema(t *testing.T) {
	tool := NewGetMarketsTool(config.DefaultConfig())
	params := tool.Parameters()

	if params == nil {
		t.Fatal("Parameters() should not return nil")
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties in Parameters")
	}

	expectedProps := []string{"provider", "account", "base", "quote", "type"}
	for _, prop := range expectedProps {
		if _, ok := props[prop]; !ok {
			t.Errorf("expected property %q in Parameters", prop)
		}
	}

	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("expected required in Parameters")
	}
	// Only provider should be required
	if len(required) != 1 {
		t.Errorf("expected 1 required field, got %d", len(required))
	}
}

func TestGetMarkets_Name(t *testing.T) {
	tool := NewGetMarketsTool(config.DefaultConfig())
	name := tool.Name()
	if name != NameGetMarkets {
		t.Errorf("Name() = %q, want %q", name, NameGetMarkets)
	}
}

func TestGetMarkets_Description(t *testing.T) {
	tool := NewGetMarketsTool(config.DefaultConfig())
	desc := tool.Description()
	if desc == "" {
		t.Fatal("Description() should not be empty")
	}
}

func TestGetMarkets_EmptyFilters(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetMarketsTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"base":     "",
		"quote":    "",
		"type":     "",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetMarkets_SuccessWithMock(t *testing.T) {
	// LoadMarkets returns nil map by default — exercises empty-markets output path
	tool := NewGetMarketsTool(config.DefaultConfig())
	result := tool.Execute(context.Background(), map[string]any{
		"provider": "mock-market-dyn",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %q", result.ForLLM)
	}
}

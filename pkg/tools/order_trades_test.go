package tools

import (
	"context"
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/config"
)

func TestGetTradeHistory_MissingProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetTradeHistoryTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{})
	if !result.IsError {
		t.Fatal("expected error when provider is missing")
	}
}

func TestGetTradeHistory_EmptyProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetTradeHistoryTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "",
	})
	if !result.IsError {
		t.Fatal("expected error when provider is empty")
	}
}

func TestGetTradeHistory_NonexistentProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetTradeHistoryTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetTradeHistory_WithSymbol(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetTradeHistoryTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"symbol":   "BTC/USDT",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetTradeHistory_WithSince(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetTradeHistoryTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"since":    float64(1609459200000), // 2021-01-01
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetTradeHistory_WithLimit(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetTradeHistoryTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"limit":    float64(50),
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetTradeHistory_LimitExceedsMax(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetTradeHistoryTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"limit":    float64(maxTradeHistoryLimit + 100),
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetTradeHistory_AllParameters(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetTradeHistoryTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"account":  "myaccount",
		"symbol":   "BTC/USDT",
		"since":    float64(1609459200000),
		"limit":    float64(100),
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetTradeHistory_ParametersSchema(t *testing.T) {
	tool := NewGetTradeHistoryTool(config.DefaultConfig())
	params := tool.Parameters()

	if params == nil {
		t.Fatal("Parameters() should not return nil")
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties in Parameters")
	}

	expectedProps := []string{"provider", "account", "symbol", "since", "limit"}
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

func TestGetTradeHistory_Name(t *testing.T) {
	tool := NewGetTradeHistoryTool(config.DefaultConfig())
	name := tool.Name()
	if name != NameGetTradeHistory {
		t.Errorf("Name() = %q, want %q", name, NameGetTradeHistory)
	}
}

func TestGetTradeHistory_Description(t *testing.T) {
	tool := NewGetTradeHistoryTool(config.DefaultConfig())
	desc := tool.Description()
	if desc == "" {
		t.Fatal("Description() should not be empty")
	}
}

func TestGetTradeHistory_WithAccount(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetTradeHistoryTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"account":  "myaccount",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetTradeHistory_MaxLimit(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetTradeHistoryTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"limit":    float64(maxTradeHistoryLimit),
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

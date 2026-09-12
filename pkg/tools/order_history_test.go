package tools

import (
	"context"
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/config"
)

func TestGetOrderHistory_MissingProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOrderHistoryTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{})
	if !result.IsError {
		t.Fatal("expected error when provider is missing")
	}
}

func TestGetOrderHistory_EmptyProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOrderHistoryTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "",
	})
	if !result.IsError {
		t.Fatal("expected error when provider is empty")
	}
}

func TestGetOrderHistory_NonexistentProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOrderHistoryTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetOrderHistory_WithSymbol(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOrderHistoryTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"symbol":   "BTC/USDT",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetOrderHistory_WithSince(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOrderHistoryTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"since":    float64(1609459200000), // 2021-01-01
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetOrderHistory_WithLimit(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOrderHistoryTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"limit":    float64(50),
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetOrderHistory_LimitExceedsMax(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOrderHistoryTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"limit":    float64(maxOrderHistoryLimit + 100),
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetOrderHistory_AllParameters(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOrderHistoryTool(cfg)

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

func TestGetOrderHistory_ParametersSchema(t *testing.T) {
	tool := NewGetOrderHistoryTool(config.DefaultConfig())
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

func TestGetOrderHistory_Name(t *testing.T) {
	tool := NewGetOrderHistoryTool(config.DefaultConfig())
	name := tool.Name()
	if name != NameGetOrderHistory {
		t.Errorf("Name() = %q, want %q", name, NameGetOrderHistory)
	}
}

func TestGetOrderHistory_Description(t *testing.T) {
	tool := NewGetOrderHistoryTool(config.DefaultConfig())
	desc := tool.Description()
	if desc == "" {
		t.Fatal("Description() should not be empty")
	}
}

func TestGetOrderHistory_ZeroSince(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOrderHistoryTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"since":    float64(0),
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetOrderHistory_MaxLimit(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOrderHistoryTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"limit":    float64(maxOrderHistoryLimit),
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

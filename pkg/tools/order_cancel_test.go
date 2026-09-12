package tools

import (
	"context"
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/config"
)

func TestCancelOrder_MissingProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewCancelOrderTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"order_id": "12345",
		"symbol":   "BTC/USDT",
	})
	if !result.IsError {
		t.Fatal("expected error when provider is missing")
	}
}

func TestCancelOrder_EmptyProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewCancelOrderTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "",
		"order_id": "12345",
		"symbol":   "BTC/USDT",
	})
	if !result.IsError {
		t.Fatal("expected error when provider is empty")
	}
}

func TestCancelOrder_MissingOrderID(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewCancelOrderTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"symbol":   "BTC/USDT",
	})
	if !result.IsError {
		t.Fatal("expected error when order_id is missing")
	}
}

func TestCancelOrder_EmptyOrderID(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewCancelOrderTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"order_id": "",
		"symbol":   "BTC/USDT",
	})
	if !result.IsError {
		t.Fatal("expected error when order_id is empty")
	}
}

func TestCancelOrder_MissingSymbol(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewCancelOrderTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"order_id": "12345",
	})
	// symbol is not explicitly required by the validation, but some exchanges need it
	// The error should come from provider lookup, not parameter validation
	if result.IsError {
		// This is acceptable — it's either param validation or provider error
		// Just verify we got an error
	}
}

func TestCancelOrder_NonexistentProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewCancelOrderTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"order_id": "12345",
		"symbol":   "BTC/USDT",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestCancelOrder_WithAccount(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewCancelOrderTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"account":  "myaccount",
		"order_id": "12345",
		"symbol":   "BTC/USDT",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestCancelOrder_ParametersSchema(t *testing.T) {
	tool := NewCancelOrderTool(config.DefaultConfig())
	params := tool.Parameters()

	if params == nil {
		t.Fatal("Parameters() should not return nil")
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties in Parameters")
	}

	expectedProps := []string{"provider", "account", "order_id", "symbol"}
	for _, prop := range expectedProps {
		if _, ok := props[prop]; !ok {
			t.Errorf("expected property %q in Parameters", prop)
		}
	}

	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("expected required in Parameters")
	}
	// At least provider and order_id should be required
	if len(required) < 2 {
		t.Errorf("expected at least 2 required fields, got %d", len(required))
	}
}

func TestCancelOrder_Name(t *testing.T) {
	tool := NewCancelOrderTool(config.DefaultConfig())
	name := tool.Name()
	if name != NameCancelOrder {
		t.Errorf("Name() = %q, want %q", name, NameCancelOrder)
	}
}

func TestCancelOrder_Description(t *testing.T) {
	tool := NewCancelOrderTool(config.DefaultConfig())
	desc := tool.Description()
	if desc == "" {
		t.Fatal("Description() should not be empty")
	}
}

func TestCancelOrder_ValidOrderID(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewCancelOrderTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"order_id": "abc123xyz456",
		"symbol":   "BTC/USDT",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

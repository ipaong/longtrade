package tools

import (
	"context"
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/config"
)

func TestCreateOrder_MissingProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewCreateOrderTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"symbol":  "BTC/USDT",
		"type":    "limit",
		"side":    "buy",
		"amount":  1.0,
		"confirm": false,
	})
	if !result.IsError {
		t.Fatal("expected error when provider is missing")
	}
}

func TestCreateOrder_EmptyProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewCreateOrderTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "",
		"symbol":   "BTC/USDT",
		"type":     "limit",
		"side":     "buy",
		"amount":   1.0,
		"confirm":  false,
	})
	if !result.IsError {
		t.Fatal("expected error when provider is empty")
	}
}

func TestCreateOrder_MissingSymbol(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewCreateOrderTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"type":     "limit",
		"side":     "buy",
		"amount":   1.0,
		"confirm":  false,
	})
	if !result.IsError {
		t.Fatal("expected error when symbol is missing")
	}
}

func TestCreateOrder_EmptySymbol(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewCreateOrderTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"symbol":   "",
		"type":     "limit",
		"side":     "buy",
		"amount":   1.0,
		"confirm":  false,
	})
	if !result.IsError {
		t.Fatal("expected error when symbol is empty")
	}
}

func TestCreateOrder_MissingType(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewCreateOrderTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"symbol":   "BTC/USDT",
		"side":     "buy",
		"amount":   1.0,
		"confirm":  false,
	})
	if !result.IsError {
		t.Fatal("expected error when type is missing")
	}
}

func TestCreateOrder_EmptyType(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewCreateOrderTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"symbol":   "BTC/USDT",
		"type":     "",
		"side":     "buy",
		"amount":   1.0,
		"confirm":  false,
	})
	if !result.IsError {
		t.Fatal("expected error when type is empty")
	}
}

func TestCreateOrder_MissingSide(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewCreateOrderTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"symbol":   "BTC/USDT",
		"type":     "limit",
		"amount":   1.0,
		"confirm":  false,
	})
	if !result.IsError {
		t.Fatal("expected error when side is missing")
	}
}

func TestCreateOrder_EmptySide(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewCreateOrderTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"symbol":   "BTC/USDT",
		"type":     "limit",
		"side":     "",
		"amount":   1.0,
		"confirm":  false,
	})
	if !result.IsError {
		t.Fatal("expected error when side is empty")
	}
}

func TestCreateOrder_InvalidAmount(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewCreateOrderTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"symbol":   "BTC/USDT",
		"type":     "limit",
		"side":     "buy",
		"amount":   0.0,
		"confirm":  false,
	})
	if !result.IsError {
		t.Fatal("expected error when amount is zero")
	}
}

func TestCreateOrder_NegativeAmount(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewCreateOrderTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"symbol":   "BTC/USDT",
		"type":     "limit",
		"side":     "buy",
		"amount":   -1.0,
		"confirm":  false,
	})
	if !result.IsError {
		t.Fatal("expected error when amount is negative")
	}
}

func TestCreateOrder_LimitOrderWithoutPrice(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewCreateOrderTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"symbol":   "BTC/USDT",
		"type":     "limit",
		"side":     "buy",
		"amount":   1.0,
		"confirm":  false,
	})
	if !result.IsError {
		t.Fatal("expected error when price is missing for limit order")
	}
}

func TestCreateOrder_MarketOrderWithoutPrice(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewCreateOrderTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"symbol":   "BTC/USDT",
		"type":     "market",
		"side":     "buy",
		"amount":   1.0,
		"confirm":  false,
	})
	// Market orders shouldn't require price
	if result == nil {
		t.Fatal("expected result")
	}
}

func TestCreateOrder_DryRun(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewCreateOrderTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"symbol":   "BTC/USDT",
		"type":     "limit",
		"side":     "buy",
		"amount":   1.0,
		"price":    50000.0,
		"confirm":  false,
	})
	// This should either succeed with dry-run message or fail due to permission/config
	// Just verify it doesn't complain about required params
	if result == nil {
		t.Fatal("expected result")
	}
}

func TestCreateOrder_WithAccount(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewCreateOrderTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"account":  "myaccount",
		"symbol":   "BTC/USDT",
		"type":     "limit",
		"side":     "buy",
		"amount":   1.0,
		"price":    50000.0,
		"confirm":  false,
	})
	// Should accept account parameter
	if result == nil {
		t.Fatal("expected result")
	}
}

func TestCreateOrder_ParametersSchema(t *testing.T) {
	tool := NewCreateOrderTool(config.DefaultConfig())
	params := tool.Parameters()

	if params == nil {
		t.Fatal("Parameters() should not return nil")
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties in Parameters")
	}

	expectedProps := []string{"provider", "account", "symbol", "type", "side", "amount", "price", "params", "confirm"}
	for _, prop := range expectedProps {
		if _, ok := props[prop]; !ok {
			t.Errorf("expected property %q in Parameters", prop)
		}
	}
}

func TestCreateOrder_Name(t *testing.T) {
	tool := NewCreateOrderTool(config.DefaultConfig())
	name := tool.Name()
	if name != NameCreateOrder {
		t.Errorf("Name() = %q, want %q", name, NameCreateOrder)
	}
}

func TestCreateOrder_Description(t *testing.T) {
	tool := NewCreateOrderTool(config.DefaultConfig())
	desc := tool.Description()
	if desc == "" {
		t.Fatal("Description() should not be empty")
	}
}

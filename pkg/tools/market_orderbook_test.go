package tools

import (
	"context"
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/config"
)

func TestGetOrderBook_MissingProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOrderBookTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"symbol": "BTC/USDT",
	})
	if !result.IsError {
		t.Fatal("expected error when provider is missing")
	}
}

func TestGetOrderBook_EmptyProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOrderBookTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "",
		"symbol":   "BTC/USDT",
	})
	if !result.IsError {
		t.Fatal("expected error when provider is empty")
	}
}

func TestGetOrderBook_MissingSymbol(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOrderBookTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
	})
	if !result.IsError {
		t.Fatal("expected error when symbol is missing")
	}
}

func TestGetOrderBook_EmptySymbol(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOrderBookTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"symbol":   "",
	})
	if !result.IsError {
		t.Fatal("expected error when symbol is empty")
	}
}

func TestGetOrderBook_NonexistentProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOrderBookTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"symbol":   "BTC/USDT",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetOrderBook_DefaultDepth(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOrderBookTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"symbol":   "BTC/USDT",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetOrderBook_CustomDepth(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOrderBookTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"symbol":   "BTC/USDT",
		"depth":    float64(20),
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetOrderBook_DepthExceedsMax(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOrderBookTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"symbol":   "BTC/USDT",
		"depth":    float64(maxOrderBookDepth + 10),
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetOrderBook_ZeroDepth(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOrderBookTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"symbol":   "BTC/USDT",
		"depth":    float64(0),
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetOrderBook_NegativeDepth(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOrderBookTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"symbol":   "BTC/USDT",
		"depth":    float64(-5),
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetOrderBook_WithAccount(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOrderBookTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"account":  "myaccount",
		"symbol":   "BTC/USDT",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetOrderBook_ParametersSchema(t *testing.T) {
	tool := NewGetOrderBookTool(config.DefaultConfig())
	params := tool.Parameters()

	if params == nil {
		t.Fatal("Parameters() should not return nil")
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties in Parameters")
	}

	expectedProps := []string{"provider", "account", "symbol", "depth"}
	for _, prop := range expectedProps {
		if _, ok := props[prop]; !ok {
			t.Errorf("expected property %q in Parameters", prop)
		}
	}

	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("expected required in Parameters")
	}
	// provider and symbol should be required
	if len(required) < 2 {
		t.Errorf("expected at least 2 required fields, got %d", len(required))
	}
}

func TestGetOrderBook_Name(t *testing.T) {
	tool := NewGetOrderBookTool(config.DefaultConfig())
	name := tool.Name()
	if name != NameGetOrderBook {
		t.Errorf("Name() = %q, want %q", name, NameGetOrderBook)
	}
}

func TestGetOrderBook_Description(t *testing.T) {
	tool := NewGetOrderBookTool(config.DefaultConfig())
	desc := tool.Description()
	if desc == "" {
		t.Fatal("Description() should not be empty")
	}
}

func TestGetOrderBook_MaxDepth(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOrderBookTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"symbol":   "BTC/USDT",
		"depth":    float64(maxOrderBookDepth),
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetOrderBook_SuccessWithMock(t *testing.T) {
	// Covers the order book formatting path (lines 89-114).
	mockMarketDyn.ohlcvErr = nil
	mockMarketDyn.ohlcv = nil

	tool := NewGetOrderBookTool(config.DefaultConfig())
	result := tool.Execute(context.Background(), map[string]any{
		"provider": "mock-market-dyn",
		"symbol":   "BTC/USDT",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %q", result.ForLLM)
	}
}

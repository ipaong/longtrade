package tools

import (
	"context"
	"testing"

	ccxt "github.com/ccxt/ccxt/go/v4"
	"github.com/cryptoquantumwave/khunquant/pkg/config"
)

func TestGetTicker_MissingProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetTickerTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"symbol": "BTC/USDT",
	})
	if !result.IsError {
		t.Fatal("expected error when provider is missing")
	}
}

func TestGetTicker_EmptyProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetTickerTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "",
		"symbol":   "BTC/USDT",
	})
	if !result.IsError {
		t.Fatal("expected error when provider is empty")
	}
}

func TestGetTicker_MissingSymbol(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetTickerTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
	})
	if !result.IsError {
		t.Fatal("expected error when symbol is missing")
	}
}

func TestGetTicker_EmptySymbol(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetTickerTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"symbol":   "",
	})
	if !result.IsError {
		t.Fatal("expected error when symbol is empty")
	}
}

func TestGetTicker_NonexistentProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetTickerTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"symbol":   "BTC/USDT",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetTicker_WithAccount(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetTickerTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"account":  "myaccount",
		"symbol":   "BTC/USDT",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetTicker_ParametersSchema(t *testing.T) {
	tool := NewGetTickerTool(config.DefaultConfig())
	params := tool.Parameters()

	if params == nil {
		t.Fatal("Parameters() should not return nil")
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties in Parameters")
	}

	expectedProps := []string{"provider", "account", "symbol"}
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

func TestGetTicker_Name(t *testing.T) {
	tool := NewGetTickerTool(config.DefaultConfig())
	name := tool.Name()
	if name != NameGetTicker {
		t.Errorf("Name() = %q, want %q", name, NameGetTicker)
	}
}

func TestGetTicker_Description(t *testing.T) {
	tool := NewGetTickerTool(config.DefaultConfig())
	desc := tool.Description()
	if desc == "" {
		t.Fatal("Description() should not be empty")
	}
	if len(desc) < 20 {
		t.Fatal("Description() too short")
	}
}

func TestGetTicker_SuccessWithMock(t *testing.T) {
	last := 50000.0
	high := 51000.0
	low := 49000.0
	pct := 2.5
	vol := 100.0
	mockMarketDyn.tickerErr = nil
	mockMarketDyn.ticker = ccxt.Ticker{Last: &last, High: &high, Low: &low, Percentage: &pct, BaseVolume: &vol}
	t.Cleanup(func() { mockMarketDyn.ticker = ccxt.Ticker{} })

	tool := NewGetTickerTool(config.DefaultConfig())
	result := tool.Execute(context.Background(), map[string]any{
		"provider": "mock-market-dyn",
		"symbol":   "BTC/USDT",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %q", result.ForLLM)
	}
}

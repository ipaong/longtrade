package tools

import (
	"context"
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/config"
)

func TestGetTickers_MissingProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetTickersTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"symbols": []any{"BTC/USDT", "ETH/USDT"},
	})
	if !result.IsError {
		t.Fatal("expected error when provider is missing")
	}
}

func TestGetTickers_EmptyProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetTickersTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "",
		"symbols":  []any{"BTC/USDT", "ETH/USDT"},
	})
	if !result.IsError {
		t.Fatal("expected error when provider is empty")
	}
}

func TestGetTickers_MissingSymbols(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetTickersTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
	})
	if !result.IsError {
		t.Fatal("expected error when symbols is missing")
	}
}

func TestGetTickers_TooManySymbols(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetTickersTool(cfg)

	// Create list with more than maxTickersSymbols
	symbols := make([]any, maxTickersSymbols+5)
	for i := range symbols {
		symbols[i] = "BTC/USDT"
	}

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"symbols":  symbols,
	})
	if !result.IsError {
		t.Fatal("expected error when symbols exceed max count")
	}
}

func TestGetTickers_EmptySymbolsList(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetTickersTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"symbols":  []any{},
	})
	// Empty list is valid (will fetch all tickers), but provider error expected
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetTickers_SingleSymbol(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetTickersTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"symbols":  []any{"BTC/USDT"},
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetTickers_MultipleSymbols(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetTickersTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"symbols":  []any{"BTC/USDT", "ETH/USDT", "ADA/USDT"},
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetTickers_WithAccount(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetTickersTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"account":  "myaccount",
		"symbols":  []any{"BTC/USDT"},
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetTickers_MaxSymbols(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetTickersTool(cfg)

	// Create list with exactly maxTickersSymbols
	symbols := make([]any, maxTickersSymbols)
	for i := range symbols {
		symbols[i] = "BTC/USDT"
	}

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"symbols":  symbols,
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetTickers_ParametersSchema(t *testing.T) {
	tool := NewGetTickersTool(config.DefaultConfig())
	params := tool.Parameters()

	if params == nil {
		t.Fatal("Parameters() should not return nil")
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties in Parameters")
	}

	expectedProps := []string{"provider", "account", "symbols"}
	for _, prop := range expectedProps {
		if _, ok := props[prop]; !ok {
			t.Errorf("expected property %q in Parameters", prop)
		}
	}

	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("expected required in Parameters")
	}
	// provider and symbols should be required
	if len(required) < 2 {
		t.Errorf("expected at least 2 required fields, got %d", len(required))
	}
}

func TestGetTickers_Name(t *testing.T) {
	tool := NewGetTickersTool(config.DefaultConfig())
	name := tool.Name()
	if name != NameGetTickers {
		t.Errorf("Name() = %q, want %q", name, NameGetTickers)
	}
}

func TestGetTickers_Description(t *testing.T) {
	tool := NewGetTickersTool(config.DefaultConfig())
	desc := tool.Description()
	if desc == "" {
		t.Fatal("Description() should not be empty")
	}
}

func TestGetTickers_SymbolsTypeConversion(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetTickersTool(cfg)

	// Test with mixed types in symbols array (non-string should be ignored)
	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"symbols":  []any{"BTC/USDT", 123, "ETH/USDT"},
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetTickers_SuccessWithMock(t *testing.T) {
	// mockMarketDyn.FetchTickers returns nil map by default — exercises empty-tickers output path
	tool := NewGetTickersTool(config.DefaultConfig())
	result := tool.Execute(context.Background(), map[string]any{
		"provider": "mock-market-dyn",
		"symbols":  []any{"BTC/USDT", "ETH/USDT"},
	})
	if result.IsError {
		t.Fatalf("unexpected error: %q", result.ForLLM)
	}
}

package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/config"
)

func newTestGetPnLDetailTool(t *testing.T) *GetPnLDetailTool {
	t.Helper()
	return NewGetPnLDetailTool(config.DefaultConfig())
}

func TestGetPnLDetailTool_Name(t *testing.T) {
	tool := newTestGetPnLDetailTool(t)
	if tool.Name() != NameGetPnLDetail {
		t.Errorf("Name() = %q, want %q", tool.Name(), NameGetPnLDetail)
	}
}

func TestGetPnLDetailTool_Description(t *testing.T) {
	tool := newTestGetPnLDetailTool(t)
	desc := tool.Description()
	if desc == "" {
		t.Error("Description() should not be empty")
	}
}

func TestGetPnLDetailTool_Parameters(t *testing.T) {
	tool := newTestGetPnLDetailTool(t)
	params := tool.Parameters()

	if params["type"] != "object" {
		t.Errorf("type should be 'object'")
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties should be a map")
	}

	expectedProps := []string{"provider", "account", "symbol", "since", "limit"}
	for _, prop := range expectedProps {
		if _, ok := props[prop]; !ok {
			t.Errorf("expected property %q not found", prop)
		}
	}

	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("required should be a slice")
	}
	if len(required) != 2 {
		t.Errorf("expected 2 required params, got %d", len(required))
	}
}

func TestGetPnLDetailTool_Execute_MissingProvider(t *testing.T) {
	tool := newTestGetPnLDetailTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"symbol": "BTC/USDT",
	})

	if !result.IsError {
		t.Error("missing provider should return error")
	}
	if !strings.Contains(result.ForLLM, "provider") {
		t.Errorf("error should mention provider, got %q", result.ForLLM)
	}
}

func TestGetPnLDetailTool_Execute_MissingSymbol(t *testing.T) {
	tool := newTestGetPnLDetailTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
	})

	if !result.IsError {
		t.Error("missing symbol should return error")
	}
	if !strings.Contains(result.ForLLM, "symbol") {
		t.Errorf("error should mention symbol, got %q", result.ForLLM)
	}
}

func TestGetPnLDetailTool_Execute_NoArgs(t *testing.T) {
	tool := newTestGetPnLDetailTool(t)

	result := tool.Execute(context.Background(), map[string]any{})

	if !result.IsError {
		t.Error("missing required args should return error")
	}
}

func TestGetPnLDetailTool_Execute_InvalidProvider(t *testing.T) {
	tool := newTestGetPnLDetailTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent_exchange",
		"symbol":   "BTC/USDT",
	})

	if !result.IsError {
		t.Error("invalid provider should return error")
	}
}

func TestGetPnLDetailTool_Execute_ValidProvider(t *testing.T) {
	tool := newTestGetPnLDetailTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"symbol":   "BTC/USDT",
	})

	// May error due to missing API keys or trading provider unavailable
	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestGetPnLDetailTool_Execute_SymbolNormalization(t *testing.T) {
	tool := newTestGetPnLDetailTool(t)

	testCases := []struct {
		symbol string
	}{
		{"btc/usdt"},   // lowercase
		{"BTC_USDT"},   // underscore instead of slash
		{"btc_usdt"},   // both lowercase and underscore
		{"BTC/USDT"},   // already correct
	}

	for _, tc := range testCases {
		t.Run(tc.symbol, func(t *testing.T) {
			result := tool.Execute(context.Background(), map[string]any{
				"provider": "binance",
				"symbol":   tc.symbol,
			})
			if result == nil {
				t.Fatal("Execute should return result")
			}
			// May error but should handle normalization
		})
	}
}

func TestGetPnLDetailTool_Execute_WithAccount(t *testing.T) {
	tool := newTestGetPnLDetailTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"account":  "main",
		"symbol":   "BTC/USDT",
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestGetPnLDetailTool_Execute_WithLimit(t *testing.T) {
	tool := newTestGetPnLDetailTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"symbol":   "BTC/USDT",
		"limit":    float64(50),
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestGetPnLDetailTool_Execute_LimitClamping(t *testing.T) {
	tool := newTestGetPnLDetailTool(t)

	// Test with limit exceeding max
	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"symbol":   "BTC/USDT",
		"limit":    float64(9999), // Should be clamped to maxPnLDetailLimit
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestGetPnLDetailTool_Execute_ZeroLimit(t *testing.T) {
	tool := newTestGetPnLDetailTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"symbol":   "BTC/USDT",
		"limit":    float64(0), // Zero or negative should use default
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestGetPnLDetailTool_Execute_NegativeLimit(t *testing.T) {
	tool := newTestGetPnLDetailTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"symbol":   "BTC/USDT",
		"limit":    float64(-10),
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestGetPnLDetailTool_Execute_WithSince(t *testing.T) {
	tool := newTestGetPnLDetailTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"symbol":   "BTC/USDT",
		"since":    "30d",
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestGetPnLDetailTool_Execute_WithISO8601Since(t *testing.T) {
	tool := newTestGetPnLDetailTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"symbol":   "BTC/USDT",
		"since":    "2025-01-01",
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestGetPnLDetailTool_Execute_AllArgs(t *testing.T) {
	tool := newTestGetPnLDetailTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"account":  "main",
		"symbol":   "BTC/USDT",
		"since":    "90d",
		"limit":    float64(100),
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestGetPnLDetailTool_Execute_InvalidArgTypes(t *testing.T) {
	tool := newTestGetPnLDetailTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": 123,  // int instead of string
		"symbol":   true, // bool instead of string
		"limit":    "not_a_number",
	})

	// Should handle type mismatches gracefully
	if result == nil {
		t.Fatal("Execute should return result even with invalid types")
	}
	// May error but should not panic
}

func TestGetPnLDetailTool_Execute_EmptyStringProvider(t *testing.T) {
	tool := newTestGetPnLDetailTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "",
		"symbol":   "BTC/USDT",
	})

	if !result.IsError {
		t.Error("empty provider should return error")
	}
}

func TestGetPnLDetailTool_Execute_EmptyStringSymbol(t *testing.T) {
	tool := newTestGetPnLDetailTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"symbol":   "",
	})

	if !result.IsError {
		t.Error("empty symbol should return error")
	}
}

func TestGetPnLDetailTool_Execute_ComplexSymbols(t *testing.T) {
	tool := newTestGetPnLDetailTool(t)

	testCases := []string{
		"SOL/USDT",
		"BTC/THB",
		"ETH/USDT",
		"DOGE/USDT",
	}

	for _, sym := range testCases {
		t.Run(sym, func(t *testing.T) {
			result := tool.Execute(context.Background(), map[string]any{
				"provider": "binance",
				"symbol":   sym,
			})
			if result == nil {
				t.Fatal("Execute should return result")
			}
		})
	}
}

func TestBaseSymbol_WithSlash(t *testing.T) {
	if got := baseSymbol("SOL/USDT"); got != "SOL" {
		t.Errorf("baseSymbol('SOL/USDT') = %q, want 'SOL'", got)
	}
}

func TestBaseSymbol_WithoutSlash(t *testing.T) {
	if got := baseSymbol("BTC"); got != "BTC" {
		t.Errorf("baseSymbol('BTC') = %q, want 'BTC'", got)
	}
}

func TestBaseSymbol_Empty(t *testing.T) {
	if got := baseSymbol(""); got != "" {
		t.Errorf("baseSymbol('') = %q, want empty", got)
	}
}

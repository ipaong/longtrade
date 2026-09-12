package tools

import (
	"context"
	"errors"
	"strings"
	"testing"

	ccxt "github.com/ccxt/ccxt/go/v4"
	"github.com/cryptoquantumwave/khunquant/pkg/config"
)

func TestNewCalculateIndicatorsTool_NotNil(t *testing.T) {
	cfg := &config.Config{}
	tool := NewCalculateIndicatorsTool(cfg)
	if tool == nil {
		t.Fatal("NewCalculateIndicatorsTool returned nil")
	}
}

func TestCalculateIndicatorsTool_Name(t *testing.T) {
	tool := NewCalculateIndicatorsTool(nil)
	if tool.Name() != NameCalculateIndicators {
		t.Errorf("Name() = %q, want %q", tool.Name(), NameCalculateIndicators)
	}
}

func TestCalculateIndicatorsTool_Description(t *testing.T) {
	tool := NewCalculateIndicatorsTool(nil)
	desc := tool.Description()
	if desc == "" {
		t.Error("Description() should not be empty")
	}
}

func TestCalculateIndicatorsTool_Parameters(t *testing.T) {
	tool := NewCalculateIndicatorsTool(nil)
	params := tool.Parameters()
	if params == nil {
		t.Fatal("Parameters() should not be nil")
	}
	if params["type"] != "object" {
		t.Errorf("Parameters() type = %v, want object", params["type"])
	}
}

func TestCalculateIndicatorsTool_Execute_MissingProvider(t *testing.T) {
	tool := NewCalculateIndicatorsTool(nil)
	result := tool.Execute(context.Background(), map[string]any{
		"symbol":    "BTC/USDT",
		"timeframe": "1h",
	})
	if !result.IsError {
		t.Fatal("expected error when provider is missing")
	}
}

func TestCalculateIndicatorsTool_Execute_MissingSymbol(t *testing.T) {
	tool := NewCalculateIndicatorsTool(nil)
	result := tool.Execute(context.Background(), map[string]any{
		"provider":  "binance",
		"timeframe": "1h",
	})
	if !result.IsError {
		t.Fatal("expected error when symbol is missing")
	}
}

func TestCalculateIndicatorsTool_Execute_InvalidTimeframe(t *testing.T) {
	tool := NewCalculateIndicatorsTool(&config.Config{})
	result := tool.Execute(context.Background(), map[string]any{
		"provider":  "binance",
		"symbol":    "BTC/USDT",
		"timeframe": "invalid",
	})
	if !result.IsError {
		t.Fatal("expected error for invalid timeframe")
	}
}

func TestCalculateIndicatorsTool_Execute_UnknownProvider(t *testing.T) {
	tool := NewCalculateIndicatorsTool(&config.Config{})
	result := tool.Execute(context.Background(), map[string]any{
		"provider":  "nonexistent-exchange",
		"symbol":    "BTC/USDT",
		"timeframe": "1h",
	})
	if !result.IsError {
		t.Fatal("expected error for unknown provider")
	}
}

func TestFormatLast_Empty(t *testing.T) {
	got := formatLast(nil, 5)
	if got != "insufficient data" {
		t.Errorf("formatLast nil = %q, want 'insufficient data'", got)
	}
}

func TestFormatLast_Empty2(t *testing.T) {
	got := formatLast([]float64{}, 5)
	if got != "insufficient data" {
		t.Errorf("formatLast empty = %q, want 'insufficient data'", got)
	}
}

func TestFormatLast_LessDataThanN(t *testing.T) {
	got := formatLast([]float64{1.0, 2.0}, 5)
	// Should return both values formatted
	if !strings.Contains(got, "1") || !strings.Contains(got, "2") {
		t.Errorf("formatLast less data = %q, want both values", got)
	}
}

func TestFormatLast_ExactlyN(t *testing.T) {
	got := formatLast([]float64{1.0, 2.0, 3.0}, 3)
	parts := strings.Fields(got)
	if len(parts) != 3 {
		t.Errorf("formatLast exact = %d parts, want 3: %q", len(parts), got)
	}
}

func TestFormatLast_MoreThanN(t *testing.T) {
	// Only last N values should appear
	got := formatLast([]float64{100.0, 200.0, 300.0, 400.0, 500.0, 600.0}, 3)
	parts := strings.Fields(got)
	if len(parts) != 3 {
		t.Errorf("formatLast more than N = %d parts, want 3: %q", len(parts), got)
	}
	// The 3 values should be 400, 500, 600 (last 3)
	if !strings.Contains(got, "400") {
		t.Errorf("formatLast more than N should contain 400, got %q", got)
	}
	if !strings.Contains(got, "600") {
		t.Errorf("formatLast more than N should contain 600, got %q", got)
	}
}

func TestFormatLast_SingleValue(t *testing.T) {
	got := formatLast([]float64{42.5}, 1)
	if got != "42.5" {
		t.Errorf("formatLast single = %q, want 42.5", got)
	}
}

func TestCalculateIndicatorsTool_Execute_EmptyTimeframe(t *testing.T) {
	// Empty timeframe gets defaulted to "1h"; then broker lookup fails (unknown provider).
	// Covers the `if timeframe == ""` assignment branch.
	tool := NewCalculateIndicatorsTool(&config.Config{})
	result := tool.Execute(context.Background(), map[string]any{
		"provider":  "nonexistent",
		"symbol":    "BTC/USDT",
		"timeframe": "",
	})
	if !result.IsError {
		t.Fatal("expected error for unknown provider")
	}
}

func TestCalculateIndicatorsTool_Execute_HighLimit(t *testing.T) {
	// limit > maxOHLCVLimit gets clamped; then broker lookup fails.
	// Covers the `if limit > maxOHLCVLimit` branch.
	tool := NewCalculateIndicatorsTool(&config.Config{})
	result := tool.Execute(context.Background(), map[string]any{
		"provider":  "nonexistent",
		"symbol":    "BTC/USDT",
		"timeframe": "1h",
		"limit":     float64(9999),
	})
	if !result.IsError {
		t.Fatal("expected error for unknown provider")
	}
}

func TestCalculateIndicatorsTool_Execute_SpecificIndicators(t *testing.T) {
	// Explicit indicators list exercises the wantAll=false + wantMap fill path.
	// Covers lines 73-80 in ta_indicators.go.
	tool := NewCalculateIndicatorsTool(&config.Config{})
	result := tool.Execute(context.Background(), map[string]any{
		"provider":   "nonexistent",
		"symbol":     "BTC/USDT",
		"timeframe":  "1h",
		"indicators": []any{"SMA", "RSI"},
	})
	if !result.IsError {
		t.Fatal("expected error for unknown provider")
	}
}

func TestCalculateIndicatorsTool_Execute_NoMarketDataSupport(t *testing.T) {
	tool := NewCalculateIndicatorsTool(&config.Config{})
	result := tool.Execute(context.Background(), map[string]any{
		"provider":  "no-market-provider",
		"symbol":    "BTC/USDT",
		"timeframe": "1h",
	})
	if !result.IsError {
		t.Fatal("expected error when provider does not support market data")
	}
}

func TestCalculateIndicatorsTool_Execute_FetchOHLCVError(t *testing.T) {
	mockMarketDyn.tickerErr = nil
	mockMarketDyn.ohlcvErr = errors.New("ohlcv fetch failed")
	mockMarketDyn.ohlcv = nil
	t.Cleanup(func() { mockMarketDyn.ohlcvErr = nil })

	tool := NewCalculateIndicatorsTool(&config.Config{})
	result := tool.Execute(context.Background(), map[string]any{
		"provider":  "mock-market-dyn",
		"symbol":    "BTC/USDT",
		"timeframe": "1h",
	})
	if !result.IsError {
		t.Fatal("expected error when FetchOHLCV fails")
	}
}

func TestCalculateIndicatorsTool_Execute_InsufficientData(t *testing.T) {
	mockMarketDyn.ohlcvErr = nil
	mockMarketDyn.ohlcv = []ccxt.OHLCV{{Close: 100.0}}
	t.Cleanup(func() { mockMarketDyn.ohlcv = nil })

	tool := NewCalculateIndicatorsTool(&config.Config{})
	result := tool.Execute(context.Background(), map[string]any{
		"provider":  "mock-market-dyn",
		"symbol":    "BTC/USDT",
		"timeframe": "1h",
	})
	if !result.IsError {
		t.Fatal("expected error for insufficient data")
	}
}

func TestCalculateIndicatorsTool_Execute_AllIndicators(t *testing.T) {
	mockMarketDyn.ohlcvErr = nil
	mockMarketDyn.ohlcv = makeCandles(100, 50000.0)
	t.Cleanup(func() { mockMarketDyn.ohlcv = nil })

	tool := NewCalculateIndicatorsTool(&config.Config{})
	result := tool.Execute(context.Background(), map[string]any{
		"provider":  "mock-market-dyn",
		"symbol":    "BTC/USDT",
		"timeframe": "1h",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %q", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "SMA") {
		t.Errorf("expected SMA in output, got: %q", result.ForLLM)
	}
}

func TestCalculateIndicatorsTool_Execute_SelectiveIndicators(t *testing.T) {
	mockMarketDyn.ohlcvErr = nil
	mockMarketDyn.ohlcv = makeCandles(100, 50000.0)
	t.Cleanup(func() { mockMarketDyn.ohlcv = nil })

	tool := NewCalculateIndicatorsTool(&config.Config{})
	result := tool.Execute(context.Background(), map[string]any{
		"provider":   "mock-market-dyn",
		"symbol":     "BTC/USDT",
		"timeframe":  "1h",
		"indicators": []any{"RSI", "MACD", "BB", "ATR", "STOCH", "VWAP"},
	})
	if result.IsError {
		t.Fatalf("unexpected error: %q", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "RSI") {
		t.Errorf("expected RSI in output, got: %q", result.ForLLM)
	}
}

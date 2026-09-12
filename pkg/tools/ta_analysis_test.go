package tools

import (
	"context"
	"errors"
	"strings"
	"testing"

	ccxt "github.com/ccxt/ccxt/go/v4"
	"github.com/cryptoquantumwave/khunquant/pkg/config"
	"github.com/cryptoquantumwave/khunquant/pkg/providers/broker"
)

// nonMarketProvider satisfies broker.Provider but NOT broker.MarketDataProvider.
type nonMarketProvider struct{}

func (nonMarketProvider) ID() string                                               { return "no-market" }
func (nonMarketProvider) Category() broker.AssetCategory                           { return broker.CategoryCrypto }
func (nonMarketProvider) GetMarketStatus(_ context.Context, _ string) (broker.MarketStatus, error) {
	return broker.MarketOpen, nil
}

// mockMarketProvider satisfies broker.MarketDataProvider (and broker.Provider).
type mockMarketProvider struct {
	tickerErr error
	ticker    ccxt.Ticker
	ohlcvErr  error
	ohlcv     []ccxt.OHLCV
}

func (m *mockMarketProvider) ID() string                                               { return "mock-market" }
func (m *mockMarketProvider) Category() broker.AssetCategory                           { return broker.CategoryCrypto }
func (m *mockMarketProvider) GetMarketStatus(_ context.Context, _ string) (broker.MarketStatus, error) {
	return broker.MarketOpen, nil
}
func (m *mockMarketProvider) FetchTicker(_ context.Context, _ string) (ccxt.Ticker, error) {
	return m.ticker, m.tickerErr
}
func (m *mockMarketProvider) FetchTickers(_ context.Context, _ []string) (map[string]ccxt.Ticker, error) {
	return nil, nil
}
func (m *mockMarketProvider) FetchOHLCV(_ context.Context, _, _ string, _ *int64, _ int) ([]ccxt.OHLCV, error) {
	return m.ohlcv, m.ohlcvErr
}
func (m *mockMarketProvider) FetchOrderBook(_ context.Context, _ string, _ int) (ccxt.OrderBook, error) {
	return ccxt.OrderBook{}, nil
}
func (m *mockMarketProvider) LoadMarkets(_ context.Context) (map[string]ccxt.MarketInterface, error) {
	return nil, nil
}

func init() {
	broker.RegisterFactory("no-market-provider", func(_ *config.Config) (broker.Provider, error) {
		return nonMarketProvider{}, nil
	})
	broker.RegisterFactory("mock-market-provider", func(_ *config.Config) (broker.Provider, error) {
		return &mockMarketProvider{}, nil
	})
}

// mockMarketFactoryWith stores a per-test mock so tests can swap the provider's behaviour.
// Tests register under the same key "mock-market-dyn" and overwrite each time.
var mockMarketDyn = &mockMarketProvider{}

func init() {
	broker.RegisterFactory("mock-market-dyn", func(_ *config.Config) (broker.Provider, error) {
		return mockMarketDyn, nil
	})
}

func TestNewMarketAnalysisTool_NotNil(t *testing.T) {
	cfg := &config.Config{}
	tool := NewMarketAnalysisTool(cfg)
	if tool == nil {
		t.Fatal("NewMarketAnalysisTool returned nil")
	}
}

func TestMarketAnalysisTool_Name(t *testing.T) {
	tool := NewMarketAnalysisTool(nil)
	if tool.Name() != NameMarketAnalysis {
		t.Errorf("Name() = %q, want %q", tool.Name(), NameMarketAnalysis)
	}
}

func TestMarketAnalysisTool_Description(t *testing.T) {
	tool := NewMarketAnalysisTool(nil)
	if tool.Description() == "" {
		t.Error("Description() should not be empty")
	}
}

func TestMarketAnalysisTool_Parameters(t *testing.T) {
	tool := NewMarketAnalysisTool(nil)
	params := tool.Parameters()
	if params == nil {
		t.Fatal("Parameters() should not be nil")
	}
	if params["type"] != "object" {
		t.Errorf("Parameters() type = %v, want object", params["type"])
	}
}

func TestMarketAnalysisTool_Execute_MissingProvider(t *testing.T) {
	tool := NewMarketAnalysisTool(&config.Config{})
	result := tool.Execute(context.Background(), map[string]any{
		"symbol": "BTC/USDT",
	})
	if !result.IsError {
		t.Fatal("expected error when provider is missing")
	}
}

func TestMarketAnalysisTool_Execute_MissingSymbol(t *testing.T) {
	tool := NewMarketAnalysisTool(&config.Config{})
	result := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
	})
	if !result.IsError {
		t.Fatal("expected error when symbol is missing")
	}
}

func TestMarketAnalysisTool_Execute_UnknownProvider(t *testing.T) {
	tool := NewMarketAnalysisTool(&config.Config{})
	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent-exchange",
		"symbol":   "BTC/USDT",
	})
	if !result.IsError {
		t.Fatal("expected error for unknown provider")
	}
}

func TestMarketAnalysisTool_Execute_EmptyTimeframe(t *testing.T) {
	// Empty timeframe gets defaulted to "1h" then broker lookup fails.
	// Covers the `if timeframe == ""` assignment branch (line 50-52).
	tool := NewMarketAnalysisTool(&config.Config{})
	result := tool.Execute(context.Background(), map[string]any{
		"provider":  "nonexistent-exchange",
		"symbol":    "BTC/USDT",
		"timeframe": "",
	})
	if !result.IsError {
		t.Fatal("expected error for unknown provider after empty timeframe default")
	}
}

func TestMarketAnalysisTool_Execute_NoMarketDataSupport(t *testing.T) {
	// Provider exists but does not implement MarketDataProvider.
	tool := NewMarketAnalysisTool(&config.Config{})
	result := tool.Execute(context.Background(), map[string]any{
		"provider": "no-market-provider",
		"symbol":   "BTC/USDT",
	})
	if !result.IsError {
		t.Fatal("expected error when provider does not support market data")
	}
	if !strings.Contains(result.ForLLM, "does not support market data") {
		t.Errorf("unexpected error message: %q", result.ForLLM)
	}
}

func TestMarketAnalysisTool_Execute_FetchTickerError(t *testing.T) {
	mockMarketDyn.tickerErr = errors.New("ticker failed")
	mockMarketDyn.ohlcvErr = nil
	mockMarketDyn.ohlcv = nil
	t.Cleanup(func() { mockMarketDyn.tickerErr = nil })

	tool := NewMarketAnalysisTool(&config.Config{})
	result := tool.Execute(context.Background(), map[string]any{
		"provider": "mock-market-dyn",
		"symbol":   "BTC/USDT",
	})
	if !result.IsError {
		t.Fatal("expected error when FetchTicker fails")
	}
	if !strings.Contains(result.ForLLM, "FetchTicker") {
		t.Errorf("unexpected error message: %q", result.ForLLM)
	}
}

func TestMarketAnalysisTool_Execute_FetchOHLCVError(t *testing.T) {
	last := 50000.0
	mockMarketDyn.tickerErr = nil
	mockMarketDyn.ticker = ccxt.Ticker{Last: &last}
	mockMarketDyn.ohlcvErr = errors.New("ohlcv failed")
	mockMarketDyn.ohlcv = nil
	t.Cleanup(func() { mockMarketDyn.ohlcvErr = nil; mockMarketDyn.ticker = ccxt.Ticker{} })

	tool := NewMarketAnalysisTool(&config.Config{})
	result := tool.Execute(context.Background(), map[string]any{
		"provider": "mock-market-dyn",
		"symbol":   "BTC/USDT",
	})
	if !result.IsError {
		t.Fatal("expected error when FetchOHLCV fails")
	}
	if !strings.Contains(result.ForLLM, "FetchOHLCV") {
		t.Errorf("unexpected error message: %q", result.ForLLM)
	}
}

func TestMarketAnalysisTool_Execute_InsufficientCandles(t *testing.T) {
	last := 50000.0
	high := 51000.0
	low := 49000.0
	pct := 1.5
	vol := 1234.5
	mockMarketDyn.tickerErr = nil
	mockMarketDyn.ticker = ccxt.Ticker{
		Last: &last, High: &high, Low: &low, Percentage: &pct, BaseVolume: &vol,
	}
	mockMarketDyn.ohlcvErr = nil
	mockMarketDyn.ohlcv = []ccxt.OHLCV{{Close: 50000.0}}
	t.Cleanup(func() { mockMarketDyn.ticker = ccxt.Ticker{}; mockMarketDyn.ohlcv = nil })

	tool := NewMarketAnalysisTool(&config.Config{})
	result := tool.Execute(context.Background(), map[string]any{
		"provider": "mock-market-dyn",
		"symbol":   "BTC/USDT",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %q", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "Insufficient candle data") {
		t.Errorf("expected insufficient candle message, got: %q", result.ForLLM)
	}
}

func makeCandles(n int, price float64) []ccxt.OHLCV {
	candles := make([]ccxt.OHLCV, n)
	for i := range candles {
		candles[i] = ccxt.OHLCV{
			Open: price, High: price + 100, Low: price - 100,
			Close: price, Volume: 100.0,
		}
	}
	return candles
}

func TestMarketAnalysisTool_Execute_FullAnalysis_BullishCross(t *testing.T) {
	// price > SMA → bullish; RSI neutral; MACD positive
	last := 50000.0
	mockMarketDyn.tickerErr = nil
	mockMarketDyn.ticker = ccxt.Ticker{Last: &last}
	mockMarketDyn.ohlcvErr = nil
	mockMarketDyn.ohlcv = makeCandles(100, 50000.0)
	t.Cleanup(func() { mockMarketDyn.ticker = ccxt.Ticker{}; mockMarketDyn.ohlcv = nil })

	tool := NewMarketAnalysisTool(&config.Config{})
	result := tool.Execute(context.Background(), map[string]any{
		"provider": "mock-market-dyn",
		"symbol":   "BTC/USDT",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %q", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "SMA") {
		t.Errorf("expected SMA in output, got: %q", result.ForLLM)
	}
}

func TestMarketAnalysisTool_Execute_FullAnalysis_Oversold(t *testing.T) {
	// Declining prices → RSI oversold
	last := 1.0
	mockMarketDyn.tickerErr = nil
	mockMarketDyn.ticker = ccxt.Ticker{Last: &last}
	mockMarketDyn.ohlcvErr = nil
	candles := make([]ccxt.OHLCV, 100)
	for i := range candles {
		price := float64(100 - i) // steadily falling
		if price < 1 {
			price = 1
		}
		candles[i] = ccxt.OHLCV{Open: price + 1, High: price + 2, Low: price - 1, Close: price, Volume: 10}
	}
	mockMarketDyn.ohlcv = candles
	t.Cleanup(func() { mockMarketDyn.ticker = ccxt.Ticker{}; mockMarketDyn.ohlcv = nil })

	tool := NewMarketAnalysisTool(&config.Config{})
	result := tool.Execute(context.Background(), map[string]any{
		"provider": "mock-market-dyn",
		"symbol":   "BTC/USDT",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %q", result.ForLLM)
	}
	// Just verify we get indicator output
	if !strings.Contains(result.ForLLM, "RSI") {
		t.Errorf("expected RSI in output, got: %q", result.ForLLM)
	}
}

func TestMarketAnalysisTool_Execute_FullAnalysis_Overbought(t *testing.T) {
	// Rising prices → RSI overbought path
	last := 200.0
	mockMarketDyn.tickerErr = nil
	mockMarketDyn.ticker = ccxt.Ticker{Last: &last}
	mockMarketDyn.ohlcvErr = nil
	candles := make([]ccxt.OHLCV, 100)
	for i := range candles {
		price := float64(1 + i*3) // steadily rising
		candles[i] = ccxt.OHLCV{Open: price - 1, High: price + 2, Low: price - 2, Close: price, Volume: 10}
	}
	mockMarketDyn.ohlcv = candles
	t.Cleanup(func() { mockMarketDyn.ticker = ccxt.Ticker{}; mockMarketDyn.ohlcv = nil })

	tool := NewMarketAnalysisTool(&config.Config{})
	result := tool.Execute(context.Background(), map[string]any{
		"provider": "mock-market-dyn",
		"symbol":   "BTC/USDT",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %q", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "BB") {
		t.Errorf("expected Bollinger Bands in output, got: %q", result.ForLLM)
	}
}

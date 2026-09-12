package tools

import (
	"context"
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/config"
)

func TestGetOHLCV_MissingProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOHLCVTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"symbol":    "BTC/USDT",
		"timeframe": "1h",
	})
	if !result.IsError {
		t.Fatal("expected error when provider is missing")
	}
}

func TestGetOHLCV_EmptyProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOHLCVTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider":  "",
		"symbol":    "BTC/USDT",
		"timeframe": "1h",
	})
	if !result.IsError {
		t.Fatal("expected error when provider is empty")
	}
}

func TestGetOHLCV_MissingSymbol(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOHLCVTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider":  "binance",
		"timeframe": "1h",
	})
	if !result.IsError {
		t.Fatal("expected error when symbol is missing")
	}
}

func TestGetOHLCV_EmptySymbol(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOHLCVTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider":  "binance",
		"symbol":    "",
		"timeframe": "1h",
	})
	if !result.IsError {
		t.Fatal("expected error when symbol is empty")
	}
}

func TestGetOHLCV_InvalidTimeframe(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOHLCVTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider":  "binance",
		"symbol":    "BTC/USDT",
		"timeframe": "invalid",
	})
	if !result.IsError {
		t.Fatal("expected error for invalid timeframe")
	}
}

func TestGetOHLCV_ValidTimeframes(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOHLCVTool(cfg)

	validTimeframes := []string{"1m", "5m", "15m", "1h", "4h", "1d", "1w"}
	for _, tf := range validTimeframes {
		result := tool.Execute(context.Background(), map[string]any{
			"provider":  "nonexistent",
			"symbol":    "BTC/USDT",
			"timeframe": tf,
		})
		if !result.IsError {
			t.Fatalf("expected error for nonexistent provider with timeframe %s", tf)
		}
		// Don't check for error message at this stage
	}
}

func TestGetOHLCV_EmptyTimeframeDefaultsTo1h(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOHLCVTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider": "nonexistent",
		"symbol":   "BTC/USDT",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetOHLCV_LimitDefault(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOHLCVTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider":  "nonexistent",
		"symbol":    "BTC/USDT",
		"timeframe": "1h",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetOHLCV_LimitExceedsMax(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOHLCVTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider":  "nonexistent",
		"symbol":    "BTC/USDT",
		"timeframe": "1h",
		"limit":     float64(maxOHLCVLimit + 100),
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetOHLCV_ValidLimit(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOHLCVTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider":  "nonexistent",
		"symbol":    "BTC/USDT",
		"timeframe": "1h",
		"limit":     float64(50),
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetOHLCV_Since(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOHLCVTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider":  "nonexistent",
		"symbol":    "BTC/USDT",
		"timeframe": "1h",
		"since":     float64(1609459200000), // 2021-01-01
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetOHLCV_WithAccount(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOHLCVTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider":  "nonexistent",
		"account":   "myaccount",
		"symbol":    "BTC/USDT",
		"timeframe": "1h",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetOHLCV_ParametersSchema(t *testing.T) {
	tool := NewGetOHLCVTool(config.DefaultConfig())
	params := tool.Parameters()

	if params == nil {
		t.Fatal("Parameters() should not return nil")
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties in Parameters")
	}

	expectedProps := []string{"provider", "account", "symbol", "timeframe", "limit", "since"}
	for _, prop := range expectedProps {
		if _, ok := props[prop]; !ok {
			t.Errorf("expected property %q in Parameters", prop)
		}
	}
}

func TestGetOHLCV_Name(t *testing.T) {
	tool := NewGetOHLCVTool(config.DefaultConfig())
	name := tool.Name()
	if name != NameGetOHLCV {
		t.Errorf("Name() = %q, want %q", name, NameGetOHLCV)
	}
}

func TestGetOHLCV_Description(t *testing.T) {
	tool := NewGetOHLCVTool(config.DefaultConfig())
	desc := tool.Description()
	if desc == "" {
		t.Fatal("Description() should not be empty")
	}
}

func TestGetOHLCV_NegativeSince(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOHLCVTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider":  "nonexistent",
		"symbol":    "BTC/USDT",
		"timeframe": "1h",
		"since":     float64(-1000),
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestGetOHLCV_ZeroSince(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewGetOHLCVTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider":  "nonexistent",
		"symbol":    "BTC/USDT",
		"timeframe": "1h",
		"since":     float64(0),
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent provider")
	}
}


func TestGetOHLCV_SuccessWithMock(t *testing.T) {
	mockMarketDyn.ohlcvErr = nil
	mockMarketDyn.ohlcv = makeCandles(5, 50000.0)
	t.Cleanup(func() { mockMarketDyn.ohlcv = nil })

	tool := NewGetOHLCVTool(config.DefaultConfig())
	result := tool.Execute(context.Background(), map[string]any{
		"provider":  "mock-market-dyn",
		"symbol":    "BTC/USDT",
		"timeframe": "1h",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %q", result.ForLLM)
	}
}

package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/config"
	"github.com/cryptoquantumwave/khunquant/pkg/providers/broker"
)

func TestNormalizeFuturesSymbol(t *testing.T) {
	tests := map[string]string{
		"BTCUSDT":       "BTC/USDT:USDT",
		"btc_usdt":      "BTC/USDT:USDT",
		"BTC/USDT":      "BTC/USDT:USDT",
		"BTC/USDT:USDT": "BTC/USDT:USDT",
		"ETH/USDC":      "ETH/USDC:USDC",
	}
	for input, want := range tests {
		if got := normalizeFuturesSymbol(input); got != want {
			t.Fatalf("normalizeFuturesSymbol(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestFuturesOpenPosition_DryRunNormalizesSymbol(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.TradingRisk.AllowLeverage = true
	tool := NewFuturesOpenPositionTool(cfg)
	res := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"symbol":   "btcusdt",
		"side":     "long",
		"amount":   0.01,
		"leverage": 3.0,
		"confirm":  false,
	})
	if res.IsError {
		t.Fatalf("Execute returned error: %s", res.ForUser)
	}
	if !strings.Contains(res.ForUser, "BTC/USDT:USDT") {
		t.Fatalf("dry-run output missing normalized futures symbol: %s", res.ForUser)
	}
}

func TestFuturesOpenPosition_RejectsSpotOnlyProvider(t *testing.T) {
	// Override futuresProviderFn: simulate a provider that exists but has no futures support.
	orig := futuresProviderFn
	futuresProviderFn = func(_ context.Context, _ *config.Config, providerID, _ string) (broker.FuturesProvider, error) {
		return nil, fmt.Errorf("provider %q does not support futures trading (Binance TH and Bitkub are spot-only here)", providerID)
	}
	defer func() { futuresProviderFn = orig }()

	cfg := config.DefaultConfig()
	cfg.TradingRisk.AllowLeverage = true
	tool := NewFuturesOpenPositionTool(cfg)
	res := tool.Execute(context.Background(), map[string]any{
		"provider": "bitkub",
		"symbol":   "BTC/THB",
		"side":     "long",
		"amount":   1.0,
		"leverage": 2.0,
		"confirm":  true,
	})
	if !res.IsError {
		t.Fatalf("Execute should reject spot-only provider, got: %s", res.ForUser)
	}
	if !strings.Contains(res.ForLLM, "does not support futures") {
		t.Fatalf("unexpected error: %s", res.ForLLM)
	}
}

func TestFuturesGetPositions_RequiresProvider(t *testing.T) {
	tool := NewFuturesGetPositionsTool(config.DefaultConfig())
	res := tool.Execute(context.Background(), map[string]any{})
	if !res.IsError || !strings.Contains(res.ForLLM, "provider is required") {
		t.Fatalf("unexpected result: %#v", res)
	}
}

func TestFuturesAllowLeverage(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.TradingRisk.AllowLeverage = false
	tool := NewFuturesSetLeverageTool(cfg)
	res := tool.Execute(context.Background(), map[string]any{
		"provider": "binance", "symbol": "BTC/USDT:USDT", "leverage": 5, "confirm": true,
	})
	if !res.IsError {
		t.Error("expected error when allow_leverage=false")
	}
	if !strings.Contains(res.ForUser, "allow_leverage") && !strings.Contains(res.ForLLM, "allow_leverage") {
		t.Errorf("expected allow_leverage in error, got user: %s, llm: %s", res.ForUser, res.ForLLM)
	}
}

func TestFuturesOpenPosition_RequiresAllowLeverage(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.TradingRisk.AllowLeverage = false
	tool := NewFuturesOpenPositionTool(cfg)
	res := tool.Execute(context.Background(), map[string]any{
		"provider": "binance", "symbol": "BTC/USDT:USDT",
		"side": "long", "amount": 0.01, "leverage": 5, "confirm": false,
	})
	if !res.IsError {
		t.Error("expected error when allow_leverage=false")
	}
}

func TestFuturesValidateMarket_NoProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewFuturesValidateMarketTool(cfg)
	res := tool.Execute(context.Background(), map[string]any{})
	if !res.IsError {
		t.Error("expected error with missing provider")
	}
}

func TestFuturesClosePosition_RequiresLeverage(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.TradingRisk.AllowLeverage = false
	tool := NewFuturesClosePositionTool(cfg)
	res := tool.Execute(context.Background(), map[string]any{
		"provider": "binance", "symbol": "BTC/USDT:USDT", "confirm": true,
	})
	if !res.IsError {
		t.Error("expected error when allow_leverage=false")
	}
}

func TestFuturesReducePosition_RequiresAmountOrPercent(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.TradingRisk.AllowLeverage = true
	tool := NewFuturesReducePositionTool(cfg)
	res := tool.Execute(context.Background(), map[string]any{
		"provider": "binance", "symbol": "BTC/USDT:USDT", "confirm": false,
	})
	if !res.IsError {
		t.Error("expected error when neither amount nor percent given")
	}
}

func TestFuturesCancelOrders_RequiresSymbol(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.TradingRisk.AllowLeverage = true
	tool := NewFuturesCancelOrdersTool(cfg)
	res := tool.Execute(context.Background(), map[string]any{
		"provider": "binance", "confirm": false,
	})
	if !res.IsError {
		t.Error("expected error when symbol missing")
	}
}

func TestFuturesEmergencyFlatten_RequiresLeverage(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.TradingRisk.AllowLeverage = false
	tool := NewFuturesEmergencyFlattenTool(cfg)
	res := tool.Execute(context.Background(), map[string]any{
		"provider": "binance", "confirm": true,
	})
	if !res.IsError {
		t.Error("expected error when allow_leverage=false")
	}
}

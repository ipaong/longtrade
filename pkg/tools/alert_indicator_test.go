package tools

import (
	"context"
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/config"
)

func TestSetIndicatorAlert_MissingAction(t *testing.T) {
	cfg := config.DefaultConfig()
	cronSvc := newTestCronServiceForAlerts(t)
	tool := NewSetIndicatorAlertTool(cfg, cronSvc)

	result := tool.Execute(context.Background(), map[string]any{})
	if !result.IsError {
		t.Fatal("expected error when action is missing")
	}
}

func TestSetIndicatorAlert_UnknownAction(t *testing.T) {
	cfg := config.DefaultConfig()
	cronSvc := newTestCronServiceForAlerts(t)
	tool := NewSetIndicatorAlertTool(cfg, cronSvc)

	result := tool.Execute(context.Background(), map[string]any{
		"action": "unknown",
	})
	if !result.IsError {
		t.Fatal("expected error for unknown action")
	}
}

func TestSetIndicatorAlert_ListAction(t *testing.T) {
	cfg := config.DefaultConfig()
	cronSvc := newTestCronServiceForAlerts(t)
	tool := NewSetIndicatorAlertTool(cfg, cronSvc)

	result := tool.Execute(context.Background(), map[string]any{
		"action": "list",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
}

func TestSetIndicatorAlert_CancelMissingID(t *testing.T) {
	cfg := config.DefaultConfig()
	cronSvc := newTestCronServiceForAlerts(t)
	tool := NewSetIndicatorAlertTool(cfg, cronSvc)

	result := tool.Execute(context.Background(), map[string]any{
		"action": "cancel",
	})
	if !result.IsError {
		t.Fatal("expected error when alert_id is missing")
	}
}

func TestSetIndicatorAlert_CreateMissingProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	cronSvc := newTestCronServiceForAlerts(t)
	tool := NewSetIndicatorAlertTool(cfg, cronSvc)

	result := tool.Execute(context.Background(), map[string]any{
		"action":    "create",
		"symbol":    "BTC/USDT",
		"indicator": "RSI",
		"condition": "above",
		"threshold": 70.0,
	})
	if !result.IsError {
		t.Fatal("expected error when provider is missing")
	}
}

func TestSetIndicatorAlert_CreateMissingSymbol(t *testing.T) {
	cfg := config.DefaultConfig()
	cronSvc := newTestCronServiceForAlerts(t)
	tool := NewSetIndicatorAlertTool(cfg, cronSvc)

	result := tool.Execute(context.Background(), map[string]any{
		"action":    "create",
		"provider":  "binance",
		"indicator": "RSI",
		"condition": "above",
		"threshold": 70.0,
	})
	if !result.IsError {
		t.Fatal("expected error when symbol is missing")
	}
}

func TestSetIndicatorAlert_CreateMissingIndicator(t *testing.T) {
	cfg := config.DefaultConfig()
	cronSvc := newTestCronServiceForAlerts(t)
	tool := NewSetIndicatorAlertTool(cfg, cronSvc)

	result := tool.Execute(context.Background(), map[string]any{
		"action":    "create",
		"provider":  "binance",
		"symbol":    "BTC/USDT",
		"condition": "above",
		"threshold": 70.0,
	})
	if !result.IsError {
		t.Fatal("expected error when indicator is missing")
	}
}

func TestSetIndicatorAlert_CreateMissingCondition(t *testing.T) {
	cfg := config.DefaultConfig()
	cronSvc := newTestCronServiceForAlerts(t)
	tool := NewSetIndicatorAlertTool(cfg, cronSvc)

	result := tool.Execute(context.Background(), map[string]any{
		"action":    "create",
		"provider":  "binance",
		"symbol":    "BTC/USDT",
		"indicator": "RSI",
		"threshold": 70.0,
	})
	if !result.IsError {
		t.Fatal("expected error when condition is missing")
	}
}

func TestSetIndicatorAlert_CreateDefaultTimeframe(t *testing.T) {
	cfg := config.DefaultConfig()
	cronSvc := newTestCronServiceForAlerts(t)
	tool := NewSetIndicatorAlertTool(cfg, cronSvc)

	result := tool.Execute(WithToolContext(context.Background(), "test", "chat-1"), map[string]any{
		"action":    "create",
		"provider":  "binance",
		"symbol":    "BTC/USDT",
		"indicator": "RSI",
		"condition": "above",
		"threshold": 70.0,
	})
	// Should default timeframe to 1h and create
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
}

func TestSetIndicatorAlert_CreateValidIndicators(t *testing.T) {
	cfg := config.DefaultConfig()
	cronSvc := newTestCronServiceForAlerts(t)
	tool := NewSetIndicatorAlertTool(cfg, cronSvc)

	validIndicators := []string{"RSI", "MACD", "SMA20", "EMA9"}
	for _, ind := range validIndicators {
		result := tool.Execute(WithToolContext(context.Background(), "test", "chat-1"), map[string]any{
			"action":    "create",
			"provider":  "binance",
			"symbol":    "BTC/USDT",
			"indicator": ind,
			"condition": "above",
			"threshold": 50.0,
		})
		if result.IsError {
			t.Fatalf("unexpected error for indicator %s: %s", ind, result.ForLLM)
		}
	}
}

func TestSetIndicatorAlert_CreateValidTimeframes(t *testing.T) {
	cfg := config.DefaultConfig()
	cronSvc := newTestCronServiceForAlerts(t)
	tool := NewSetIndicatorAlertTool(cfg, cronSvc)

	validTimeframes := []string{"1m", "5m", "15m", "1h", "4h", "1d", "1w"}
	for _, tf := range validTimeframes {
		result := tool.Execute(WithToolContext(context.Background(), "test", "chat-1"), map[string]any{
			"action":    "create",
			"provider":  "binance",
			"symbol":    "BTC/USDT",
			"indicator": "RSI",
			"timeframe": tf,
			"condition": "above",
			"threshold": 70.0,
		})
		if result.IsError {
			t.Fatalf("unexpected error for timeframe %s: %s", tf, result.ForLLM)
		}
	}
}

func TestSetIndicatorAlert_CreateValidConditions(t *testing.T) {
	cfg := config.DefaultConfig()
	cronSvc := newTestCronServiceForAlerts(t)
	tool := NewSetIndicatorAlertTool(cfg, cronSvc)

	validConditions := []string{"above", "below"}
	for _, cond := range validConditions {
		result := tool.Execute(WithToolContext(context.Background(), "test", "chat-1"), map[string]any{
			"action":    "create",
			"provider":  "binance",
			"symbol":    "BTC/USDT",
			"indicator": "RSI",
			"condition": cond,
			"threshold": 70.0,
		})
		if result.IsError {
			t.Fatalf("unexpected error for condition %s: %s", cond, result.ForLLM)
		}
	}
}

func TestSetIndicatorAlert_ParametersSchema(t *testing.T) {
	tool := NewSetIndicatorAlertTool(config.DefaultConfig(), newTestCronServiceForAlerts(t))
	params := tool.Parameters()

	if params == nil {
		t.Fatal("Parameters() should not return nil")
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties in Parameters")
	}

	expectedProps := []string{"action", "provider", "symbol", "indicator", "timeframe", "condition", "threshold", "message", "channel", "chat_id", "recurring", "alert_id", "account"}
	for _, prop := range expectedProps {
		if _, ok := props[prop]; !ok {
			t.Errorf("expected property %q in Parameters", prop)
		}
	}
}

func TestSetIndicatorAlert_Name(t *testing.T) {
	tool := NewSetIndicatorAlertTool(config.DefaultConfig(), newTestCronServiceForAlerts(t))
	name := tool.Name()
	if name != NameSetIndicatorAlert {
		t.Errorf("Name() = %q, want %q", name, NameSetIndicatorAlert)
	}
}

func TestSetIndicatorAlert_Description(t *testing.T) {
	tool := NewSetIndicatorAlertTool(config.DefaultConfig(), newTestCronServiceForAlerts(t))
	desc := tool.Description()
	if desc == "" {
		t.Fatal("Description() should not be empty")
	}
}

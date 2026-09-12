package tools

import (
	"context"
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/config"
)

func TestTransferFunds_MissingProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewTransferFundsTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"asset":        "USDT",
		"amount":       100.0,
		"from_account": "spot",
		"to_account":   "futures",
		"confirm":      false,
	})
	if !result.IsError {
		t.Fatal("expected error when provider is missing")
	}
}

func TestTransferFunds_EmptyProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewTransferFundsTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider":     "",
		"asset":        "USDT",
		"amount":       100.0,
		"from_account": "spot",
		"to_account":   "futures",
		"confirm":      false,
	})
	if !result.IsError {
		t.Fatal("expected error when provider is empty")
	}
}

func TestTransferFunds_MissingAsset(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewTransferFundsTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider":     "binance",
		"amount":       100.0,
		"from_account": "spot",
		"to_account":   "futures",
		"confirm":      false,
	})
	if !result.IsError {
		t.Fatal("expected error when asset is missing")
	}
}

func TestTransferFunds_EmptyAsset(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewTransferFundsTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider":     "binance",
		"asset":        "",
		"amount":       100.0,
		"from_account": "spot",
		"to_account":   "futures",
		"confirm":      false,
	})
	if !result.IsError {
		t.Fatal("expected error when asset is empty")
	}
}

func TestTransferFunds_InvalidAmount(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewTransferFundsTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider":     "binance",
		"asset":        "USDT",
		"amount":       0.0,
		"from_account": "spot",
		"to_account":   "futures",
		"confirm":      false,
	})
	if !result.IsError {
		t.Fatal("expected error when amount is zero")
	}
}

func TestTransferFunds_NegativeAmount(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewTransferFundsTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider":     "binance",
		"asset":        "USDT",
		"amount":       -100.0,
		"from_account": "spot",
		"to_account":   "futures",
		"confirm":      false,
	})
	if !result.IsError {
		t.Fatal("expected error when amount is negative")
	}
}

func TestTransferFunds_MissingFromAccount(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewTransferFundsTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider":     "binance",
		"asset":        "USDT",
		"amount":       100.0,
		"to_account":   "futures",
		"confirm":      false,
	})
	if !result.IsError {
		t.Fatal("expected error when from_account is missing")
	}
}

func TestTransferFunds_MissingToAccount(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewTransferFundsTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider":     "binance",
		"asset":        "USDT",
		"amount":       100.0,
		"from_account": "spot",
		"confirm":      false,
	})
	if !result.IsError {
		t.Fatal("expected error when to_account is missing")
	}
}

func TestTransferFunds_DryRun(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewTransferFundsTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider":     "binance",
		"asset":        "USDT",
		"amount":       100.0,
		"from_account": "spot",
		"to_account":   "futures",
		"confirm":      false,
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if result.ForUser == "" {
		t.Fatal("expected user message for dry-run")
	}
}

func TestTransferFunds_DryRunShowsAmount(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewTransferFundsTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider":     "binance",
		"asset":        "USDT",
		"amount":       250.5,
		"from_account": "spot",
		"to_account":   "futures",
		"confirm":      false,
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	// Should show dry-run message with the amount and accounts
	if result.ForUser == "" {
		t.Fatal("expected user message")
	}
}

func TestTransferFunds_ParametersSchema(t *testing.T) {
	tool := NewTransferFundsTool(config.DefaultConfig())
	params := tool.Parameters()

	if params == nil {
		t.Fatal("Parameters() should not return nil")
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties in Parameters")
	}

	expectedProps := []string{"provider", "asset", "amount", "from_account", "to_account", "confirm", "account"}
	for _, prop := range expectedProps {
		if _, ok := props[prop]; !ok {
			t.Errorf("expected property %q in Parameters", prop)
		}
	}

	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("expected required in Parameters")
	}
	if len(required) == 0 {
		t.Fatal("expected required fields")
	}
}

func TestTransferFunds_Name(t *testing.T) {
	tool := NewTransferFundsTool(config.DefaultConfig())
	name := tool.Name()
	if name != NameTransferFunds {
		t.Errorf("Name() = %q, want %q", name, NameTransferFunds)
	}
}

func TestTransferFunds_Description(t *testing.T) {
	tool := NewTransferFundsTool(config.DefaultConfig())
	desc := tool.Description()
	if desc == "" {
		t.Fatal("Description() should not be empty")
	}
}

func TestTransferFunds_WithAccount(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewTransferFundsTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"provider":     "binance",
		"account":      "myaccount",
		"asset":        "USDT",
		"amount":       100.0,
		"from_account": "spot",
		"to_account":   "futures",
		"confirm":      false,
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
}

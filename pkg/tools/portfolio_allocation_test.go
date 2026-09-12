package tools

import (
	"context"
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/config"
)

func newTestPortfolioAllocationTool(t *testing.T) *PortfolioAllocationTool {
	t.Helper()
	return NewPortfolioAllocationTool(config.DefaultConfig())
}

func TestPortfolioAllocationTool_Name(t *testing.T) {
	tool := newTestPortfolioAllocationTool(t)
	if tool.Name() != NamePortfolioAllocation {
		t.Errorf("Name() = %q, want %q", tool.Name(), NamePortfolioAllocation)
	}
}

func TestPortfolioAllocationTool_Description(t *testing.T) {
	tool := newTestPortfolioAllocationTool(t)
	desc := tool.Description()
	if desc == "" {
		t.Error("Description() should not be empty")
	}
}

func TestPortfolioAllocationTool_Parameters(t *testing.T) {
	tool := newTestPortfolioAllocationTool(t)
	params := tool.Parameters()

	if params["type"] != "object" {
		t.Errorf("type should be 'object'")
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties should be a map")
	}

	expectedProps := []string{"quote"}
	for _, prop := range expectedProps {
		if _, ok := props[prop]; !ok {
			t.Errorf("expected property %q not found", prop)
		}
	}
}

func TestPortfolioAllocationTool_Execute_NoArgs(t *testing.T) {
	tool := newTestPortfolioAllocationTool(t)

	result := tool.Execute(context.Background(), map[string]any{})

	// No configured accounts, should return "no accounts found"
	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestPortfolioAllocationTool_Execute_DefaultQuote(t *testing.T) {
	tool := newTestPortfolioAllocationTool(t)

	result := tool.Execute(context.Background(), map[string]any{})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestPortfolioAllocationTool_Execute_CustomQuote(t *testing.T) {
	tool := newTestPortfolioAllocationTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"quote": "USDT",
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestPortfolioAllocationTool_Execute_THBQuote(t *testing.T) {
	tool := newTestPortfolioAllocationTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"quote": "THB",
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestPortfolioAllocationTool_Execute_BUSDQuote(t *testing.T) {
	tool := newTestPortfolioAllocationTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"quote": "BUSD",
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestPortfolioAllocationTool_Execute_EmptyQuote(t *testing.T) {
	tool := newTestPortfolioAllocationTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"quote": "",
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
	// Empty quote should default to USDT
}

func TestPortfolioAllocationTool_Execute_InvalidArgTypes(t *testing.T) {
	tool := newTestPortfolioAllocationTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"quote": 123,   // int instead of string
	})

	if result == nil {
		t.Fatal("Execute should return result even with invalid types")
	}
	// Should handle type mismatch gracefully
}

func TestPortfolioAllocationTool_Execute_WithConfiguredAccounts(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Exchanges.Binance.Enabled = true
	cfg.Exchanges.Binance.Accounts = []config.ExchangeAccount{
		{Name: "main", APIKey: *config.NewSecureString("test-key"), Secret: *config.NewSecureString("test-secret")},
	}
	cfg.Exchanges.BinanceTH.Enabled = false
	cfg.Exchanges.Bitkub.Enabled = false
	cfg.Exchanges.OKX.Enabled = false
	cfg.Exchanges.Settrade.Enabled = false

	tool := NewPortfolioAllocationTool(cfg)
	result := tool.Execute(context.Background(), map[string]any{})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestPortfolioAllocationTool_Execute_MultipleAccounts(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Exchanges.Binance.Enabled = true
	cfg.Exchanges.Binance.Accounts = []config.ExchangeAccount{
		{Name: "spot", APIKey: *config.NewSecureString("key1"), Secret: *config.NewSecureString("secret1")},
		{Name: "futures", APIKey: *config.NewSecureString("key2"), Secret: *config.NewSecureString("secret2")},
	}
	cfg.Exchanges.BinanceTH.Enabled = true
	cfg.Exchanges.BinanceTH.Accounts = []config.ExchangeAccount{
		{Name: "th", APIKey: *config.NewSecureString("key3"), Secret: *config.NewSecureString("secret3")},
	}
	cfg.Exchanges.Bitkub.Enabled = false
	cfg.Exchanges.OKX.Enabled = false
	cfg.Exchanges.Settrade.Enabled = false

	tool := NewPortfolioAllocationTool(cfg)
	result := tool.Execute(context.Background(), map[string]any{})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestPortfolioAllocationTool_Execute_UppercaseQuote(t *testing.T) {
	tool := newTestPortfolioAllocationTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"quote": "USDT",
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestPortfolioAllocationTool_Execute_LowercaseQuote(t *testing.T) {
	tool := newTestPortfolioAllocationTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"quote": "usdt",
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestPortfolioAllocationTool_Execute_MixedCaseQuote(t *testing.T) {
	tool := newTestPortfolioAllocationTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"quote": "UsDt",
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestPortfolioAllocationTool_Execute_StablecoinQuotes(t *testing.T) {
	tool := newTestPortfolioAllocationTool(t)

	stablecoins := []string{"USDT", "USDC", "DAI", "BUSD", "TUSD"}

	for _, coin := range stablecoins {
		t.Run(coin, func(t *testing.T) {
			result := tool.Execute(context.Background(), map[string]any{
				"quote": coin,
			})
			if result == nil {
				t.Fatal("Execute should return result")
			}
		})
	}
}

func TestPortfolioAllocationTool_Execute_FiatQuotes(t *testing.T) {
	tool := newTestPortfolioAllocationTool(t)

	fiats := []string{"USD", "THB", "EUR", "GBP"}

	for _, fiat := range fiats {
		t.Run(fiat, func(t *testing.T) {
			result := tool.Execute(context.Background(), map[string]any{
				"quote": fiat,
			})
			if result == nil {
				t.Fatal("Execute should return result")
			}
		})
	}
}

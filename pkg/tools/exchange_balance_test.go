package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/config"
	"github.com/cryptoquantumwave/khunquant/pkg/exchanges"
)

// MockExchange implements exchanges.Exchange for testing
type MockExchange struct {
	getBalancesFn    func(ctx context.Context) ([]exchanges.Balance, error)
	getWalletBalsFn  func(ctx context.Context, walletType string) ([]exchanges.WalletBalance, error)
}

func (m *MockExchange) GetBalances(ctx context.Context) ([]exchanges.Balance, error) {
	if m.getBalancesFn != nil {
		return m.getBalancesFn(ctx)
	}
	return nil, nil
}

func (m *MockExchange) GetWalletBalances(ctx context.Context, walletType string) ([]exchanges.WalletBalance, error) {
	if m.getWalletBalsFn != nil {
		return m.getWalletBalsFn(ctx, walletType)
	}
	return nil, nil
}

func TestExchangeBalance_MissingExchange(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewExchangeBalanceTool(cfg)

	// Empty config, provider lookup will fail
	result := tool.Execute(context.Background(), map[string]any{})
	if !result.IsError {
		t.Fatal("expected error when exchange not configured")
	}
}

func TestExchangeBalance_DefaultExchange(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewExchangeBalanceTool(cfg)

	// Not providing exchange should use "binance" as default, which will fail to create
	result := tool.Execute(context.Background(), map[string]any{})
	if !result.IsError {
		t.Fatal("expected error for unconfigured exchange")
	}
}

func TestExchangeBalance_EmptyWallet(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewExchangeBalanceTool(cfg)

	// All params empty/default
	result := tool.Execute(context.Background(), map[string]any{
		"exchange": "binance", // will fail to find, but we're testing the validation path
	})
	if !result.IsError {
		t.Fatal("expected error for unconfigured exchange")
	}
}

func TestExchangeBalance_InvalidWalletType(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewExchangeBalanceTool(cfg)

	// The tool doesn't validate wallet_type values — they're just passed through
	// But provider creation will fail
	result := tool.Execute(context.Background(), map[string]any{
		"exchange":    "nonexistent",
		"wallet_type": "spot",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent exchange")
	}
}

func TestExchangeBalance_AssetFilter(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewExchangeBalanceTool(cfg)

	// Asset filter is just a string — no validation error expected here
	// Error will come from provider lookup
	result := tool.Execute(context.Background(), map[string]any{
		"exchange": "nonexistent",
		"asset":    "BTC",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent exchange")
	}
}

func TestExchangeBalance_ParametersSchema(t *testing.T) {
	tool := NewExchangeBalanceTool(config.DefaultConfig())
	params := tool.Parameters()

	if params == nil {
		t.Fatal("Parameters() should not return nil")
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties in Parameters")
	}

	// Check that exchange property exists
	if _, ok := props["exchange"]; !ok {
		t.Fatal("expected 'exchange' property in Parameters")
	}

	// Check that wallet_type property exists
	if _, ok := props["wallet_type"]; !ok {
		t.Fatal("expected 'wallet_type' property in Parameters")
	}
}

func TestExchangeBalance_Name(t *testing.T) {
	tool := NewExchangeBalanceTool(config.DefaultConfig())
	name := tool.Name()
	if name != NameGetAssetsList {
		t.Errorf("Name() = %q, want %q", name, NameGetAssetsList)
	}
}

func TestExchangeBalance_Description(t *testing.T) {
	tool := NewExchangeBalanceTool(config.DefaultConfig())
	desc := tool.Description()
	if desc == "" {
		t.Fatal("Description() should not be empty")
	}
	if len(desc) < 20 {
		t.Fatal("Description() too short")
	}
}

func TestExchangeBalance_AccountParameter(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewExchangeBalanceTool(cfg)

	// Account parameter should be accepted but lead to provider lookup error
	result := tool.Execute(context.Background(), map[string]any{
		"exchange": "nonexistent",
		"account":  "myaccount",
	})
	if !result.IsError {
		t.Fatal("expected error for nonexistent exchange even with account param")
	}
}

// basicExchange implements only exchanges.Exchange (not WalletExchange)
// so that fallbackGetBalances is exercised.
type basicExchange struct {
	name     string
	balances []exchanges.Balance
	err      error
}

func (b *basicExchange) Name() string { return b.name }
func (b *basicExchange) GetBalances(ctx context.Context) ([]exchanges.Balance, error) {
	return b.balances, b.err
}

func TestFallbackGetBalances_NoBalances(t *testing.T) {
	tool := NewExchangeBalanceTool(config.DefaultConfig())
	mock := &basicExchange{name: "testex", balances: nil}
	result := tool.fallbackGetBalances(context.Background(), mock, "testex", "", "")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForUser, "No non-zero balances") {
		t.Errorf("expected empty-balance message, got: %s", result.ForUser)
	}
}

func TestFallbackGetBalances_WithBalances(t *testing.T) {
	tool := NewExchangeBalanceTool(config.DefaultConfig())
	mock := &basicExchange{
		name: "myex",
		balances: []exchanges.Balance{
			{Asset: "BTC", Free: 1.5, Locked: 0.0},
			{Asset: "ETH", Free: 10.0, Locked: 0.5},
		},
	}
	result := tool.fallbackGetBalances(context.Background(), mock, "myex", "main", "")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForUser, "BTC") {
		t.Errorf("expected BTC in output, got: %s", result.ForUser)
	}
	if !strings.Contains(result.ForUser, "main") {
		t.Errorf("expected account name in output, got: %s", result.ForUser)
	}
}

func TestFallbackGetBalances_WithAssetFilter(t *testing.T) {
	tool := NewExchangeBalanceTool(config.DefaultConfig())
	mock := &basicExchange{
		name: "myex",
		balances: []exchanges.Balance{
			{Asset: "BTC", Free: 1.0},
			{Asset: "ETH", Free: 2.0},
		},
	}
	result := tool.fallbackGetBalances(context.Background(), mock, "myex", "", "BTC")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForUser, "BTC") {
		t.Errorf("expected BTC in result, got: %s", result.ForUser)
	}
	if strings.Contains(result.ForUser, "ETH") {
		t.Errorf("ETH should be filtered out, got: %s", result.ForUser)
	}
}

func TestFallbackGetBalances_FilterNoMatch(t *testing.T) {
	tool := NewExchangeBalanceTool(config.DefaultConfig())
	mock := &basicExchange{
		name:     "myex",
		balances: []exchanges.Balance{{Asset: "ETH", Free: 1.0}},
	}
	result := tool.fallbackGetBalances(context.Background(), mock, "myex", "", "BTC")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !strings.Contains(result.ForUser, "No non-zero balances") {
		t.Errorf("expected no-balance message, got: %s", result.ForUser)
	}
}

func TestFormatAmount_TrailingZeros(t *testing.T) {
	if got := formatAmount(0.10000000); got != "0.1" {
		t.Errorf("formatAmount(0.1) = %q, want \"0.1\"", got)
	}
}

func TestWriteBalanceTable_Basic(t *testing.T) {
	balances := []exchanges.WalletBalance{
		{Balance: exchanges.Balance{Asset: "BTC", Free: 1.5, Locked: 0.1}},
		{Balance: exchanges.Balance{Asset: "ETH", Free: 10.0, Locked: 0}},
	}
	var sb strings.Builder
	writeBalanceTable(&sb, balances)
	out := sb.String()
	if !strings.Contains(out, "BTC") {
		t.Errorf("expected BTC in output, got: %s", out)
	}
	if !strings.Contains(out, "ETH") {
		t.Errorf("expected ETH in output, got: %s", out)
	}
	if !strings.Contains(out, "1.5") {
		t.Errorf("expected 1.5 in output, got: %s", out)
	}
}

func TestWriteBalanceTable_WithExtra(t *testing.T) {
	balances := []exchanges.WalletBalance{
		{
			Balance: exchanges.Balance{Asset: "BTC", Free: 2.0},
			Extra:   map[string]string{"unrealized_pnl": "150.00"},
		},
	}
	var sb strings.Builder
	writeBalanceTable(&sb, balances)
	out := sb.String()
	if !strings.Contains(out, "unrealized pnl") {
		t.Errorf("expected 'unrealized pnl' column header, got: %s", out)
	}
	if !strings.Contains(out, "150.00") {
		t.Errorf("expected extra value in output, got: %s", out)
	}
}

func TestWriteBalanceTable_ExtraKeyMissing(t *testing.T) {
	// One balance has the extra key, another does not — missing should show "0"
	balances := []exchanges.WalletBalance{
		{Balance: exchanges.Balance{Asset: "BTC"}, Extra: map[string]string{"fee": "10"}},
		{Balance: exchanges.Balance{Asset: "ETH"}, Extra: nil},
	}
	var sb strings.Builder
	writeBalanceTable(&sb, balances)
	out := sb.String()
	if !strings.Contains(out, "fee") {
		t.Errorf("expected fee column, got: %s", out)
	}
}

func TestFormatWalletBalances_SingleType(t *testing.T) {
	balances := []exchanges.WalletBalance{
		{Balance: exchanges.Balance{Asset: "BTC", Free: 1.5, Locked: 0.1}, WalletType: "spot"},
	}
	got := formatWalletBalances("binance", "", "spot", balances)
	if !strings.Contains(got, "BTC") {
		t.Errorf("expected BTC in output, got: %s", got)
	}
	if !strings.Contains(got, "Spot") {
		t.Errorf("expected Spot label, got: %s", got)
	}
}

func TestFormatWalletBalances_WithAccountName(t *testing.T) {
	balances := []exchanges.WalletBalance{
		{Balance: exchanges.Balance{Asset: "USDT", Free: 500.0}, WalletType: "spot"},
	}
	got := formatWalletBalances("binance", "HighRiskPort", "spot", balances)
	if !strings.Contains(got, "HighRiskPort") {
		t.Errorf("expected account name in header, got: %s", got)
	}
}

func TestFormatWalletBalances_AllTypes(t *testing.T) {
	balances := []exchanges.WalletBalance{
		{Balance: exchanges.Balance{Asset: "BTC", Free: 1.0}, WalletType: "spot"},
		{Balance: exchanges.Balance{Asset: "ETH", Free: 2.0}, WalletType: "funding"},
	}
	got := formatWalletBalances("binance", "", "all", balances)
	if !strings.Contains(got, "BTC") {
		t.Errorf("expected BTC in output, got: %s", got)
	}
	if !strings.Contains(got, "ETH") {
		t.Errorf("expected ETH in output, got: %s", got)
	}
	if !strings.Contains(got, "Spot") {
		t.Errorf("expected Spot section, got: %s", got)
	}
	if !strings.Contains(got, "Funding") {
		t.Errorf("expected Funding section, got: %s", got)
	}
}

func TestFormatWalletBalances_AllTypes_UnknownWalletType(t *testing.T) {
	balances := []exchanges.WalletBalance{
		{Balance: exchanges.Balance{Asset: "XYZ", Free: 1.0}, WalletType: "custom_wallet"},
	}
	got := formatWalletBalances("myex", "", "all", balances)
	if !strings.Contains(got, "XYZ") {
		t.Errorf("expected XYZ in output, got: %s", got)
	}
	if !strings.Contains(got, "custom_wallet") {
		t.Errorf("expected unknown wallet type as label, got: %s", got)
	}
}

func TestFormatWalletBalances_SortsByAsset(t *testing.T) {
	balances := []exchanges.WalletBalance{
		{Balance: exchanges.Balance{Asset: "USDT", Free: 100}, WalletType: "spot"},
		{Balance: exchanges.Balance{Asset: "BTC", Free: 1.0}, WalletType: "spot"},
	}
	got := formatWalletBalances("binance", "", "spot", balances)
	btcIdx := strings.Index(got, "BTC")
	usdtIdx := strings.Index(got, "USDT")
	if btcIdx > usdtIdx {
		t.Errorf("expected BTC before USDT (alphabetical sort), got: %s", got)
	}
}

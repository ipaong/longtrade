package tools

import (
	"errors"
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/exchanges"
)

func TestFormatAmount_Zero(t *testing.T) {
	if got := formatAmount(0); got != "0" {
		t.Errorf("formatAmount(0) = %q, want %q", got, "0")
	}
}

func TestFormatAmount_Integer(t *testing.T) {
	if got := formatAmount(1.0); got != "1" {
		t.Errorf("formatAmount(1.0) = %q, want %q", got, "1")
	}
}

func TestFormatAmount_TrimTrailingZeros(t *testing.T) {
	if got := formatAmount(1.5); got != "1.5" {
		t.Errorf("formatAmount(1.5) = %q, want %q", got, "1.5")
	}
}

func TestFormatAmount_Small(t *testing.T) {
	if got := formatAmount(0.00000001); got != "0.00000001" {
		t.Errorf("formatAmount(0.00000001) = %q, want %q", got, "0.00000001")
	}
}

func TestFormatAmount_Large(t *testing.T) {
	if got := formatAmount(50000.0); got != "50000" {
		t.Errorf("formatAmount(50000.0) = %q, want %q", got, "50000")
	}
}

func TestFormatAmount_Decimal(t *testing.T) {
	got := formatAmount(1234.5678)
	if got != "1234.5678" {
		t.Errorf("formatAmount(1234.5678) = %q, want %q", got, "1234.5678")
	}
}

func TestWalletTypeLabel_Known(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"spot", "Spot"},
		{"funding", "Funding"},
		{"futures_usdt", "Futures (USDT-M)"},
		{"futures_coin", "Futures (Coin-M)"},
		{"margin", "Cross Margin"},
		{"earn_flexible", "Simple Earn (Flexible)"},
		{"earn_locked", "Simple Earn (Locked)"},
		{"earn", "Simple Earn"},
	}
	for _, tc := range cases {
		got := walletTypeLabel(tc.input)
		if got != tc.want {
			t.Errorf("walletTypeLabel(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestWalletTypeLabel_Unknown(t *testing.T) {
	got := walletTypeLabel("unknown_type")
	if got == "" {
		t.Error("walletTypeLabel should return non-empty for unknown type")
	}
}

func TestCollectExtraKeys_Empty(t *testing.T) {
	result := collectExtraKeys(nil)
	if len(result) != 0 {
		t.Errorf("collectExtraKeys(nil) = %v, want empty", result)
	}
}

func TestCollectExtraKeys_NoExtras(t *testing.T) {
	balances := []exchanges.WalletBalance{
		{Extra: nil},
		{Extra: map[string]string{}},
	}
	result := collectExtraKeys(balances)
	if len(result) != 0 {
		t.Errorf("collectExtraKeys with no extra keys = %v, want empty", result)
	}
}

func TestCollectExtraKeys_Deduplication(t *testing.T) {
	balances := []exchanges.WalletBalance{
		{Extra: map[string]string{"unrealized_pnl": "100", "borrowed": "0"}},
		{Extra: map[string]string{"unrealized_pnl": "50"}}, // duplicate key
	}
	result := collectExtraKeys(balances)
	if len(result) != 2 {
		t.Errorf("collectExtraKeys dedup: got %d keys, want 2: %v", len(result), result)
	}
}

func TestCollectExtraKeys_Sorted(t *testing.T) {
	balances := []exchanges.WalletBalance{
		{Extra: map[string]string{"zzz": "1", "aaa": "2", "mmm": "3"}},
	}
	result := collectExtraKeys(balances)
	if len(result) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(result))
	}
	if result[0] != "aaa" || result[1] != "mmm" || result[2] != "zzz" {
		t.Errorf("collectExtraKeys not sorted: %v", result)
	}
}

func TestTrimCCXTError_NoStack(t *testing.T) {
	err := errors.New("[ccxt]::[NetworkError]: connection refused")
	got := trimCCXTError(err)
	if got != "[ccxt]::[NetworkError]: connection refused" {
		t.Errorf("trimCCXTError no stack = %q", got)
	}
}

func TestTrimCCXTError_WithStack(t *testing.T) {
	err := errors.New("[ccxt]::[NetworkError]: connection refused\nStack:\n  at line 1\n  at line 2")
	got := trimCCXTError(err)
	if got != "[ccxt]::[NetworkError]: connection refused" {
		t.Errorf("trimCCXTError with stack = %q, want trimmed", got)
	}
}

func TestTrimCCXTError_MultiLine(t *testing.T) {
	err := errors.New("first line\nsecond line")
	got := trimCCXTError(err)
	if got != "first line" {
		t.Errorf("trimCCXTError multiline = %q, want only first line", got)
	}
}

func TestIsNetworkError_True(t *testing.T) {
	cases := []string{
		"NetworkError: failed",
		"no such host: example.com",
		"dial tcp 127.0.0.1:443: refused",
		"connection refused",
		"i/o timeout",
		"network error occurred",
	}
	for _, msg := range cases {
		if !isNetworkError(msg) {
			t.Errorf("isNetworkError(%q) = false, want true", msg)
		}
	}
}

func TestIsNetworkError_False(t *testing.T) {
	cases := []string{
		"invalid symbol",
		"insufficient balance",
		"order not found",
		"",
	}
	for _, msg := range cases {
		if isNetworkError(msg) {
			t.Errorf("isNetworkError(%q) = true, want false", msg)
		}
	}
}

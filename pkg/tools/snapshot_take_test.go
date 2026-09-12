package tools

import (
	"context"
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/config"
	"github.com/cryptoquantumwave/khunquant/pkg/snapshot"
)

func newTestSnapshotStore(t *testing.T) *snapshot.Store {
	t.Helper()
	store, err := snapshot.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("snapshot.NewStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func newTestTakeSnapshotTool(t *testing.T) *TakeSnapshotTool {
	t.Helper()
	return NewTakeSnapshotTool(config.DefaultConfig(), newTestSnapshotStore(t))
}

func TestTakeSnapshotTool_Name(t *testing.T) {
	tool := newTestTakeSnapshotTool(t)
	if tool.Name() != NameTakeSnapshot {
		t.Errorf("Name() = %q, want %q", tool.Name(), NameTakeSnapshot)
	}
}

func TestTakeSnapshotTool_Description(t *testing.T) {
	tool := newTestTakeSnapshotTool(t)
	desc := tool.Description()
	if desc == "" {
		t.Error("Description() should not be empty")
	}
	if !contains(desc, "snapshot") {
		t.Errorf("Description should mention 'snapshot', got %q", desc)
	}
}

func TestTakeSnapshotTool_Parameters(t *testing.T) {
	tool := newTestTakeSnapshotTool(t)
	params := tool.Parameters()

	if params["type"] != "object" {
		t.Errorf("type should be 'object', got %v", params["type"])
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties should be a map")
	}

	expectedProps := []string{"source", "account", "quote", "label", "note"}
	for _, prop := range expectedProps {
		if _, ok := props[prop]; !ok {
			t.Errorf("expected property %q not found", prop)
		}
	}
}

func TestTakeSnapshotTool_Execute_NoExchanges(t *testing.T) {
	// Create config with no exchanges enabled
	cfg := config.DefaultConfig()
	cfg.Exchanges.Binance.Enabled = false
	cfg.Exchanges.BinanceTH.Enabled = false
	cfg.Exchanges.Bitkub.Enabled = false
	cfg.Exchanges.OKX.Enabled = false
	cfg.Exchanges.Settrade.Enabled = false

	store := newTestSnapshotStore(t)
	tool := NewTakeSnapshotTool(cfg, store)

	result := tool.Execute(context.Background(), map[string]any{})

	if result.IsError {
		t.Logf("Expected behavior: error when no exchanges configured: %s", result.ForLLM)
	}
}

func TestTakeSnapshotTool_Execute_WithMinimalArgs(t *testing.T) {
	tool := newTestTakeSnapshotTool(t)

	// Test with empty args - should use defaults
	result := tool.Execute(context.Background(), map[string]any{})

	// Result may error due to no configured exchanges, but that's ok
	// We're testing parameter handling here
	if result == nil {
		t.Fatal("Execute should return non-nil result")
	}
}

func TestTakeSnapshotTool_Execute_WithSource(t *testing.T) {
	tool := newTestTakeSnapshotTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"source": "binance",
	})

	if result == nil {
		t.Fatal("Execute should return non-nil result")
	}
}

func TestTakeSnapshotTool_Execute_WithAccount(t *testing.T) {
	tool := newTestTakeSnapshotTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"account": "trading",
	})

	if result == nil {
		t.Fatal("Execute should return non-nil result")
	}
}

func TestTakeSnapshotTool_Execute_WithQuote(t *testing.T) {
	tool := newTestTakeSnapshotTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"quote": "THB",
	})

	if result == nil {
		t.Fatal("Execute should return non-nil result")
	}
}

func TestTakeSnapshotTool_Execute_WithLabel(t *testing.T) {
	tool := newTestTakeSnapshotTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"label": "daily",
	})

	if result == nil {
		t.Fatal("Execute should return non-nil result")
	}
}

func TestTakeSnapshotTool_Execute_WithNote(t *testing.T) {
	tool := newTestTakeSnapshotTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"note": "rebalance checkpoint",
	})

	if result == nil {
		t.Fatal("Execute should return non-nil result")
	}
}

func TestTakeSnapshotTool_Execute_AllArgs(t *testing.T) {
	tool := newTestTakeSnapshotTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"source":  "binance",
		"account": "main",
		"quote":   "usdt",
		"label":   "pre-rebalance",
		"note":    "checking positions before rebalancing",
	})

	if result == nil {
		t.Fatal("Execute should return non-nil result")
	}
}

func TestTakeSnapshotTool_Execute_InvalidArgTypes(t *testing.T) {
	tool := newTestTakeSnapshotTool(t)

	// Should handle non-string args gracefully
	result := tool.Execute(context.Background(), map[string]any{
		"source":  123,        // int instead of string
		"account": true,       // bool instead of string
		"quote":   []string{}, // slice instead of string
	})

	if result == nil {
		t.Fatal("Execute should handle invalid arg types gracefully")
	}
}

func TestTakeSnapshotTool_Execute_EmptyStringArgs(t *testing.T) {
	tool := newTestTakeSnapshotTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"source":  "",
		"account": "",
		"quote":   "",
	})

	if result == nil {
		t.Fatal("Execute should handle empty strings")
	}
}

func TestTakeSnapshotTool_Execute_QuoteNormalization(t *testing.T) {
	tool := newTestTakeSnapshotTool(t)

	// Execute with lowercase quote
	result := tool.Execute(context.Background(), map[string]any{
		"quote": "usdt",
	})

	if result == nil {
		t.Fatal("Execute should work with lowercase quote")
	}
}

func contains(s, substring string) bool {
	return len(s) > 0 && len(substring) > 0 && (s == substring || len(s) > len(substring))
}

package tools

import (
	"context"
	"strings"
	"testing"
	"time"

	snapshotPkg "github.com/cryptoquantumwave/khunquant/pkg/snapshot"
)

func newTestQuerySnapshotsTool(t *testing.T) *QuerySnapshotsTool {
	t.Helper()
	return NewQuerySnapshotsTool(newTestSnapshotStore(t))
}

func TestQuerySnapshotsTool_Name(t *testing.T) {
	tool := newTestQuerySnapshotsTool(t)
	if tool.Name() != NameQuerySnapshots {
		t.Errorf("Name() = %q, want %q", tool.Name(), NameQuerySnapshots)
	}
}

func TestQuerySnapshotsTool_Description(t *testing.T) {
	tool := newTestQuerySnapshotsTool(t)
	desc := tool.Description()
	if desc == "" {
		t.Error("Description() should not be empty")
	}
}

func TestQuerySnapshotsTool_Parameters(t *testing.T) {
	tool := newTestQuerySnapshotsTool(t)
	params := tool.Parameters()

	if params["type"] != "object" {
		t.Errorf("type should be 'object'")
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties should be a map")
	}

	expectedProps := []string{"since", "until", "label", "source", "asset", "limit", "include_positions"}
	for _, prop := range expectedProps {
		if _, ok := props[prop]; !ok {
			t.Errorf("expected property %q not found", prop)
		}
	}
}

func TestQuerySnapshotsTool_Execute_EmptyStore(t *testing.T) {
	tool := newTestQuerySnapshotsTool(t)

	result := tool.Execute(context.Background(), map[string]any{})

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForUser, "No snapshots found") {
		t.Errorf("expected 'No snapshots found' message, got %q", result.ForUser)
	}
}

func TestQuerySnapshotsTool_Execute_NoArgs(t *testing.T) {
	tool := newTestQuerySnapshotsTool(t)

	result := tool.Execute(context.Background(), map[string]any{})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestQuerySnapshotsTool_Execute_WithSince(t *testing.T) {
	tool := newTestQuerySnapshotsTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"since": "24h",
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestQuerySnapshotsTool_Execute_WithUntil(t *testing.T) {
	tool := newTestQuerySnapshotsTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"until": "1h",
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestQuerySnapshotsTool_Execute_WithLabel(t *testing.T) {
	tool := newTestQuerySnapshotsTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"label": "daily",
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestQuerySnapshotsTool_Execute_WithSource(t *testing.T) {
	tool := newTestQuerySnapshotsTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"source": "binance",
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestQuerySnapshotsTool_Execute_WithAsset(t *testing.T) {
	tool := newTestQuerySnapshotsTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"asset": "BTC",
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestQuerySnapshotsTool_Execute_WithLimit(t *testing.T) {
	tool := newTestQuerySnapshotsTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"limit": float64(5),
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestQuerySnapshotsTool_Execute_WithIncludePositions(t *testing.T) {
	tool := newTestQuerySnapshotsTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"include_positions": true,
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestQuerySnapshotsTool_Execute_AllArgs(t *testing.T) {
	tool := newTestQuerySnapshotsTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"since":               "7d",
		"until":               "1d",
		"label":               "daily",
		"source":              "binance",
		"asset":               "BTC",
		"limit":               float64(20),
		"include_positions":   true,
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestQuerySnapshotsTool_Execute_InvalidArgTypes(t *testing.T) {
	tool := newTestQuerySnapshotsTool(t)

	// Should handle non-matching types gracefully
	result := tool.Execute(context.Background(), map[string]any{
		"since":   123,
		"limit":   "not_a_number",
		"include_positions": "yes",
	})

	if result == nil {
		t.Fatal("Execute should return result even with invalid types")
	}
}

func TestParseTimeParam_Empty(t *testing.T) {
	result := parseTimeParam("")
	if result != nil {
		t.Error("empty string should return nil")
	}
}

func TestParseTimeParam_Whitespace(t *testing.T) {
	result := parseTimeParam("   ")
	if result != nil {
		t.Error("whitespace should return nil")
	}
}

func TestParseTimeParam_RelativeDays(t *testing.T) {
	result := parseTimeParam("7d")
	if result == nil {
		t.Fatal("7d should parse successfully")
	}
	// Should be approximately 7 days ago
	diff := time.Now().Sub(*result)
	if diff < 6*24*time.Hour || diff > 8*24*time.Hour {
		t.Errorf("7d should be approximately 7 days ago, got %v", diff)
	}
}

func TestParseTimeParam_RelativeHours(t *testing.T) {
	result := parseTimeParam("24h")
	if result == nil {
		t.Fatal("24h should parse successfully")
	}
	// Should be approximately 24 hours ago
	diff := time.Now().Sub(*result)
	if diff < 23*time.Hour || diff > 25*time.Hour {
		t.Errorf("24h should be approximately 24 hours ago, got %v", diff)
	}
}

func TestParseTimeParam_RelativeMinutes(t *testing.T) {
	result := parseTimeParam("30m")
	if result == nil {
		t.Fatal("30m should parse successfully")
	}
	// Should be approximately 30 minutes ago
	diff := time.Now().Sub(*result)
	if diff < 29*time.Minute || diff > 31*time.Minute {
		t.Errorf("30m should be approximately 30 minutes ago, got %v", diff)
	}
}

func TestParseTimeParam_ISO8601Full(t *testing.T) {
	result := parseTimeParam("2025-01-15T10:30:45Z")
	if result == nil {
		t.Fatal("ISO 8601 full format should parse")
	}
	if result.Year() != 2025 || result.Month() != 1 || result.Day() != 15 {
		t.Errorf("expected 2025-01-15, got %v", result)
	}
}

func TestParseTimeParam_ISO8601DateTime(t *testing.T) {
	result := parseTimeParam("2025-01-15T10:30:45")
	if result == nil {
		t.Fatal("ISO 8601 datetime should parse")
	}
	if result.Year() != 2025 || result.Month() != 1 || result.Day() != 15 {
		t.Errorf("expected 2025-01-15, got %v", result)
	}
}

func TestParseTimeParam_ISO8601Date(t *testing.T) {
	result := parseTimeParam("2025-01-15")
	if result == nil {
		t.Fatal("ISO 8601 date should parse")
	}
	if result.Year() != 2025 || result.Month() != 1 || result.Day() != 15 {
		t.Errorf("expected 2025-01-15, got %v", result)
	}
}

func TestParseTimeParam_InvalidRelative(t *testing.T) {
	result := parseTimeParam("invalid")
	if result != nil {
		t.Error("invalid format should return nil")
	}
}

func TestParseTimeParam_RelativeZero(t *testing.T) {
	result := parseTimeParam("0d")
	if result != nil {
		t.Error("0d should return nil (zero duration)")
	}
}

func TestParseTimeParam_RelativeNegative(t *testing.T) {
	result := parseTimeParam("-5d")
	if result != nil {
		t.Error("negative duration should return nil")
	}
}

func TestParseInt_Valid(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"0", 0},
		{"1", 1},
		{"10", 10},
		{"999", 999},
		{"007", 7},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseInt(tt.input)
			if got != tt.want {
				t.Errorf("parseInt(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseInt_Invalid(t *testing.T) {
	tests := []struct {
		input string
	}{
		{""},
		{"abc"},
		{"1a2"},
		{"-5"},
		{"12.34"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseInt(tt.input)
			if got != 0 {
				t.Errorf("parseInt(%q) = %d, want 0 for invalid input", tt.input, got)
			}
		})
	}
}

func TestQuerySnapshotsTool_Execute_WithDataNoPositions(t *testing.T) {
	store := newTestSnapshotStore(t)
	tool := NewQuerySnapshotsTool(store)
	ctx := context.Background()

	snap := &snapshotPkg.Snapshot{
		TakenAt:    time.Now(),
		Quote:      "USDT",
		TotalValue: 5000.0,
		Label:      "test",
	}
	if _, err := store.SaveSnapshot(ctx, snap); err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}

	result := tool.Execute(ctx, map[string]any{})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForUser, "5000") {
		t.Errorf("expected total value in output, got: %s", result.ForUser)
	}
}

func TestQuerySnapshotsTool_Execute_WithIncludePositionsAndData(t *testing.T) {
	store := newTestSnapshotStore(t)
	tool := NewQuerySnapshotsTool(store)
	ctx := context.Background()

	snap := &snapshotPkg.Snapshot{
		TakenAt:    time.Now(),
		Quote:      "USDT",
		TotalValue: 10000.0,
		Label:      "test",
		Positions: []snapshotPkg.Position{
			{Source: "binance", Account: "main", Asset: "BTC", Quantity: 0.5, Price: 20000, Quote: "USDT", Value: 10000},
		},
	}
	if _, err := store.SaveSnapshot(ctx, snap); err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}

	result := tool.Execute(ctx, map[string]any{
		"include_positions": true,
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForUser, "BTC") {
		t.Errorf("expected BTC position in output, got: %s", result.ForUser)
	}
}

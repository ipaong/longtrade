package tools

import (
	"context"
	"testing"
	"time"

	snapshotPkg "github.com/cryptoquantumwave/khunquant/pkg/snapshot"
)

func newTestSnapshotSummaryTool(t *testing.T) *SnapshotSummaryTool {
	t.Helper()
	return NewSnapshotSummaryTool(newTestSnapshotStore(t))
}

func TestSnapshotSummaryTool_Name(t *testing.T) {
	tool := newTestSnapshotSummaryTool(t)
	if tool.Name() != NameSnapshotSummary {
		t.Errorf("Name() = %q, want %q", tool.Name(), NameSnapshotSummary)
	}
}

func TestSnapshotSummaryTool_Description(t *testing.T) {
	tool := newTestSnapshotSummaryTool(t)
	desc := tool.Description()
	if desc == "" {
		t.Error("Description() should not be empty")
	}
}

func TestSnapshotSummaryTool_Parameters(t *testing.T) {
	tool := newTestSnapshotSummaryTool(t)
	params := tool.Parameters()

	if params["type"] != "object" {
		t.Errorf("type should be 'object'")
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties should be a map")
	}

	expectedProps := []string{"since", "until", "label", "source", "group_by"}
	for _, prop := range expectedProps {
		if _, ok := props[prop]; !ok {
			t.Errorf("expected property %q not found", prop)
		}
	}
}

func TestSnapshotSummaryTool_Execute_NoArgs(t *testing.T) {
	tool := newTestSnapshotSummaryTool(t)

	result := tool.Execute(context.Background(), map[string]any{})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestSnapshotSummaryTool_Execute_WithSince(t *testing.T) {
	tool := newTestSnapshotSummaryTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"since": "30d",
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestSnapshotSummaryTool_Execute_WithUntil(t *testing.T) {
	tool := newTestSnapshotSummaryTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"until": "2025-01-01",
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestSnapshotSummaryTool_Execute_WithLabel(t *testing.T) {
	tool := newTestSnapshotSummaryTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"label": "daily",
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestSnapshotSummaryTool_Execute_WithSource(t *testing.T) {
	tool := newTestSnapshotSummaryTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"source": "binance",
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestSnapshotSummaryTool_Execute_GroupByDay(t *testing.T) {
	tool := newTestSnapshotSummaryTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"group_by": "day",
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestSnapshotSummaryTool_Execute_GroupByWeek(t *testing.T) {
	tool := newTestSnapshotSummaryTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"group_by": "week",
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestSnapshotSummaryTool_Execute_GroupByMonth(t *testing.T) {
	tool := newTestSnapshotSummaryTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"group_by": "month",
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestSnapshotSummaryTool_Execute_GroupBySource(t *testing.T) {
	tool := newTestSnapshotSummaryTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"group_by": "source",
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestSnapshotSummaryTool_Execute_AllArgs(t *testing.T) {
	tool := newTestSnapshotSummaryTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"since":    "30d",
		"until":    "7d",
		"label":    "daily",
		"source":   "binance",
		"group_by": "week",
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestSnapshotSummaryTool_Execute_InvalidArgTypes(t *testing.T) {
	tool := newTestSnapshotSummaryTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"since":    123,
		"until":    true,
		"group_by": 456,
	})

	if result == nil {
		t.Fatal("Execute should return result even with invalid types")
	}
}

func TestSnapshotSummaryTool_Execute_WithData(t *testing.T) {
	store := newTestSnapshotStore(t)
	tool := NewSnapshotSummaryTool(store)
	ctx := context.Background()

	// Seed two snapshots so Count > 0 and the formatting block runs.
	for i, v := range []float64{1000.0, 1200.0} {
		snap := &snapshotPkg.Snapshot{
			TakenAt:    time.Now().Add(time.Duration(i) * time.Hour),
			Quote:      "USDT",
			TotalValue: v,
			Label:      "test",
		}
		if _, err := store.SaveSnapshot(ctx, snap); err != nil {
			t.Fatalf("SaveSnapshot: %v", err)
		}
	}

	result := tool.Execute(ctx, map[string]any{})
	if result == nil {
		t.Fatal("Execute should return result")
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if !contains(result.ForUser, "Portfolio Snapshot Summary") {
		t.Errorf("expected summary header in result, got: %s", result.ForUser)
	}
}

func TestSnapshotSummaryTool_Execute_WithDataAndGroupBy(t *testing.T) {
	store := newTestSnapshotStore(t)
	tool := NewSnapshotSummaryTool(store)
	ctx := context.Background()

	// Seed snapshots so the Groups block is exercised.
	for i, v := range []float64{900.0, 1100.0, 1300.0} {
		snap := &snapshotPkg.Snapshot{
			TakenAt:    time.Now().Add(time.Duration(i*24) * time.Hour),
			Quote:      "USDT",
			TotalValue: v,
			Label:      "daily",
		}
		if _, err := store.SaveSnapshot(ctx, snap); err != nil {
			t.Fatalf("SaveSnapshot: %v", err)
		}
	}

	result := tool.Execute(ctx, map[string]any{"group_by": "day"})
	if result == nil {
		t.Fatal("Execute should return result")
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
}

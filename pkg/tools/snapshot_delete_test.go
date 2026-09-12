package tools

import (
	"context"
	"testing"
)

func newTestDeleteSnapshotsTool(t *testing.T) *DeleteSnapshotsTool {
	t.Helper()
	return NewDeleteSnapshotsTool(newTestSnapshotStore(t))
}

func TestDeleteSnapshotsTool_Name(t *testing.T) {
	tool := newTestDeleteSnapshotsTool(t)
	if tool.Name() != NameDeleteSnapshots {
		t.Errorf("Name() = %q, want %q", tool.Name(), NameDeleteSnapshots)
	}
}

func TestDeleteSnapshotsTool_Description(t *testing.T) {
	tool := newTestDeleteSnapshotsTool(t)
	desc := tool.Description()
	if desc == "" {
		t.Error("Description() should not be empty")
	}
}

func TestDeleteSnapshotsTool_Parameters(t *testing.T) {
	tool := newTestDeleteSnapshotsTool(t)
	params := tool.Parameters()

	if params["type"] != "object" {
		t.Errorf("type should be 'object'")
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties should be a map")
	}

	expectedProps := []string{"id", "before", "label", "keep_last", "confirm"}
	for _, prop := range expectedProps {
		if _, ok := props[prop]; !ok {
			t.Errorf("expected property %q not found", prop)
		}
	}
}

func TestDeleteSnapshotsTool_Execute_NoArgs(t *testing.T) {
	tool := newTestDeleteSnapshotsTool(t)

	result := tool.Execute(context.Background(), map[string]any{})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestDeleteSnapshotsTool_Execute_ByID(t *testing.T) {
	tool := newTestDeleteSnapshotsTool(t)

	// Try to delete non-existent ID (should not error, just not find anything)
	result := tool.Execute(context.Background(), map[string]any{
		"id": float64(999),
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestDeleteSnapshotsTool_Execute_ByLabel(t *testing.T) {
	tool := newTestDeleteSnapshotsTool(t)

	// Without confirm should return error or warning
	result := tool.Execute(context.Background(), map[string]any{
		"label": "temp",
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestDeleteSnapshotsTool_Execute_ByBefore(t *testing.T) {
	tool := newTestDeleteSnapshotsTool(t)

	// Without confirm should return error or warning
	result := tool.Execute(context.Background(), map[string]any{
		"before": "2020-01-01",
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestDeleteSnapshotsTool_Execute_WithConfirm(t *testing.T) {
	tool := newTestDeleteSnapshotsTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"label":   "temp",
		"confirm": true,
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestDeleteSnapshotsTool_Execute_WithKeepLast(t *testing.T) {
	tool := newTestDeleteSnapshotsTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"before":    "2020-01-01",
		"keep_last": float64(10),
		"confirm":   true,
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestDeleteSnapshotsTool_Execute_AllArgs(t *testing.T) {
	tool := newTestDeleteSnapshotsTool(t)

	result := tool.Execute(context.Background(), map[string]any{
		"label":     "old_backup",
		"keep_last": float64(5),
		"confirm":   true,
	})

	if result == nil {
		t.Fatal("Execute should return result")
	}
}

func TestDeleteSnapshotsTool_Execute_InvalidArgTypes(t *testing.T) {
	tool := newTestDeleteSnapshotsTool(t)

	// Should handle non-matching types gracefully
	result := tool.Execute(context.Background(), map[string]any{
		"id":        "not_a_number",
		"before":    123,
		"keep_last": "lots",
		"confirm":   "yes",
	})

	if result == nil {
		t.Fatal("Execute should return result even with invalid types")
	}
}

package tools

import (
	"context"
	"testing"
	"time"

	"github.com/cryptoquantumwave/khunquant/pkg/config"
	"github.com/cryptoquantumwave/khunquant/pkg/dca"
)

func TestExecuteDCAOrder_MissingPlanID(t *testing.T) {
	tool := NewExecuteDCAOrderTool(config.DefaultConfig(), newTestDCAStore(t))
	result := tool.Execute(testCtx(), map[string]any{})
	if !result.IsError {
		t.Fatal("expected error when plan_id is missing")
	}
}

func TestExecuteDCAOrder_PlanNotFound(t *testing.T) {
	tool := NewExecuteDCAOrderTool(config.DefaultConfig(), newTestDCAStore(t))
	result := tool.Execute(testCtx(), map[string]any{
		"plan_id": float64(99999),
	})
	if !result.IsError {
		t.Fatal("expected error for non-existent plan")
	}
}

func TestExecuteDCAOrder_DisabledPlan(t *testing.T) {
	store := newTestDCAStore(t)
	planID := seedPlan(t, store, "DisabledPlan", false)
	tool := NewExecuteDCAOrderTool(config.DefaultConfig(), store)

	result := tool.Execute(testCtx(), map[string]any{
		"plan_id": float64(planID),
	})
	if !result.IsError {
		t.Fatal("expected error for disabled plan")
	}
}

func TestExecuteDCAOrder_ExpiredPlan(t *testing.T) {
	store := newTestDCAStore(t)
	planID := seedPlan(t, store, "ExpiredPlan", true)

	plan, _ := store.GetPlan(context.Background(), planID)
	past := time.Now().UTC().Add(-24 * time.Hour)
	plan.EndDate = &past
	if err := store.UpdatePlan(context.Background(), plan); err != nil {
		t.Fatalf("UpdatePlan: %v", err)
	}

	tool := NewExecuteDCAOrderTool(config.DefaultConfig(), store)
	result := tool.Execute(testCtx(), map[string]any{
		"plan_id": float64(planID),
	})
	if !result.IsError {
		t.Fatal("expected error for expired plan")
	}
}

func TestExecuteDCAOrder_GuardrailExceeded(t *testing.T) {
	store := newTestDCAStore(t)
	planID := seedPlan(t, store, "GuardrailPlan", true)

	plan, _ := store.GetPlan(context.Background(), planID)
	plan.MaxExecPerPeriod = 1
	plan.ExecPeriod = "day"
	if err := store.UpdatePlan(context.Background(), plan); err != nil {
		t.Fatalf("UpdatePlan: %v", err)
	}

	// Insert one execution in the current day to trip the guardrail.
	// Use local time so CountExecutionsInPeriod's local-midnight "since" string-compares correctly.
	now := time.Now()
	exec := &dca.Execution{
		PlanID:      planID,
		ExecutedAt:  now,
		Symbol:      "BTC/THB",
		Provider:    "bitkub",
		AmountQuote: 100,
		Status:      "completed",
		CreatedAt:   now,
	}
	if _, err := store.SaveExecution(context.Background(), exec); err != nil {
		t.Fatalf("SaveExecution: %v", err)
	}

	// The execute tool hits CheckPermission before guardrail, so we use a real
	// config that will fail permission (no API keys). The guardrail would kick
	// in after the permission gate — verify via CountExecutionsInPeriod directly.
	count, err := store.CountExecutionsInPeriod(context.Background(), planID, "day")
	if err != nil {
		t.Fatalf("CountExecutionsInPeriod: %v", err)
	}
	if count < plan.MaxExecPerPeriod {
		t.Errorf("expected count >= %d, got %d", plan.MaxExecPerPeriod, count)
	}
}

func TestSplit(t *testing.T) {
	tests := []struct {
		symbol string
		want   string
	}{
		{"BTC/USDT", "BTC"},
		{"ETH/THB", "ETH"},
		{"noSlash", "noSlash"},
	}
	for _, tc := range tests {
		got := split(tc.symbol)
		if got != tc.want {
			t.Errorf("split(%q) = %q, want %q", tc.symbol, got, tc.want)
		}
	}
}

func TestExecuteDCAOrderTool_Name(t *testing.T) {
	tool := NewExecuteDCAOrderTool(config.DefaultConfig(), newTestDCAStore(t))
	if tool.Name() != NameExecuteDCAOrder {
		t.Errorf("Name() = %q, want %q", tool.Name(), NameExecuteDCAOrder)
	}
}

func TestExecuteDCAOrderTool_Description(t *testing.T) {
	tool := NewExecuteDCAOrderTool(config.DefaultConfig(), newTestDCAStore(t))
	desc := tool.Description()
	if desc == "" {
		t.Fatal("Description() should not be empty")
	}
	if !containsStr(desc, "order") && !containsStr(desc, "Order") {
		t.Errorf("Description should mention orders, got: %s", desc)
	}
}

func TestExecuteDCAOrderTool_Parameters(t *testing.T) {
	tool := NewExecuteDCAOrderTool(config.DefaultConfig(), newTestDCAStore(t))
	params := tool.Parameters()

	if params == nil {
		t.Fatal("Parameters() should not return nil")
	}
	if params["type"] != "object" {
		t.Errorf("type should be 'object', got %q", params["type"])
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties to be a map")
	}

	if _, ok := props["plan_id"]; !ok {
		t.Error("expected property 'plan_id' in Parameters")
	}

	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("required should be a slice")
	}
	if len(required) == 0 || required[0] != "plan_id" {
		t.Errorf("plan_id should be required, got %v", required)
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

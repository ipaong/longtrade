package tools

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cryptoquantumwave/khunquant/pkg/config"
	"github.com/cryptoquantumwave/khunquant/pkg/dca"
)

func TestGetDCASummary_MissingPlanID(t *testing.T) {
	tool := NewGetDCASummaryTool(config.DefaultConfig(), newTestDCAStore(t))
	result := tool.Execute(testCtx(), map[string]any{})
	if !result.IsError {
		t.Fatal("expected error when plan_id is missing")
	}
}

func TestGetDCASummary_PlanNotFound(t *testing.T) {
	tool := NewGetDCASummaryTool(config.DefaultConfig(), newTestDCAStore(t))
	result := tool.Execute(testCtx(), map[string]any{
		"plan_id": float64(99999),
	})
	if !result.IsError {
		t.Fatal("expected error for non-existent plan")
	}
}

func TestGetDCASummary_ZeroInvestment(t *testing.T) {
	store := newTestDCAStore(t)
	planID := seedPlan(t, store, "SummaryEmpty", true)
	tool := NewGetDCASummaryTool(config.DefaultConfig(), store)

	result := tool.Execute(testCtx(), map[string]any{
		"plan_id": float64(planID),
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForUser, "live price unavailable") {
		t.Errorf("expected 'live price unavailable', got: %s", result.ForUser)
	}
	// totals should be zero
	if !strings.Contains(result.ForUser, "0.0000") {
		t.Errorf("expected zero totals, got: %s", result.ForUser)
	}
}

func TestGetDCASummary_WithStats(t *testing.T) {
	store := newTestDCAStore(t)
	planID := seedPlan(t, store, "SummaryStats", true)

	// Two executions: 100 quote for 0.1 base, then 200 quote for 0.2 base.
	if err := store.UpdatePlanStats(context.Background(), planID, 100, 0.1); err != nil {
		t.Fatalf("UpdatePlanStats 1: %v", err)
	}
	if err := store.UpdatePlanStats(context.Background(), planID, 200, 0.2); err != nil {
		t.Fatalf("UpdatePlanStats 2: %v", err)
	}

	tool := NewGetDCASummaryTool(config.DefaultConfig(), store)
	result := tool.Execute(testCtx(), map[string]any{
		"plan_id": float64(planID),
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}

	// Total invested = 300, avg cost = 300/0.3 = 1000
	if !strings.Contains(result.ForUser, "300.0000") {
		t.Errorf("expected total invested 300, got: %s", result.ForUser)
	}
	if !strings.Contains(result.ForUser, "1000.0000") {
		t.Errorf("expected avg cost 1000, got: %s", result.ForUser)
	}
}

func TestGetDCASummaryTool_Name(t *testing.T) {
	tool := NewGetDCASummaryTool(config.DefaultConfig(), newTestDCAStore(t))
	if tool.Name() != NameGetDCASummary {
		t.Errorf("Name() = %q, want %q", tool.Name(), NameGetDCASummary)
	}
}

func TestGetDCASummaryTool_Description(t *testing.T) {
	tool := NewGetDCASummaryTool(config.DefaultConfig(), newTestDCAStore(t))
	desc := tool.Description()
	if desc == "" {
		t.Fatal("Description() should not be empty")
	}
	if !strings.Contains(desc, "PnL") && !strings.Contains(desc, "summary") {
		t.Errorf("Description should mention PnL or summary, got: %s", desc)
	}
}

func TestGetDCASummaryTool_Parameters(t *testing.T) {
	tool := NewGetDCASummaryTool(config.DefaultConfig(), newTestDCAStore(t))
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

func TestGetDCASummary_WithLastExecution(t *testing.T) {
	store := newTestDCAStore(t)
	planID := seedPlan(t, store, "SummaryLast", true)

	if _, err := store.SaveExecution(context.Background(), &dca.Execution{
		PlanID:         planID,
		ExecutedAt:     time.Now().Add(-1 * time.Hour),
		Symbol:         "BTCUSDT",
		Provider:       "test",
		Account:        "main",
		Status:         "completed",
		AmountQuote:    100,
		FilledPrice:    50000,
		FilledQuantity: 0.002,
	}); err != nil {
		t.Fatalf("SaveExecution: %v", err)
	}

	tool := NewGetDCASummaryTool(config.DefaultConfig(), store)
	result := tool.Execute(context.Background(), map[string]any{
		"plan_id": float64(planID),
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForUser, "Last execution") {
		t.Errorf("expected 'Last execution' in output, got: %s", result.ForUser)
	}
}

func TestEnabledLabel_Active(t *testing.T) {
	got := enabledLabel(true)
	if got == "" {
		t.Error("enabledLabel(true) should return non-empty string")
	}
}

func TestEnabledLabel_Paused(t *testing.T) {
	got := enabledLabel(false)
	if got == "" {
		t.Error("enabledLabel(false) should return non-empty string")
	}
}

func TestEnabledLabel_Distinct(t *testing.T) {
	active := enabledLabel(true)
	paused := enabledLabel(false)
	if active == paused {
		t.Errorf("enabledLabel(true) and enabledLabel(false) should return different strings, both got %q", active)
	}
}

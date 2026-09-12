package tools

import (
	"context"
	"testing"
)

func TestDeleteDCAPlan_MissingPlanID(t *testing.T) {
	store := newTestDCAStore(t)
	cronSvc := newTestCronService(t)
	tool := NewDeleteDCAPlanTool(store, cronSvc)

	result := tool.Execute(testCtx(), map[string]any{})
	if !result.IsError {
		t.Fatal("expected error when plan_id is missing")
	}
}

func TestDeleteDCAPlan_PlanNotFound(t *testing.T) {
	store := newTestDCAStore(t)
	cronSvc := newTestCronService(t)
	tool := NewDeleteDCAPlanTool(store, cronSvc)

	result := tool.Execute(testCtx(), map[string]any{
		"plan_id": float64(99999),
	})
	if !result.IsError {
		t.Fatal("expected error for non-existent plan")
	}
}

func TestDeleteDCAPlan_Success(t *testing.T) {
	store := newTestDCAStore(t)
	cronSvc := newTestCronService(t)
	planID, jobID := seedPlanWithJob(t, store, cronSvc)
	tool := NewDeleteDCAPlanTool(store, cronSvc)

	result := tool.Execute(testCtx(), map[string]any{
		"plan_id": float64(planID),
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}

	// Plan must be gone from store.
	if _, err := store.GetPlan(context.Background(), planID); err == nil {
		t.Error("expected GetPlan to error after deletion")
	}

	// Cron job must be removed.
	for _, j := range cronSvc.ListJobs(false) {
		if j.ID == jobID {
			t.Errorf("cron job %q still exists after plan deletion", jobID)
		}
	}
}

func TestDeleteDCAPlanTool_Name(t *testing.T) {
	store := newTestDCAStore(t)
	cronSvc := newTestCronService(t)
	tool := NewDeleteDCAPlanTool(store, cronSvc)
	if tool.Name() != NameDeleteDCAPlan {
		t.Errorf("Name() = %q, want %q", tool.Name(), NameDeleteDCAPlan)
	}
}

func TestDeleteDCAPlanTool_Description(t *testing.T) {
	store := newTestDCAStore(t)
	cronSvc := newTestCronService(t)
	tool := NewDeleteDCAPlanTool(store, cronSvc)
	desc := tool.Description()
	if desc == "" {
		t.Fatal("Description() should not be empty")
	}
	if !containsSubstr(desc, "delete") && !containsSubstr(desc, "Delete") {
		t.Errorf("Description should mention deletion, got: %s", desc)
	}
}

func TestDeleteDCAPlanTool_Parameters(t *testing.T) {
	store := newTestDCAStore(t)
	cronSvc := newTestCronService(t)
	tool := NewDeleteDCAPlanTool(store, cronSvc)
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

func containsSubstr(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

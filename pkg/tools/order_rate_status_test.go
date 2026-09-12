package tools

import (
	"context"
	"testing"
)

func TestGetOrderRateStatus_NoArgs(t *testing.T) {
	tool := NewGetOrderRateStatusTool()

	result := tool.Execute(context.Background(), map[string]any{})
	if result == nil {
		t.Fatal("expected result")
	}
	// This tool doesn't fail — it returns status info
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
}

func TestGetOrderRateStatus_IgnoresArgs(t *testing.T) {
	tool := NewGetOrderRateStatusTool()

	result := tool.Execute(context.Background(), map[string]any{
		"extra": "arg",
	})
	if result == nil {
		t.Fatal("expected result")
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
}

func TestGetOrderRateStatus_ParametersSchema(t *testing.T) {
	tool := NewGetOrderRateStatusTool()
	params := tool.Parameters()

	if params == nil {
		t.Fatal("Parameters() should not return nil")
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties in Parameters")
	}

	// Should have empty properties (no required args)
	if len(props) != 0 {
		t.Errorf("expected no properties, got %d", len(props))
	}

	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("expected required in Parameters")
	}
	// Should have no required fields
	if len(required) != 0 {
		t.Errorf("expected no required fields, got %d", len(required))
	}
}

func TestGetOrderRateStatus_Name(t *testing.T) {
	tool := NewGetOrderRateStatusTool()
	name := tool.Name()
	if name != NameGetOrderRateStatus {
		t.Errorf("Name() = %q, want %q", name, NameGetOrderRateStatus)
	}
}

func TestGetOrderRateStatus_Description(t *testing.T) {
	tool := NewGetOrderRateStatusTool()
	desc := tool.Description()
	if desc == "" {
		t.Fatal("Description() should not be empty")
	}
	if len(desc) < 10 {
		t.Fatal("Description() too short")
	}
}

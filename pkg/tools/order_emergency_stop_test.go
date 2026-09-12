package tools

import (
	"context"
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/config"
)

func TestEmergencyStop_NoConfirmation(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewEmergencyStopTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"confirm": false,
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	// Should return dry-run message
	if result.ForUser == "" {
		t.Fatal("expected dry-run message")
	}
}

func TestEmergencyStop_WithConfirmation(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewEmergencyStopTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"confirm": true,
	})
	// Should succeed with summary (may or may not have errors depending on config)
	if result == nil {
		t.Fatal("expected result")
	}
}

func TestEmergencyStop_MissingConfirm(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewEmergencyStopTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{})
	// confirm defaults to false
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
}

func TestEmergencyStop_ConfirmFalse(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewEmergencyStopTool(cfg)

	result := tool.Execute(context.Background(), map[string]any{
		"confirm": false,
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	// Should return dry-run message
	if result.ForUser == "" {
		t.Fatal("expected dry-run message")
	}
}

func TestEmergencyStop_ParametersSchema(t *testing.T) {
	tool := NewEmergencyStopTool(config.DefaultConfig())
	params := tool.Parameters()

	if params == nil {
		t.Fatal("Parameters() should not return nil")
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties in Parameters")
	}

	if _, ok := props["confirm"]; !ok {
		t.Fatal("expected 'confirm' property in Parameters")
	}

	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("expected required in Parameters")
	}
	// confirm should be required
	if len(required) != 1 {
		t.Errorf("expected 1 required field, got %d", len(required))
	}
}

func TestEmergencyStop_Name(t *testing.T) {
	tool := NewEmergencyStopTool(config.DefaultConfig())
	name := tool.Name()
	if name != NameEmergencyStop {
		t.Errorf("Name() = %q, want %q", name, NameEmergencyStop)
	}
}

func TestEmergencyStop_Description(t *testing.T) {
	tool := NewEmergencyStopTool(config.DefaultConfig())
	desc := tool.Description()
	if desc == "" {
		t.Fatal("Description() should not be empty")
	}
	if desc == "" {
		t.Fatal("Description should not be empty")
	}
}

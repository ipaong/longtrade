package providers

import (
	"encoding/json"
	"testing"
)

func TestNormalizeToolCall_EmptyInput(t *testing.T) {
	tc := ToolCall{}
	normalized := NormalizeToolCall(tc)

	if normalized.Arguments == nil {
		t.Error("expected Arguments to be initialized to empty map, got nil")
	}
	if len(normalized.Arguments) != 0 {
		t.Errorf("expected empty Arguments map, got %d items", len(normalized.Arguments))
	}
}

func TestNormalizeToolCall_NameFromFunction(t *testing.T) {
	tc := ToolCall{
		Function: &FunctionCall{
			Name:      "fetch_weather",
			Arguments: `{"location":"NYC"}`,
		},
	}

	normalized := NormalizeToolCall(tc)

	if normalized.Name != "fetch_weather" {
		t.Errorf("expected name 'fetch_weather', got %q", normalized.Name)
	}
}

func TestNormalizeToolCall_NameAtTopLevel(t *testing.T) {
	tc := ToolCall{
		Name: "get_user_info",
		Function: &FunctionCall{
			Name:      "",
			Arguments: `{"id":"123"}`,
		},
	}

	normalized := NormalizeToolCall(tc)

	if normalized.Name != "get_user_info" {
		t.Errorf("expected name 'get_user_info', got %q", normalized.Name)
	}
	if normalized.Function.Name != "get_user_info" {
		t.Errorf("expected Function.Name to be synced to 'get_user_info', got %q", normalized.Function.Name)
	}
}

func TestNormalizeToolCall_ParseArgumentsFromFunction(t *testing.T) {
	tc := ToolCall{
		Name: "test_func",
		Function: &FunctionCall{
			Name:      "test_func",
			Arguments: `{"param1":"value1","param2":42}`,
		},
	}

	normalized := NormalizeToolCall(tc)

	if len(normalized.Arguments) != 2 {
		t.Fatalf("expected 2 arguments, got %d", len(normalized.Arguments))
	}
	if normalized.Arguments["param1"] != "value1" {
		t.Errorf("expected param1='value1', got %v", normalized.Arguments["param1"])
	}
	if normalized.Arguments["param2"] != float64(42) {
		t.Errorf("expected param2=42, got %v", normalized.Arguments["param2"])
	}
}

func TestNormalizeToolCall_InvalidJSONInFunctionArguments(t *testing.T) {
	tc := ToolCall{
		Name: "test_func",
		Function: &FunctionCall{
			Name:      "test_func",
			Arguments: `{invalid json}`,
		},
	}

	normalized := NormalizeToolCall(tc)

	// Should not crash, Arguments should be empty
	if len(normalized.Arguments) != 0 {
		t.Errorf("expected empty Arguments on parse error, got %d items", len(normalized.Arguments))
	}
}

func TestNormalizeToolCall_ArgumentsAlreadyParsed(t *testing.T) {
	tc := ToolCall{
		Name: "my_tool",
		Arguments: map[string]any{
			"key1": "val1",
			"key2": 123,
		},
		Function: &FunctionCall{
			Name:      "my_tool",
			Arguments: `{"old":"value"}`,
		},
	}

	normalized := NormalizeToolCall(tc)

	if normalized.Arguments["key1"] != "val1" {
		t.Errorf("expected key1='val1', got %v", normalized.Arguments["key1"])
	}
	if normalized.Arguments["key2"] != 123 {
		t.Errorf("expected key2=123, got %v", normalized.Arguments["key2"])
	}
}

func TestNormalizeToolCall_CreatesFunctionIfNil(t *testing.T) {
	tc := ToolCall{
		Name: "create_order",
		Arguments: map[string]any{
			"symbol": "BTC/USD",
			"amount": 0.5,
		},
	}

	normalized := NormalizeToolCall(tc)

	if normalized.Function == nil {
		t.Fatal("expected Function to be created, got nil")
	}
	if normalized.Function.Name != "create_order" {
		t.Errorf("expected Function.Name='create_order', got %q", normalized.Function.Name)
	}

	// Verify Arguments are JSON-encoded in Function
	var parsed map[string]any
	err := json.Unmarshal([]byte(normalized.Function.Arguments), &parsed)
	if err != nil {
		t.Fatalf("expected valid JSON in Function.Arguments, got error: %v", err)
	}
	if parsed["symbol"] != "BTC/USD" {
		t.Errorf("expected symbol='BTC/USD', got %v", parsed["symbol"])
	}
}

func TestNormalizeToolCall_SyncsNameBothWays(t *testing.T) {
	// Case 1: Name at top level, missing from Function
	tc1 := ToolCall{
		Name: "tool_a",
		Function: &FunctionCall{
			Name:      "",
			Arguments: "{}",
		},
	}

	normalized1 := NormalizeToolCall(tc1)
	if normalized1.Function.Name != "tool_a" {
		t.Errorf("case 1: expected Function.Name='tool_a', got %q", normalized1.Function.Name)
	}

	// Case 2: Name at Function level, missing from top
	tc2 := ToolCall{
		Name: "",
		Function: &FunctionCall{
			Name:      "tool_b",
			Arguments: "{}",
		},
	}

	normalized2 := NormalizeToolCall(tc2)
	if normalized2.Name != "tool_b" {
		t.Errorf("case 2: expected Name='tool_b', got %q", normalized2.Name)
	}
}

func TestNormalizeToolCall_EmptyFunctionArgumentsString(t *testing.T) {
	tc := ToolCall{
		Name: "my_func",
		Function: &FunctionCall{
			Name:      "my_func",
			Arguments: "",
		},
		Arguments: map[string]any{
			"x": 10,
		},
	}

	normalized := NormalizeToolCall(tc)

	if normalized.Function.Arguments == "" {
		t.Error("expected Function.Arguments to be populated, got empty string")
	}

	var parsed map[string]any
	err := json.Unmarshal([]byte(normalized.Function.Arguments), &parsed)
	if err != nil {
		t.Fatalf("expected valid JSON, got error: %v", err)
	}
	if parsed["x"] != float64(10) {
		t.Errorf("expected x=10, got %v", parsed["x"])
	}
}

func TestNormalizeToolCall_ComplexNestedArguments(t *testing.T) {
	tc := ToolCall{
		Name: "complex_tool",
		Function: &FunctionCall{
			Name: "complex_tool",
			Arguments: `{"nested":{"deep":{"value":true}},"array":[1,2,3]}`,
		},
	}

	normalized := NormalizeToolCall(tc)

	if len(normalized.Arguments) != 2 {
		t.Fatalf("expected 2 top-level arguments, got %d", len(normalized.Arguments))
	}

	nested, ok := normalized.Arguments["nested"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested to be map, got %T", normalized.Arguments["nested"])
	}

	deep, ok := nested["deep"].(map[string]any)
	if !ok {
		t.Fatalf("expected deep to be map, got %T", nested["deep"])
	}

	if deep["value"] != true {
		t.Errorf("expected value=true, got %v", deep["value"])
	}

	arr, ok := normalized.Arguments["array"].([]any)
	if !ok {
		t.Fatalf("expected array to be []any, got %T", normalized.Arguments["array"])
	}
	if len(arr) != 3 {
		t.Errorf("expected array length 3, got %d", len(arr))
	}
}

func TestBuildCLIToolsPrompt_EmptyTools(t *testing.T) {
	prompt := buildCLIToolsPrompt([]ToolDefinition{})

	if prompt == "" {
		t.Error("expected non-empty prompt for empty tools")
	}
	if !contains(prompt, "Available Tools") {
		t.Error("expected 'Available Tools' in prompt")
	}
}

func TestBuildCLIToolsPrompt_WithTools(t *testing.T) {
	tools := []ToolDefinition{
		{
			Type: "function",
			Function: ToolFunctionDefinition{
				Name:        "get_weather",
				Description: "Get current weather",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"location": map[string]any{"type": "string"},
					},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunctionDefinition{
				Name:        "set_alarm",
				Description: "Set an alarm",
				Parameters: map[string]any{
					"type": "object",
				},
			},
		},
	}

	prompt := buildCLIToolsPrompt(tools)

	if !contains(prompt, "get_weather") {
		t.Error("expected tool name 'get_weather' in prompt")
	}
	if !contains(prompt, "Get current weather") {
		t.Error("expected tool description in prompt")
	}
	if !contains(prompt, "set_alarm") {
		t.Error("expected tool name 'set_alarm' in prompt")
	}
	if !contains(prompt, "json") {
		t.Error("expected 'json' format mention in prompt")
	}
}

func TestBuildCLIToolsPrompt_NonFunctionTypeIgnored(t *testing.T) {
	tools := []ToolDefinition{
		{
			Type: "non-function",
			Function: ToolFunctionDefinition{
				Name: "ignored_tool",
			},
		},
		{
			Type: "function",
			Function: ToolFunctionDefinition{
				Name: "included_tool",
			},
		},
	}

	prompt := buildCLIToolsPrompt(tools)

	if contains(prompt, "ignored_tool") {
		t.Error("expected non-function type to be ignored")
	}
	if !contains(prompt, "included_tool") {
		t.Error("expected function type to be included")
	}
}

func TestBuildCLIToolsPrompt_ToolWithoutDescription(t *testing.T) {
	tools := []ToolDefinition{
		{
			Type: "function",
			Function: ToolFunctionDefinition{
				Name:        "simple_tool",
				Description: "",
				Parameters:  map[string]any{},
			},
		},
	}

	prompt := buildCLIToolsPrompt(tools)

	if !contains(prompt, "simple_tool") {
		t.Error("expected tool name in prompt")
	}
	// Should not crash even with empty description
}

func TestBuildCLIToolsPrompt_IncludesJsonStructureHints(t *testing.T) {
	tools := []ToolDefinition{
		{
			Type: "function",
			Function: ToolFunctionDefinition{
				Name:        "test_tool",
				Description: "Test",
			},
		},
	}

	prompt := buildCLIToolsPrompt(tools)

	// Check for required structure hints
	if !contains(prompt, "tool_calls") {
		t.Error("expected 'tool_calls' in prompt")
	}
	if !contains(prompt, "arguments") {
		t.Error("expected 'arguments' in prompt")
	}
	if !contains(prompt, "Escaping rules") {
		t.Error("expected 'Escaping rules' in prompt")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

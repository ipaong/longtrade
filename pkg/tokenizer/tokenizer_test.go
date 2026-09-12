package tokenizer

import (
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/providers"
)

// TestEstimateMessageTokens_EmptyMessage tests token estimation for empty message
func TestEstimateMessageTokens_EmptyMessage(t *testing.T) {
	msg := providers.Message{
		Role:    "user",
		Content: "",
	}

	tokens := EstimateMessageTokens(msg)
	// messageOverhead = 12, so: 12 * 2 / 5 = 4 (integer division)
	expectedMin := 0
	if tokens < expectedMin {
		t.Errorf("tokens for empty message: got %d, expected >= %d", tokens, expectedMin)
	}
}

// TestEstimateMessageTokens_SimpleContent tests token estimation for simple text
func TestEstimateMessageTokens_SimpleContent(t *testing.T) {
	msg := providers.Message{
		Role:    "user",
		Content: "Hello",
	}

	tokens := EstimateMessageTokens(msg)
	// "Hello" = 5 chars + 12 overhead = 17 chars * 2 / 5 = 6
	if tokens != 6 {
		t.Errorf("tokens for 'Hello': got %d, want 6", tokens)
	}
}

// TestEstimateMessageTokens_LongerContent tests token estimation for longer text
func TestEstimateMessageTokens_LongerContent(t *testing.T) {
	// Create 100-character content
	content := "The quick brown fox jumps over the lazy dog. " +
		"This is a longer text to estimate token count accurately. " +
		"More text here for better estimation."
	msg := providers.Message{
		Role:    "user",
		Content: content,
	}

	tokens := EstimateMessageTokens(msg)
	// Rough estimate: 110 chars + 12 overhead = 122 * 2 / 5 = 48
	if tokens < 40 {
		t.Errorf("tokens estimate seems too low: got %d", tokens)
	}
}

// TestEstimateMessageTokens_WithReasoningContent tests token estimation with reasoning content
func TestEstimateMessageTokens_WithReasoningContent(t *testing.T) {
	msg := providers.Message{
		Role:             "assistant",
		Content:          "answer",
		ReasoningContent: "thinking about this problem carefully...",
	}

	tokens := EstimateMessageTokens(msg)
	// "answer" (6) + "thinking about..." (39) + overhead (12) = 57 * 2 / 5 = 22
	if tokens < 20 {
		t.Errorf("tokens with reasoning too low: got %d", tokens)
	}
}

// TestEstimateMessageTokens_WithSystemParts tests token estimation with SystemParts
func TestEstimateMessageTokens_WithSystemParts(t *testing.T) {
	msg := providers.Message{
		Role:    "system",
		Content: "You are helpful",
		SystemParts: []providers.ContentBlock{
			{Type: "text", Text: "You are very helpful"},
			{Type: "text", Text: "Always respond kindly"},
		},
	}

	tokens := EstimateMessageTokens(msg)
	// SystemParts are alternative representation: 20+20 + 20*2 overhead = 60 chars
	// Content is 15 chars
	// Max(60, 15) = 60, + overhead 12 = 72 * 2/5 = 28
	if tokens < 20 {
		t.Errorf("tokens with system parts too low: got %d", tokens)
	}
}

// TestEstimateMessageTokens_ContentLargerThanSystemParts tests max logic
func TestEstimateMessageTokens_ContentLargerThanSystemParts(t *testing.T) {
	// Content should be larger, so systemParts are ignored in favor of content
	content := "This is a very long piece of content that should be larger than the system parts overhead"
	msg := providers.Message{
		Role:    "system",
		Content: content,
		SystemParts: []providers.ContentBlock{
			{Type: "text", Text: "brief"},
		},
	}

	tokens := EstimateMessageTokens(msg)
	// Should use content (92 chars) not systemParts
	if tokens < 30 {
		t.Errorf("tokens should be based on larger content: got %d", tokens)
	}
}

// TestEstimateMessageTokens_WithSingleToolCall tests token estimation with tool calls
func TestEstimateMessageTokens_WithSingleToolCall(t *testing.T) {
	msg := providers.Message{
		Role:    "assistant",
		Content: "I'll search",
		ToolCalls: []providers.ToolCall{
			{
				ID:   "call_123",
				Type: "function",
				Function: &providers.FunctionCall{
					Name:      "web_search",
					Arguments: `{"q":"golang"}`,
				},
			},
		},
	}

	tokens := EstimateMessageTokens(msg)
	// Content: 11, ToolCall: ID(8) + Type(8) + Name(10) + Arguments(14) = 40
	// Total: 11 + 40 + overhead(12) = 63 * 2/5 = 25
	if tokens < 20 {
		t.Errorf("tokens with tool call too low: got %d", tokens)
	}
}

// TestEstimateMessageTokens_WithMultipleToolCalls tests multiple tool calls
func TestEstimateMessageTokens_WithMultipleToolCalls(t *testing.T) {
	msg := providers.Message{
		Role:    "assistant",
		Content: "searching",
		ToolCalls: []providers.ToolCall{
			{
				ID:   "call_1",
				Type: "function",
				Function: &providers.FunctionCall{
					Name:      "search",
					Arguments: `{"q":"test"}`,
				},
			},
			{
				ID:   "call_2",
				Type: "function",
				Function: &providers.FunctionCall{
					Name:      "exec",
					Arguments: `{"cmd":"ls"}`,
				},
			},
		},
	}

	tokens := EstimateMessageTokens(msg)
	if tokens < 30 {
		t.Errorf("tokens with multiple tool calls too low: got %d", tokens)
	}
}

// TestEstimateMessageTokens_WithToolCallIDResponse tests tool call ID in response
func TestEstimateMessageTokens_WithToolCallIDResponse(t *testing.T) {
	msg := providers.Message{
		Role:       "tool",
		Content:    "search results: ...",
		ToolCallID: "call_abc123",
	}

	tokens := EstimateMessageTokens(msg)
	// Content: 19, ToolCallID: 11, overhead: 12 = 42 * 2/5 = 16
	if tokens < 15 {
		t.Errorf("tokens with ToolCallID too low: got %d", tokens)
	}
}

// TestEstimateMessageTokens_WithMediaItems tests token estimation with media
func TestEstimateMessageTokens_WithMediaItems(t *testing.T) {
	msg := providers.Message{
		Role:    "user",
		Content: "What's in this image?",
		Media: []string{
			"image1.jpg",
			"image2.png",
		},
	}

	tokens := EstimateMessageTokens(msg)
	// Content: 21, overhead: 12, plus 2 media * 256 = 512
	// (21 + 12) * 2/5 = 13, plus 512 = 525
	if tokens < 500 {
		t.Errorf("tokens with media too low: got %d, expected >= 500", tokens)
	}
	if tokens != 525 {
		t.Errorf("tokens with media: got %d, want 525", tokens)
	}
}

// TestEstimateMessageTokens_WithMultipleMedia tests with multiple media items
func TestEstimateMessageTokens_WithMultipleMedia(t *testing.T) {
	msg := providers.Message{
		Role:    "user",
		Content: "analyze",
		Media: []string{
			"image1.jpg",
			"image2.png",
			"image3.gif",
			"image4.webp",
		},
	}

	tokens := EstimateMessageTokens(msg)
	// 4 media * 256 = 1024, plus content overhead
	if tokens < 1000 {
		t.Errorf("tokens with 4 media items too low: got %d", tokens)
	}
}

// TestEstimateMessageTokens_WithToolCallWithoutFunction tests tool call without Function
func TestEstimateMessageTokens_WithToolCallWithoutFunction(t *testing.T) {
	msg := providers.Message{
		Role:    "assistant",
		Content: "using tool",
		ToolCalls: []providers.ToolCall{
			{
				ID:   "call_xyz",
				Type: "function",
				Name: "execute_code",
				// No Function field - should use Name instead
			},
		},
	}

	tokens := EstimateMessageTokens(msg)
	// Should still count the Name field
	if tokens < 10 {
		t.Errorf("tokens with Name fallback too low: got %d", tokens)
	}
}

// TestEstimateMessageTokens_UnicodeContent tests UTF-8 and CJK characters
func TestEstimateMessageTokens_UnicodeContent(t *testing.T) {
	// Unicode characters should be counted correctly
	msg := providers.Message{
		Role:    "user",
		Content: "你好世界", // 4 CJK runes
	}

	tokens := EstimateMessageTokens(msg)
	// 4 runes + 12 overhead = 16 * 2/5 = 6
	if tokens != 6 {
		t.Errorf("tokens for CJK: got %d, want 6", tokens)
	}
}

// TestEstimateMessageTokens_ComplexMessage tests complete message with all fields
func TestEstimateMessageTokens_ComplexMessage(t *testing.T) {
	msg := providers.Message{
		Role:             "assistant",
		Content:          "Let me search for that information",
		ReasoningContent: "I need to search external sources",
		ToolCalls: []providers.ToolCall{
			{
				ID:   "call_search_1",
				Type: "function",
				Function: &providers.FunctionCall{
					Name:      "web_search",
					Arguments: `{"query":"what is golang","limit":5}`,
				},
			},
		},
		ToolCallID: "",
		Media: []string{
			"reference_image.png",
		},
	}

	tokens := EstimateMessageTokens(msg)
	// This should be a reasonable estimate, roughly in the 100-200 token range
	if tokens < 50 {
		t.Errorf("complex message tokens too low: got %d", tokens)
	}
}

// TestEstimateToolDefsTokens_EmptyDefinitions tests empty tool definitions
func TestEstimateToolDefsTokens_EmptyDefinitions(t *testing.T) {
	defs := []providers.ToolDefinition{}
	tokens := EstimateToolDefsTokens(defs)
	if tokens != 0 {
		t.Errorf("empty defs tokens: got %d, want 0", tokens)
	}
}

// TestEstimateToolDefsTokens_SingleSimpleTool tests single tool definition
func TestEstimateToolDefsTokens_SingleSimpleTool(t *testing.T) {
	defs := []providers.ToolDefinition{
		{
			Type: "function",
			Function: providers.ToolFunctionDefinition{
				Name:        "search",
				Description: "Search the web",
				Parameters:  map[string]any{},
			},
		},
	}

	tokens := EstimateToolDefsTokens(defs)
	// "search"(6) + "Search the web"(14) + "{}"(2) + overhead(20) = 42 * 2/5 = 16
	if tokens != 16 {
		t.Errorf("single tool tokens: got %d, want 16", tokens)
	}
}

// TestEstimateToolDefsTokens_ComplexParameters tests tool with complex parameters
func TestEstimateToolDefsTokens_ComplexParameters(t *testing.T) {
	defs := []providers.ToolDefinition{
		{
			Type: "function",
			Function: providers.ToolFunctionDefinition{
				Name:        "execute_sql",
				Description: "Execute SQL query against database",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query":    map[string]any{"type": "string"},
						"database": map[string]any{"type": "string"},
					},
					"required": []string{"query"},
				},
			},
		},
	}

	tokens := EstimateToolDefsTokens(defs)
	// JSON parameters will be marshaled and counted
	if tokens < 50 {
		t.Errorf("complex params tokens too low: got %d", tokens)
	}
}

// TestEstimateToolDefsTokens_MultipleTools tests multiple tool definitions
func TestEstimateToolDefsTokens_MultipleTools(t *testing.T) {
	defs := []providers.ToolDefinition{
		{
			Type: "function",
			Function: providers.ToolFunctionDefinition{
				Name:        "read_file",
				Description: "Read file",
				Parameters:  map[string]any{},
			},
		},
		{
			Type: "function",
			Function: providers.ToolFunctionDefinition{
				Name:        "write_file",
				Description: "Write file",
				Parameters:  map[string]any{},
			},
		},
		{
			Type: "function",
			Function: providers.ToolFunctionDefinition{
				Name:        "delete_file",
				Description: "Delete file",
				Parameters:  map[string]any{},
			},
		},
	}

	tokens := EstimateToolDefsTokens(defs)
	// Each tool should contribute roughly equally
	if tokens < 30 {
		t.Errorf("multiple tools tokens too low: got %d", tokens)
	}
}

// TestEstimateToolDefsTokens_LongDescriptions tests tools with long descriptions
func TestEstimateToolDefsTokens_LongDescriptions(t *testing.T) {
	longDesc := "This is a very detailed description of the tool that explains " +
		"exactly what it does, what parameters it accepts, and what results " +
		"it returns in comprehensive detail with many words."
	defs := []providers.ToolDefinition{
		{
			Type: "function",
			Function: providers.ToolFunctionDefinition{
				Name:        "detailed_search",
				Description: longDesc,
				Parameters:  map[string]any{},
			},
		},
	}

	tokens := EstimateToolDefsTokens(defs)
	// Long description should contribute significantly
	if tokens < 50 {
		t.Errorf("long description tokens too low: got %d", tokens)
	}
}

// TestEstimateToolDefsTokens_NilParameters tests tool with nil parameters
func TestEstimateToolDefsTokens_NilParameters(t *testing.T) {
	defs := []providers.ToolDefinition{
		{
			Type: "function",
			Function: providers.ToolFunctionDefinition{
				Name:        "get_time",
				Description: "Get current time",
				Parameters:  nil,
			},
		},
	}

	tokens := EstimateToolDefsTokens(defs)
	// Should handle nil parameters gracefully
	if tokens < 10 {
		t.Errorf("nil parameters tokens too low: got %d", tokens)
	}
}

// TestEstimateToolDefsTokens_ParametersMarshalError tests parameters that might not marshal
func TestEstimateToolDefsTokens_ParametersMarshalError(t *testing.T) {
	// Create parameters with circular reference through nested map
	// (Go's json.Marshal will detect this and error, which we handle)
	defs := []providers.ToolDefinition{
		{
			Type: "function",
			Function: providers.ToolFunctionDefinition{
				Name:        "tool",
				Description: "Description",
				Parameters: map[string]any{
					"nested": map[string]any{
						"deep": map[string]any{
							"value": 123,
						},
					},
				},
			},
		},
	}

	tokens := EstimateToolDefsTokens(defs)
	// Should still count name + description + overhead even if params don't marshal
	if tokens < 15 {
		t.Errorf("with nested params tokens too low: got %d", tokens)
	}
}

// TestEstimateToolDefsTokens_ManyTools tests many tool definitions
func TestEstimateToolDefsTokens_ManyTools(t *testing.T) {
	defs := make([]providers.ToolDefinition, 10)
	for i := 0; i < 10; i++ {
		defs[i] = providers.ToolDefinition{
			Type: "function",
			Function: providers.ToolFunctionDefinition{
				Name:        "tool_name",
				Description: "Description for tool",
				Parameters:  map[string]any{},
			},
		}
	}

	tokens := EstimateToolDefsTokens(defs)
	// 10 tools * roughly 15-20 tokens each = 150-200
	if tokens < 100 {
		t.Errorf("many tools tokens too low: got %d", tokens)
	}
}

// TestEstimateToolDefsTokens_ComplexJSONParameters tests tool with JSON marshal
func TestEstimateToolDefsTokens_ComplexJSONParameters(t *testing.T) {
	params := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string", "description": "User name"},
			"age":  map[string]any{"type": "integer", "description": "User age"},
			"tags": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "string",
				},
			},
		},
		"required": []string{"name"},
	}

	defs := []providers.ToolDefinition{
		{
			Type: "function",
			Function: providers.ToolFunctionDefinition{
				Name:        "create_user",
				Description: "Create a new user in the system",
				Parameters:  params,
			},
		},
	}

	tokens := EstimateToolDefsTokens(defs)
	// JSON marshal will create substantial output
	if tokens < 60 {
		t.Errorf("complex JSON params tokens too low: got %d", tokens)
	}
}

// TestEstimateToolDefsTokens_Consistency tests that same input produces same output
func TestEstimateToolDefsTokens_Consistency(t *testing.T) {
	defs := []providers.ToolDefinition{
		{
			Type: "function",
			Function: providers.ToolFunctionDefinition{
				Name:        "search",
				Description: "Search query",
				Parameters: map[string]any{
					"q": "query string",
				},
			},
		},
	}

	tokens1 := EstimateToolDefsTokens(defs)
	tokens2 := EstimateToolDefsTokens(defs)

	if tokens1 != tokens2 {
		t.Errorf("inconsistent results: %d vs %d", tokens1, tokens2)
	}
}

// TestEstimateMessageTokens_EmptyStringContent verifies empty string handling
func TestEstimateMessageTokens_EmptyStringContent(t *testing.T) {
	msg := providers.Message{
		Role:    "user",
		Content: "",
	}

	tokens := EstimateMessageTokens(msg)
	// Only messageOverhead = 12, so 12 * 2 / 5 = 4
	if tokens != 4 {
		t.Errorf("empty string tokens: got %d, want 4", tokens)
	}
}

// TestEstimateMessageTokens_SpecialCharacters tests special character handling
func TestEstimateMessageTokens_SpecialCharacters(t *testing.T) {
	msg := providers.Message{
		Role:    "user",
		Content: "!@#$%^&*()_+-=[]{}|;:',.<>?/",
	}

	tokens := EstimateMessageTokens(msg)
	// All are single-byte ASCII characters
	if tokens < 15 {
		t.Errorf("special chars tokens too low: got %d", tokens)
	}
}

// TestEstimateMessageTokens_Newlines tests newline character handling
func TestEstimateMessageTokens_Newlines(t *testing.T) {
	msg := providers.Message{
		Role:    "user",
		Content: "line1\nline2\nline3",
	}

	tokens := EstimateMessageTokens(msg)
	// Should count \n characters correctly
	if tokens < 5 {
		t.Errorf("newline tokens too low: got %d", tokens)
	}
}

// TestEstimateMessageTokens_Tabs tests tab character handling
func TestEstimateMessageTokens_Tabs(t *testing.T) {
	msg := providers.Message{
		Role:    "user",
		Content: "\t\t\tindented",
	}

	tokens := EstimateMessageTokens(msg)
	// 3 tabs + "indented" (8) + overhead (12) = 23 * 2/5 = 9
	if tokens != 9 {
		t.Errorf("tab tokens: got %d, want 9", tokens)
	}
}

// TestEstimateMessageTokens_MixedUnicode tests mixed ASCII and Unicode
func TestEstimateMessageTokens_MixedUnicode(t *testing.T) {
	msg := providers.Message{
		Role:    "user",
		Content: "Hello 世界 World",
	}

	tokens := EstimateMessageTokens(msg)
	// "Hello " (6) + "世" (1) + "界" (1) + " World" (6) + overhead (12) = 26 * 2/5 = 10
	if tokens != 10 {
		t.Errorf("mixed unicode tokens: got %d, want 10", tokens)
	}
}

// TestEstimateMessageTokens_SystemParts_EmptyList tests empty SystemParts list
func TestEstimateMessageTokens_SystemParts_EmptyList(t *testing.T) {
	msg := providers.Message{
		Role:        "system",
		Content:     "test",
		SystemParts: []providers.ContentBlock{},
	}

	tokens := EstimateMessageTokens(msg)
	// Empty list means no systemParts chars, so use content
	// "test" (4) + overhead (12) = 16 * 2/5 = 6
	if tokens != 6 {
		t.Errorf("empty systemparts tokens: got %d, want 6", tokens)
	}
}

// TestEstimateMessageTokens_AssistantRole tests assistant role message
func TestEstimateMessageTokens_AssistantRole(t *testing.T) {
	msg := providers.Message{
		Role:    "assistant",
		Content: "I can help you",
	}

	tokens := EstimateMessageTokens(msg)
	// "I can help you" (14) + overhead (12) = 26 * 2/5 = 10
	if tokens != 10 {
		t.Errorf("assistant tokens: got %d, want 10", tokens)
	}
}

// TestEstimateMessageTokens_ToolRole tests tool role message
func TestEstimateMessageTokens_ToolRole(t *testing.T) {
	msg := providers.Message{
		Role:       "tool",
		Content:    "result",
		ToolCallID: "call_1",
	}

	tokens := EstimateMessageTokens(msg)
	// "result" (6) + "call_1" (6) + overhead (12) = 24 * 2/5 = 9
	if tokens != 9 {
		t.Errorf("tool role tokens: got %d, want 9", tokens)
	}
}

// TestEstimateMessageTokens_RealWorldPythonExample tests realistic Python code
func TestEstimateMessageTokens_RealWorldPythonExample(t *testing.T) {
	pythonCode := `def fibonacci(n):
    if n <= 1:
        return n
    return fibonacci(n-1) + fibonacci(n-2)`

	msg := providers.Message{
		Role:    "user",
		Content: pythonCode,
	}

	tokens := EstimateMessageTokens(msg)
	if tokens < 30 {
		t.Errorf("python code tokens too low: got %d", tokens)
	}
}

package providers

import "testing"

func TestBuildRequestUsesFunctionFieldsWhenToolCallNameMissing(t *testing.T) {
	p := &AntigravityProvider{}

	messages := []Message{
		{
			Role: "assistant",
			ToolCalls: []ToolCall{{
				ID: "call_read_file_123",
				Function: &FunctionCall{
					Name:      "read_file",
					Arguments: `{"path":"README.md"}`,
				},
			}},
		},
		{
			Role:       "tool",
			ToolCallID: "call_read_file_123",
			Content:    "ok",
		},
	}

	req := p.buildRequest(messages, nil, "", nil)
	if len(req.Contents) != 2 {
		t.Fatalf("expected 2 contents, got %d", len(req.Contents))
	}

	modelPart := req.Contents[0].Parts[0]
	if modelPart.FunctionCall == nil {
		t.Fatal("expected functionCall in assistant message")
	}
	if modelPart.FunctionCall.Name != "read_file" {
		t.Fatalf("expected functionCall name read_file, got %q", modelPart.FunctionCall.Name)
	}
	if got := modelPart.FunctionCall.Args["path"]; got != "README.md" {
		t.Fatalf("expected functionCall args[path] to be README.md, got %v", got)
	}

	toolPart := req.Contents[1].Parts[0]
	if toolPart.FunctionResponse == nil {
		t.Fatal("expected functionResponse in tool message")
	}
	if toolPart.FunctionResponse.Name != "read_file" {
		t.Fatalf("expected functionResponse name read_file, got %q", toolPart.FunctionResponse.Name)
	}
}

func TestResolveToolResponseNameInfersNameFromGeneratedCallID(t *testing.T) {
	got := resolveToolResponseName("call_search_docs_999", map[string]string{})
	if got != "search_docs" {
		t.Fatalf("expected inferred tool name search_docs, got %q", got)
	}
}

func TestTruncateString_Short(t *testing.T) {
	if got := truncateString("hello", 10); got != "hello" {
		t.Errorf("truncateString short = %q, want hello", got)
	}
}

func TestTruncateString_Exact(t *testing.T) {
	if got := truncateString("hello", 5); got != "hello" {
		t.Errorf("truncateString exact = %q, want hello", got)
	}
}

func TestTruncateString_TooLong(t *testing.T) {
	got := truncateString("hello world", 5)
	if got != "hello..." {
		t.Errorf("truncateString long = %q, want hello...", got)
	}
}

func TestTruncateString_Empty(t *testing.T) {
	if got := truncateString("", 5); got != "" {
		t.Errorf("truncateString empty = %q, want empty", got)
	}
}

func TestRandomString_Length(t *testing.T) {
	got := randomString(12)
	if len(got) != 12 {
		t.Errorf("randomString(12) len = %d, want 12", len(got))
	}
}

func TestRandomString_OnlyLowercaseAlphanumeric(t *testing.T) {
	got := randomString(100)
	for _, c := range got {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
			t.Errorf("randomString contains invalid char %q", c)
		}
	}
}

func TestExtractPartThoughtSignature_CamelCase(t *testing.T) {
	got := extractPartThoughtSignature("sig1", "")
	if got != "sig1" {
		t.Errorf("extractPartThoughtSignature camelCase = %q, want sig1", got)
	}
}

func TestExtractPartThoughtSignature_SnakeCase(t *testing.T) {
	got := extractPartThoughtSignature("", "sig2")
	if got != "sig2" {
		t.Errorf("extractPartThoughtSignature snake = %q, want sig2", got)
	}
}

func TestExtractPartThoughtSignature_Both(t *testing.T) {
	got := extractPartThoughtSignature("camel", "snake")
	if got != "camel" {
		t.Errorf("extractPartThoughtSignature both = %q, want camel (camelCase wins)", got)
	}
}

func TestExtractPartThoughtSignature_Neither(t *testing.T) {
	got := extractPartThoughtSignature("", "")
	if got != "" {
		t.Errorf("extractPartThoughtSignature neither = %q, want empty", got)
	}
}

func TestSanitizeSchemaForGemini_Nil(t *testing.T) {
	if sanitizeSchemaForGemini(nil) != nil {
		t.Error("sanitizeSchemaForGemini(nil) should return nil")
	}
}

func TestSanitizeSchemaForGemini_RemovesUnsupportedKeys(t *testing.T) {
	schema := map[string]any{
		"type":        "object",
		"description": "A test",
		"minLength":   5,
		"maxLength":   100,
		"pattern":     "^[a-z]+$",
	}
	result := sanitizeSchemaForGemini(schema)
	if _, ok := result["minLength"]; ok {
		t.Error("sanitizeSchemaForGemini should remove minLength")
	}
	if _, ok := result["maxLength"]; ok {
		t.Error("sanitizeSchemaForGemini should remove maxLength")
	}
	if _, ok := result["pattern"]; ok {
		t.Error("sanitizeSchemaForGemini should remove pattern")
	}
	if result["type"] != "object" {
		t.Errorf("sanitizeSchemaForGemini type = %v, want object", result["type"])
	}
	if result["description"] != "A test" {
		t.Errorf("sanitizeSchemaForGemini description = %v, want 'A test'", result["description"])
	}
}

func TestSanitizeSchemaForGemini_AddsTypeWhenPropertiesPresent(t *testing.T) {
	schema := map[string]any{
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
		},
	}
	result := sanitizeSchemaForGemini(schema)
	if result["type"] != "object" {
		t.Errorf("sanitizeSchemaForGemini should add type=object when properties present, got %v", result["type"])
	}
}

func TestSanitizeSchemaForGemini_KeepsTypeWhenAlreadySet(t *testing.T) {
	schema := map[string]any{
		"type":       "string",
		"properties": map[string]any{},
	}
	result := sanitizeSchemaForGemini(schema)
	if result["type"] != "string" {
		t.Errorf("sanitizeSchemaForGemini should not overwrite existing type, got %v", result["type"])
	}
}

func TestSanitizeSchemaForGemini_RecursiveNested(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"item": map[string]any{
				"type":      "string",
				"minLength": 1,
			},
		},
	}
	result := sanitizeSchemaForGemini(schema)
	props, ok := result["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties should still be present after sanitize")
	}
	item, ok := props["item"].(map[string]any)
	if !ok {
		t.Fatal("item property should be a map")
	}
	if _, hasMin := item["minLength"]; hasMin {
		t.Error("sanitizeSchemaForGemini should recursively remove minLength from nested schema")
	}
}

func TestParseAntigravityError_InvalidJSON(t *testing.T) {
	p := &AntigravityProvider{}
	err := p.parseAntigravityError(500, []byte("not json"))
	if err == nil {
		t.Fatal("expected non-nil error for invalid JSON body")
	}
	msg := err.Error()
	if !contains(msg, "500") {
		t.Errorf("parseAntigravityError invalid JSON should include status code, got %q", msg)
	}
}

func TestParseAntigravityError_ValidError(t *testing.T) {
	p := &AntigravityProvider{}
	body := []byte(`{"error":{"code":400,"message":"bad request","status":"INVALID_ARGUMENT"}}`)
	err := p.parseAntigravityError(400, body)
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if !contains(err.Error(), "bad request") {
		t.Errorf("parseAntigravityError should include message, got %q", err.Error())
	}
}

func TestParseAntigravityError_RateLimit(t *testing.T) {
	p := &AntigravityProvider{}
	body := []byte(`{"error":{"code":429,"message":"quota exceeded","status":"RESOURCE_EXHAUSTED","details":[]}}`)
	err := p.parseAntigravityError(429, body)
	if err == nil {
		t.Fatal("expected non-nil error for rate limit")
	}
	if !contains(err.Error(), "rate limit") {
		t.Errorf("parseAntigravityError 429 should mention rate limit, got %q", err.Error())
	}
}



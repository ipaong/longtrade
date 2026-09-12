package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/cryptoquantumwave/khunquant/pkg/media"
)

// MockMCPManager is a mock implementation of MCPManager interface for testing
type MockMCPManager struct {
	callToolFunc func(ctx context.Context, serverName, toolName string, arguments map[string]any) (*mcp.CallToolResult, error)
}

func (m *MockMCPManager) CallTool(
	ctx context.Context,
	serverName, toolName string,
	arguments map[string]any,
) (*mcp.CallToolResult, error) {
	if m.callToolFunc != nil {
		return m.callToolFunc(ctx, serverName, toolName, arguments)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "mock result"},
		},
		IsError: false,
	}, nil
}

// TestNewMCPTool verifies MCP tool creation
func TestNewMCPTool(t *testing.T) {
	manager := &MockMCPManager{}
	tool := &mcp.Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"input": map[string]any{
					"type":        "string",
					"description": "Test input",
				},
			},
		},
	}

	mcpTool := NewMCPTool(manager, "test_server", tool)

	if mcpTool == nil {
		t.Fatal("NewMCPTool should not return nil")
	}
	// Verify tool properties we can access
	if mcpTool.Name() != "mcp_test_server_test_tool" {
		t.Errorf("Expected tool name with prefix, got '%s'", mcpTool.Name())
	}
}

// TestMCPTool_Name verifies tool name with server prefix
func TestMCPTool_Name(t *testing.T) {
	tests := []struct {
		name       string
		serverName string
		toolName   string
		expected   string
	}{
		{
			name:       "simple name",
			serverName: "github",
			toolName:   "create_issue",
			expected:   "mcp_github_create_issue",
		},
		{
			name:       "filesystem server",
			serverName: "filesystem",
			toolName:   "read_file",
			expected:   "mcp_filesystem_read_file",
		},
		{
			name:       "remote server",
			serverName: "remote-api",
			toolName:   "fetch_data",
			expected:   "mcp_remote-api_fetch_data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &MockMCPManager{}
			tool := &mcp.Tool{Name: tt.toolName}
			mcpTool := NewMCPTool(manager, tt.serverName, tool)

			result := mcpTool.Name()
			if result != tt.expected {
				t.Errorf("Expected name '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestMCPTool_Description verifies tool description generation
func TestMCPTool_Description(t *testing.T) {
	tests := []struct {
		name            string
		serverName      string
		toolDescription string
		expectContains  []string
	}{
		{
			name:            "with description",
			serverName:      "github",
			toolDescription: "Create a GitHub issue",
			expectContains:  []string{"[MCP:github]", "Create a GitHub issue"},
		},
		{
			name:            "empty description",
			serverName:      "filesystem",
			toolDescription: "",
			expectContains:  []string{"[MCP:filesystem]", "MCP tool from filesystem server"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &MockMCPManager{}
			tool := &mcp.Tool{
				Name:        "test_tool",
				Description: tt.toolDescription,
			}
			mcpTool := NewMCPTool(manager, tt.serverName, tool)

			result := mcpTool.Description()

			for _, expected := range tt.expectContains {
				if !strings.Contains(result, expected) {
					t.Errorf("Description should contain '%s', got: %s", expected, result)
				}
			}
		})
	}
}

// TestMCPTool_Parameters verifies parameter schema conversion
func TestMCPTool_Parameters(t *testing.T) {
	tests := []struct {
		name           string
		inputSchema    any
		expectType     string
		checkProperty  string
		expectProperty bool
	}{
		{
			name: "map schema",
			inputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Search query",
					},
				},
				"required": []string{"query"},
			},
			expectType:     "object",
			checkProperty:  "query",
			expectProperty: true,
		},
		{
			name:           "nil schema",
			inputSchema:    nil,
			expectType:     "object",
			expectProperty: false,
		},
		{
			name: "json.RawMessage schema",
			inputSchema: []byte(`{
				"type": "object",
				"properties": {
					"repo": {
						"type": "string",
						"description": "Repository name"
					},
					"stars": {
						"type": "integer",
						"description": "Minimum stars"
					}
				},
				"required": ["repo"]
			}`),
			expectType:     "object",
			checkProperty:  "repo",
			expectProperty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &MockMCPManager{}
			tool := &mcp.Tool{
				Name:        "test_tool",
				InputSchema: tt.inputSchema,
			}
			mcpTool := NewMCPTool(manager, "test_server", tool)

			params := mcpTool.Parameters()

			if params == nil {
				t.Fatal("Parameters should not be nil")
			}

			if params["type"] != tt.expectType {
				t.Errorf("Expected type '%s', got '%v'", tt.expectType, params["type"])
			}

			// Check if property exists when expected
			if tt.checkProperty != "" {
				properties, ok := params["properties"].(map[string]any)
				if !ok && tt.expectProperty {
					t.Errorf("Expected properties to be a map")
					return
				}
				if ok {
					_, hasProperty := properties[tt.checkProperty]
					if hasProperty != tt.expectProperty {
						t.Errorf("Expected property '%s' existence: %v, got: %v",
							tt.checkProperty, tt.expectProperty, hasProperty)
					}
				}
			}
		})
	}
}

// TestMCPTool_Execute_Success tests successful tool execution
func TestMCPTool_Execute_Success(t *testing.T) {
	manager := &MockMCPManager{
		callToolFunc: func(ctx context.Context, serverName, toolName string, arguments map[string]any) (*mcp.CallToolResult, error) {
			// Verify correct parameters passed
			if serverName != "github" {
				t.Errorf("Expected serverName 'github', got '%s'", serverName)
			}
			if toolName != "search_repos" {
				t.Errorf("Expected toolName 'search_repos', got '%s'", toolName)
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: "Found 3 repositories"},
				},
				IsError: false,
			}, nil
		},
	}

	tool := &mcp.Tool{
		Name:        "search_repos",
		Description: "Search GitHub repositories",
	}
	mcpTool := NewMCPTool(manager, "github", tool)

	ctx := context.Background()
	args := map[string]any{
		"query": "golang mcp",
	}

	result := mcpTool.Execute(ctx, args)

	if result == nil {
		t.Fatal("Result should not be nil")
	}
	if result.IsError {
		t.Errorf("Expected no error, got error: %s", result.ForLLM)
	}
	if result.ForLLM != "Found 3 repositories" {
		t.Errorf("Expected 'Found 3 repositories', got '%s'", result.ForLLM)
	}
}

// TestMCPTool_Execute_ManagerError tests execution when manager returns error
func TestMCPTool_Execute_ManagerError(t *testing.T) {
	manager := &MockMCPManager{
		callToolFunc: func(ctx context.Context, serverName, toolName string, arguments map[string]any) (*mcp.CallToolResult, error) {
			return nil, fmt.Errorf("connection failed")
		},
	}

	tool := &mcp.Tool{Name: "test_tool"}
	mcpTool := NewMCPTool(manager, "test_server", tool)

	ctx := context.Background()
	result := mcpTool.Execute(ctx, map[string]any{})

	if result == nil {
		t.Fatal("Result should not be nil")
	}
	if !result.IsError {
		t.Error("Expected IsError to be true")
	}
	if !strings.Contains(result.ForLLM, "MCP tool execution failed") {
		t.Errorf("Error message should mention execution failure, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "connection failed") {
		t.Errorf("Error message should include original error, got: %s", result.ForLLM)
	}
}

// TestMCPTool_Execute_ServerError tests execution when server returns error
func TestMCPTool_Execute_ServerError(t *testing.T) {
	manager := &MockMCPManager{
		callToolFunc: func(ctx context.Context, serverName, toolName string, arguments map[string]any) (*mcp.CallToolResult, error) {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: "Invalid API key"},
				},
				IsError: true,
			}, nil
		},
	}

	tool := &mcp.Tool{Name: "test_tool"}
	mcpTool := NewMCPTool(manager, "test_server", tool)

	ctx := context.Background()
	result := mcpTool.Execute(ctx, map[string]any{})

	if result == nil {
		t.Fatal("Result should not be nil")
	}
	if !result.IsError {
		t.Error("Expected IsError to be true")
	}
	if !strings.Contains(result.ForLLM, "MCP tool returned error") {
		t.Errorf("Error message should mention server error, got: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "Invalid API key") {
		t.Errorf("Error message should include server message, got: %s", result.ForLLM)
	}
}

// TestMCPTool_Execute_MultipleContent tests execution with multiple content items
func TestMCPTool_Execute_MultipleContent(t *testing.T) {
	manager := &MockMCPManager{
		callToolFunc: func(ctx context.Context, serverName, toolName string, arguments map[string]any) (*mcp.CallToolResult, error) {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: "First line"},
					&mcp.TextContent{Text: "Second line"},
					&mcp.TextContent{Text: "Third line"},
				},
				IsError: false,
			}, nil
		},
	}

	tool := &mcp.Tool{Name: "multi_output"}
	mcpTool := NewMCPTool(manager, "test_server", tool)

	ctx := context.Background()
	result := mcpTool.Execute(ctx, map[string]any{})

	if result.IsError {
		t.Errorf("Expected no error, got: %s", result.ForLLM)
	}

	expected := "First line\nSecond line\nThird line"
	if result.ForLLM != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result.ForLLM)
	}
}

// TestExtractContentText_TextContent tests text content extraction
func TestExtractContentText_TextContent(t *testing.T) {
	content := []mcp.Content{
		&mcp.TextContent{Text: "Hello World"},
		&mcp.TextContent{Text: "Second message"},
	}

	result := extractContentText(content)
	expected := "Hello World\nSecond message"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestExtractContentText_ImageContent tests image content extraction
func TestExtractContentText_ImageContent(t *testing.T) {
	content := []mcp.Content{
		&mcp.ImageContent{
			Data:     []byte("base64data"),
			MIMEType: "image/png",
		},
	}

	result := extractContentText(content)

	if !strings.Contains(result, "[Image:") {
		t.Errorf("Expected image indicator, got: %s", result)
	}
	if !strings.Contains(result, "image/png") {
		t.Errorf("Expected MIME type in output, got: %s", result)
	}
}

// TestExtractContentText_MixedContent tests mixed content types
func TestExtractContentText_MixedContent(t *testing.T) {
	content := []mcp.Content{
		&mcp.TextContent{Text: "Description"},
		&mcp.ImageContent{
			Data:     []byte("data"),
			MIMEType: "image/jpeg",
		},
		&mcp.TextContent{Text: "More text"},
	}

	result := extractContentText(content)

	if !strings.Contains(result, "Description") {
		t.Errorf("Should contain text content, got: %s", result)
	}
	if !strings.Contains(result, "[Image:") {
		t.Errorf("Should contain image indicator, got: %s", result)
	}
	if !strings.Contains(result, "More text") {
		t.Errorf("Should contain second text, got: %s", result)
	}
}

// TestExtractContentText_EmptyContent tests empty content array
func TestExtractContentText_EmptyContent(t *testing.T) {
	content := []mcp.Content{}

	result := extractContentText(content)

	if result != "" {
		t.Errorf("Expected empty string for empty content, got: %s", result)
	}
}

// TestMCPTool_InterfaceCompliance verifies MCPTool implements Tool interface
func TestMCPTool_InterfaceCompliance(t *testing.T) {
	manager := &MockMCPManager{}
	tool := &mcp.Tool{Name: "test"}
	mcpTool := NewMCPTool(manager, "test_server", tool)

	// Verify it implements Tool interface
	var _ Tool = mcpTool
}

// TestMCPTool_Parameters_MapSchema tests schema that's already a map
func TestMCPTool_Parameters_MapSchema(t *testing.T) {
	manager := &MockMCPManager{}
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "The name parameter",
			},
		},
		"required": []string{"name"},
	}

	tool := &mcp.Tool{
		Name:        "test_tool",
		InputSchema: schema,
	}
	mcpTool := NewMCPTool(manager, "test_server", tool)

	params := mcpTool.Parameters()

	// Should return the schema as-is when it's already a map
	if params["type"] != "object" {
		t.Errorf("Expected type 'object', got '%v'", params["type"])
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Error("Properties should be a map")
	}

	nameParam, ok := props["name"].(map[string]any)
	if !ok {
		t.Error("Name parameter should exist")
	}

	if nameParam["type"] != "string" {
		t.Errorf("Name type should be 'string', got '%v'", nameParam["type"])
	}
}

func TestMCPTool_Execute_ImageContentStoredAsMedia(t *testing.T) {
	store := media.NewFileMediaStore()
	manager := &MockMCPManager{
		callToolFunc: func(ctx context.Context, serverName, toolName string, arguments map[string]any) (*mcp.CallToolResult, error) {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.ImageContent{
						Data:     []byte("fake-image-bytes"),
						MIMEType: "image/png",
					},
				},
			}, nil
		},
	}

	mcpTool := NewMCPTool(manager, "screenshoto", &mcp.Tool{Name: "take_screenshot"})
	mcpTool.SetMediaStore(store)

	result := mcpTool.Execute(WithToolContext(context.Background(), "telegram", "chat-42"), nil)

	if result.IsError {
		t.Fatalf("expected success, got %q", result.ForLLM)
	}
	if len(result.Media) != 1 {
		t.Fatalf("expected 1 media ref, got %d", len(result.Media))
	}
	if result.ResponseHandled {
		t.Fatal("expected MCP image artifact not to mark response as handled")
	}
	if !strings.Contains(result.ForLLM, "stored as a local media artifact") {
		t.Fatalf("expected local media artifact note, got %q", result.ForLLM)
	}

	path, meta, err := store.ResolveWithMeta(result.Media[0])
	if err != nil {
		t.Fatalf("expected stored media ref to resolve: %v", err)
	}
	if meta.ContentType != "image/png" {
		t.Fatalf("expected image/png content type, got %q", meta.ContentType)
	}
	if filepath.Ext(path) != ".png" {
		t.Fatalf("expected png temp file, got %q", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected stored media file to be readable: %v", err)
	}
	if string(data) != "fake-image-bytes" {
		t.Fatalf("expected stored media bytes to match input, got %q", string(data))
	}
}

func TestMCPTool_Execute_EmbeddedResourceBlobStoredAsMedia(t *testing.T) {
	store := media.NewFileMediaStore()
	manager := &MockMCPManager{
		callToolFunc: func(ctx context.Context, serverName, toolName string, arguments map[string]any) (*mcp.CallToolResult, error) {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.EmbeddedResource{
						Resource: &mcp.ResourceContents{
							URI:      "file:///tmp/report.png",
							MIMEType: "image/png",
							Blob:     []byte("blob-bytes"),
						},
					},
				},
			}, nil
		},
	}

	mcpTool := NewMCPTool(manager, "grafana", &mcp.Tool{Name: "get_dashboard_image"})
	mcpTool.SetMediaStore(store)

	result := mcpTool.Execute(WithToolContext(context.Background(), "telegram", "chat-42"), nil)

	if len(result.Media) != 1 {
		t.Fatalf("expected embedded resource blob to be stored as media, got %d refs", len(result.Media))
	}
	path, _, err := store.ResolveWithMeta(result.Media[0])
	if err != nil {
		t.Fatalf("expected stored media ref to resolve: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected stored media file to be readable: %v", err)
	}
	if string(data) != "blob-bytes" {
		t.Fatalf("expected stored blob bytes to match input, got %q", string(data))
	}
}

func TestMCPTool_Execute_RespectsUserAudienceForBinaryContent(t *testing.T) {
	store := media.NewFileMediaStore()
	manager := &MockMCPManager{
		callToolFunc: func(ctx context.Context, serverName, toolName string, arguments map[string]any) (*mcp.CallToolResult, error) {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.ImageContent{
						Data:        []byte("assistant-only"),
						MIMEType:    "image/png",
						Annotations: &mcp.Annotations{Audience: []mcp.Role{"assistant"}},
					},
				},
			}, nil
		},
	}

	mcpTool := NewMCPTool(manager, "screenshoto", &mcp.Tool{Name: "take_screenshot"})
	mcpTool.SetMediaStore(store)

	result := mcpTool.Execute(WithToolContext(context.Background(), "telegram", "chat-42"), nil)

	if len(result.Media) != 0 {
		t.Fatalf("expected no media ref for non-user audience, got %d", len(result.Media))
	}
	if !strings.Contains(result.ForLLM, "non-user audience") {
		t.Fatalf("expected audience note, got %q", result.ForLLM)
	}
}

func TestMCPTool_Execute_LargeBase64TextIsOmittedFromContext(t *testing.T) {
	manager := &MockMCPManager{
		callToolFunc: func(ctx context.Context, serverName, toolName string, arguments map[string]any) (*mcp.CallToolResult, error) {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: strings.Repeat("QUJD", 400)},
				},
			}, nil
		},
	}

	mcpTool := NewMCPTool(manager, "test_server", &mcp.Tool{Name: "dump_payload"})

	result := mcpTool.Execute(context.Background(), nil)

	if result.ForLLM != largeBase64OmittedMessage {
		t.Fatalf("expected sanitized large base64 note, got %q", result.ForLLM)
	}
}

func TestSummarizeResourceLink_Nil(t *testing.T) {
	got := summarizeResourceLink(nil)
	if got == "" {
		t.Error("summarizeResourceLink(nil) should return non-empty string")
	}
}

func TestSummarizeResourceLink_Empty(t *testing.T) {
	got := summarizeResourceLink(&mcp.ResourceLink{})
	if got == "" {
		t.Error("summarizeResourceLink empty should return non-empty string")
	}
}

func TestSummarizeResourceLink_WithFields(t *testing.T) {
	got := summarizeResourceLink(&mcp.ResourceLink{
		Name:     "my-resource",
		URI:      "file:///tmp/data.json",
		MIMEType: "application/json",
	})
	if !strings.Contains(got, "my-resource") {
		t.Errorf("summarizeResourceLink should contain name, got %q", got)
	}
	if !strings.Contains(got, "file:///tmp/data.json") {
		t.Errorf("summarizeResourceLink should contain uri, got %q", got)
	}
}

func TestSummarizeEmbeddedResource_NilContent(t *testing.T) {
	got := summarizeEmbeddedResource(nil)
	if got == "" {
		t.Error("summarizeEmbeddedResource(nil) should return non-empty string")
	}
}

func TestSummarizeEmbeddedResource_NilResource(t *testing.T) {
	got := summarizeEmbeddedResource(&mcp.EmbeddedResource{Resource: nil})
	if got == "" {
		t.Error("summarizeEmbeddedResource nil resource should return non-empty string")
	}
}

func TestSummarizeEmbeddedResource_WithURI(t *testing.T) {
	got := summarizeEmbeddedResource(&mcp.EmbeddedResource{
		Resource: &mcp.ResourceContents{URI: "file:///data.bin", MIMEType: "application/octet-stream"},
	})
	if !strings.Contains(got, "file:///data.bin") {
		t.Errorf("summarizeEmbeddedResource should contain URI, got %q", got)
	}
}

func TestSummarizeEmbeddedResource_NoURI(t *testing.T) {
	got := summarizeEmbeddedResource(&mcp.EmbeddedResource{
		Resource: &mcp.ResourceContents{MIMEType: "text/plain"},
	})
	if got == "" {
		t.Error("summarizeEmbeddedResource no URI should return non-empty string")
	}
}

func TestNormalizedMIMEType_Empty(t *testing.T) {
	if got := normalizedMIMEType(""); got != "application/octet-stream" {
		t.Errorf("normalizedMIMEType empty = %q, want application/octet-stream", got)
	}
}

func TestNormalizedMIMEType_WithValue(t *testing.T) {
	if got := normalizedMIMEType("text/html"); got != "text/html" {
		t.Errorf("normalizedMIMEType text/html = %q, want text/html", got)
	}
}

func TestSanitizeIdentifierComponent_Simple(t *testing.T) {
	got := sanitizeIdentifierComponent("MyServer")
	if got != "myserver" {
		t.Errorf("got %q, want 'myserver'", got)
	}
}

func TestSanitizeIdentifierComponent_SpecialChars(t *testing.T) {
	got := sanitizeIdentifierComponent("my server!")
	if got != "my_server" {
		t.Errorf("got %q, want 'my_server'", got)
	}
}

func TestSanitizeIdentifierComponent_ConsecutiveSpecial(t *testing.T) {
	got := sanitizeIdentifierComponent("a!!b")
	if got != "a_b" {
		t.Errorf("got %q, want 'a_b'", got)
	}
}

func TestSanitizeIdentifierComponent_AllSpecial(t *testing.T) {
	got := sanitizeIdentifierComponent("!!!")
	if got != "unnamed" {
		t.Errorf("got %q, want 'unnamed'", got)
	}
}

func TestSanitizeIdentifierComponent_Empty(t *testing.T) {
	got := sanitizeIdentifierComponent("")
	if got != "unnamed" {
		t.Errorf("got %q, want 'unnamed'", got)
	}
}

func TestSanitizeIdentifierComponent_LongString(t *testing.T) {
	long := ""
	for len(long) < 100 {
		long += "abcdef"
	}
	got := sanitizeIdentifierComponent(long)
	if len(got) > 64 {
		t.Errorf("len = %d, want <= 64", len(got))
	}
}

func TestSanitizeIdentifierComponent_AllowedChars(t *testing.T) {
	got := sanitizeIdentifierComponent("my-tool_v2")
	if got != "my-tool_v2" {
		t.Errorf("got %q, want 'my-tool_v2'", got)
	}
}

func TestSanitizeIdentifierComponent_LeadingTrailingSpecial(t *testing.T) {
	got := sanitizeIdentifierComponent("!hello!")
	if got != "hello" {
		t.Errorf("got %q, want 'hello'", got)
	}
}

func TestAnnotationsAllowUser_Nil(t *testing.T) {
	if !annotationsAllowUser(nil) {
		t.Error("nil annotations should allow user")
	}
}

func TestAnnotationsAllowUser_EmptyAudience(t *testing.T) {
	anns := &mcp.Annotations{Audience: []mcp.Role{}}
	if !annotationsAllowUser(anns) {
		t.Error("empty audience should allow user")
	}
}

func TestAnnotationsAllowUser_WithUser(t *testing.T) {
	anns := &mcp.Annotations{Audience: []mcp.Role{"user"}}
	if !annotationsAllowUser(anns) {
		t.Error("audience with 'user' should allow user")
	}
}

func TestAnnotationsAllowUser_WithUserUppercase(t *testing.T) {
	anns := &mcp.Annotations{Audience: []mcp.Role{"User"}}
	if !annotationsAllowUser(anns) {
		t.Error("audience with 'User' should allow user (case-insensitive)")
	}
}

func TestAnnotationsAllowUser_AssistantOnly(t *testing.T) {
	anns := &mcp.Annotations{Audience: []mcp.Role{"assistant"}}
	if annotationsAllowUser(anns) {
		t.Error("audience with 'assistant' only should NOT allow user")
	}
}

func TestSummarizeResourceLink_WithDescription(t *testing.T) {
	got := summarizeResourceLink(&mcp.ResourceLink{
		Name:        "my-resource",
		URI:         "mcp://server/resource",
		Description: "A useful resource",
	})
	if !strings.Contains(got, "description=") {
		t.Errorf("expected description= in output, got: %s", got)
	}
	if !strings.Contains(got, "A useful resource") {
		t.Errorf("expected description text in output, got: %s", got)
	}
}

func TestSummarizeResourceLink_LongDescription(t *testing.T) {
	desc := ""
	for len(desc) < 300 {
		desc += "abcdefghij"
	}
	got := summarizeResourceLink(&mcp.ResourceLink{Description: desc})
	if !strings.Contains(got, "...") {
		t.Errorf("long description should be truncated with '...', got: %s", got)
	}
}

func TestSummarizeResourceLink_WithMIMEType(t *testing.T) {
	got := summarizeResourceLink(&mcp.ResourceLink{
		URI:      "mcp://server/data",
		MIMEType: "application/json",
	})
	if !strings.Contains(got, "mime=") {
		t.Errorf("expected mime= in output, got: %s", got)
	}
}

// TestMCPTool_Name_LossyServerName checks that special chars in the server name
// trigger the hash-suffix path (sanitization is lossy).
func TestMCPTool_Name_LossyServerName(t *testing.T) {
	manager := &MockMCPManager{}
	tool := &mcp.Tool{Name: "get_data"}
	// Space and ! are not allowed → sanitize replaces them → lossless check fails
	mcpTool := NewMCPTool(manager, "my server!", tool)
	name := mcpTool.Name()
	// Name must contain an 8-char hex suffix separated by '_'
	parts := strings.Split(name, "_")
	suffix := parts[len(parts)-1]
	if len(suffix) != 8 {
		t.Errorf("expected 8-char hex suffix, got %q in name %q", suffix, name)
	}
}

// TestMCPTool_Name_LossyToolName checks that special chars in the tool name
// also trigger the hash-suffix path.
func TestMCPTool_Name_LossyToolName(t *testing.T) {
	manager := &MockMCPManager{}
	tool := &mcp.Tool{Name: "get data!"}
	mcpTool := NewMCPTool(manager, "myserver", tool)
	name := mcpTool.Name()
	parts := strings.Split(name, "_")
	suffix := parts[len(parts)-1]
	if len(suffix) != 8 {
		t.Errorf("expected 8-char hex suffix, got %q in name %q", suffix, name)
	}
}

// TestMCPTool_Name_TooLong checks that names exceeding 64 chars get truncated and hashed.
func TestMCPTool_Name_TooLong(t *testing.T) {
	manager := &MockMCPManager{}
	longTool := strings.Repeat("a", 60)
	tool := &mcp.Tool{Name: longTool}
	mcpTool := NewMCPTool(manager, "myserver", tool)
	name := mcpTool.Name()
	if len(name) > 64 {
		t.Errorf("name length %d exceeds 64: %q", len(name), name)
	}
	// Must still end with 8-char hex suffix (name was truncated → hash appended)
	parts := strings.Split(name, "_")
	suffix := parts[len(parts)-1]
	if len(suffix) != 8 {
		t.Errorf("expected 8-char hex suffix after truncation, got %q in name %q", suffix, name)
	}
}

// TestMCPTool_Parameters_JSONRawMessage covers the json.RawMessage type assertion path.
func TestMCPTool_Parameters_JSONRawMessage(t *testing.T) {
	manager := &MockMCPManager{}
	raw := json.RawMessage(`{"type":"object","properties":{"key":{"type":"string"}}}`)
	tool := &mcp.Tool{Name: "test_tool", InputSchema: raw}
	mcpTool := NewMCPTool(manager, "server", tool)
	params := mcpTool.Parameters()
	if params["type"] != "object" {
		t.Errorf("expected type=object, got %v", params["type"])
	}
	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties to be a map")
	}
	if _, ok := props["key"]; !ok {
		t.Error("expected key property to exist")
	}
}

// TestMCPTool_Parameters_InvalidJSONRawMessage covers the unmarshal-error fallback.
func TestMCPTool_Parameters_InvalidJSONRawMessage(t *testing.T) {
	manager := &MockMCPManager{}
	raw := json.RawMessage(`not valid json`)
	tool := &mcp.Tool{Name: "test_tool", InputSchema: raw}
	mcpTool := NewMCPTool(manager, "server", tool)
	params := mcpTool.Parameters()
	if params["type"] != "object" {
		t.Errorf("expected fallback type=object, got %v", params["type"])
	}
}

// TestMCPTool_Parameters_StructSchema covers the marshal/unmarshal path for struct types.
func TestMCPTool_Parameters_StructSchema(t *testing.T) {
	manager := &MockMCPManager{}
	schema := struct {
		Type       string `json:"type"`
		Properties map[string]any `json:"properties"`
	}{
		Type:       "object",
		Properties: map[string]any{"foo": map[string]any{"type": "string"}},
	}
	tool := &mcp.Tool{Name: "test_tool", InputSchema: schema}
	mcpTool := NewMCPTool(manager, "server", tool)
	params := mcpTool.Parameters()
	if params["type"] != "object" {
		t.Errorf("expected type=object from struct schema, got %v", params["type"])
	}
}

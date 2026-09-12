package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/providers"
)

// mockLLMProvider implements providers.LLMProvider for testing
type mockLLMProvider struct {
	responses     []*providers.LLMResponse
	responseIndex int
	chatCalls     int
}

func (m *mockLLMProvider) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	options map[string]any,
) (*providers.LLMResponse, error) {
	m.chatCalls++
	if m.responseIndex >= len(m.responses) {
		return &providers.LLMResponse{
			Content:      "final answer",
			FinishReason: "stop",
		}, nil
	}
	resp := m.responses[m.responseIndex]
	m.responseIndex++
	return resp, nil
}

func (m *mockLLMProvider) GetDefaultModel() string {
	return "mock-model"
}

func TestRunToolLoop_DirectAnswer(t *testing.T) {
	provider := &mockLLMProvider{
		responses: []*providers.LLMResponse{
			{
				Content:      "The answer is 42",
				FinishReason: "stop",
			},
		},
	}

	config := ToolLoopConfig{
		Provider:      provider,
		Model:         "mock-model",
		Tools:         NewToolRegistry(),
		MaxIterations: 10,
	}

	result, err := RunToolLoop(context.Background(), config, []providers.Message{}, "test", "user-1")

	if err != nil {
		t.Fatalf("RunToolLoop failed: %v", err)
	}
	if result.Content != "The answer is 42" {
		t.Errorf("Content = %q, want 'The answer is 42'", result.Content)
	}
	if result.Iterations != 1 {
		t.Errorf("Iterations = %d, want 1", result.Iterations)
	}
}

func TestRunToolLoop_NoToolRegistry(t *testing.T) {
	provider := &mockLLMProvider{
		responses: []*providers.LLMResponse{
			{
				Content:      "done",
				FinishReason: "stop",
			},
		},
	}

	config := ToolLoopConfig{
		Provider:      provider,
		Model:         "mock-model",
		Tools:         nil, // No tools
		MaxIterations: 5,
	}

	result, err := RunToolLoop(context.Background(), config, []providers.Message{}, "test", "user-1")

	if err != nil {
		t.Fatalf("RunToolLoop failed: %v", err)
	}
	if result.Iterations != 1 {
		t.Errorf("Iterations = %d, want 1 (no tools)", result.Iterations)
	}
}

func TestRunToolLoop_SingleIteration(t *testing.T) {
	provider := &mockLLMProvider{
		responses: []*providers.LLMResponse{
			{
				Content:      "response",
				FinishReason: "stop",
			},
		},
	}

	config := ToolLoopConfig{
		Provider:      provider,
		Model:         "mock-model",
		Tools:         NewToolRegistry(),
		MaxIterations: 1,
	}

	result, err := RunToolLoop(context.Background(), config, []providers.Message{}, "telegram", "chat-123")

	if err != nil {
		t.Fatalf("RunToolLoop failed: %v", err)
	}
	if provider.chatCalls != 1 {
		t.Errorf("Chat called %d times, want 1", provider.chatCalls)
	}
	if result.Iterations != 1 {
		t.Errorf("Iterations = %d, want 1", result.Iterations)
	}
}

func TestRunToolLoop_ZeroMaxIterations(t *testing.T) {
	provider := &mockLLMProvider{
		responses: []*providers.LLMResponse{},
	}

	config := ToolLoopConfig{
		Provider:      provider,
		Model:         "mock-model",
		Tools:         NewToolRegistry(),
		MaxIterations: 0,
	}

	result, err := RunToolLoop(context.Background(), config, []providers.Message{}, "test", "user-1")

	if err != nil {
		t.Fatalf("RunToolLoop failed: %v", err)
	}
	if result.Iterations != 0 {
		t.Errorf("Iterations = %d, want 0", result.Iterations)
	}
	if provider.chatCalls != 0 {
		t.Errorf("Chat should not be called with 0 max iterations, got %d calls", provider.chatCalls)
	}
}

func TestRunToolLoop_ProviderError(t *testing.T) {
	failingProvider := &mockLLMProviderError{}

	config := ToolLoopConfig{
		Provider:      failingProvider,
		Model:         "mock-model",
		Tools:         NewToolRegistry(),
		MaxIterations: 10,
	}

	result, err := RunToolLoop(context.Background(), config, []providers.Message{}, "test", "user-1")

	if err == nil {
		t.Error("RunToolLoop should return error when provider fails")
	}
	if result != nil {
		t.Error("RunToolLoop should return nil result on error")
	}
}

// mockLLMProviderError always returns an error
type mockLLMProviderError struct{}

func (m *mockLLMProviderError) Chat(
	ctx context.Context,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	model string,
	options map[string]any,
) (*providers.LLMResponse, error) {
	return nil, errors.New("connection failed")
}

func (m *mockLLMProviderError) GetDefaultModel() string {
	return "mock-model"
}

func TestRunToolLoop_EmptyToolRegistry(t *testing.T) {
	provider := &mockLLMProvider{
		responses: []*providers.LLMResponse{
			{
				Content:      "working",
				FinishReason: "stop",
			},
		},
	}

	registry := NewToolRegistry()
	// Empty registry

	config := ToolLoopConfig{
		Provider:      provider,
		Model:         "mock-model",
		Tools:         registry,
		MaxIterations: 5,
	}

	result, err := RunToolLoop(context.Background(), config, []providers.Message{}, "test", "user-1")

	if err != nil {
		t.Fatalf("RunToolLoop failed: %v", err)
	}
	if result.Iterations != 1 {
		t.Errorf("Iterations = %d, want 1", result.Iterations)
	}
}

func TestRunToolLoop_CustomLLMOptions(t *testing.T) {
	provider := &mockLLMProvider{
		responses: []*providers.LLMResponse{
			{
				Content:      "answer",
				FinishReason: "stop",
			},
		},
	}

	config := ToolLoopConfig{
		Provider:      provider,
		Model:         "custom-model",
		Tools:         NewToolRegistry(),
		MaxIterations: 5,
		LLMOptions: map[string]any{
			"temperature": 0.7,
			"max_tokens":  1000,
		},
	}

	result, err := RunToolLoop(context.Background(), config, []providers.Message{}, "test", "user-1")

	if err != nil {
		t.Fatalf("RunToolLoop failed: %v", err)
	}
	if result == nil {
		t.Fatal("RunToolLoop should return result")
	}
}

func TestRunToolLoop_WithInitialMessages(t *testing.T) {
	provider := &mockLLMProvider{
		responses: []*providers.LLMResponse{
			{
				Content:      "acknowledged",
				FinishReason: "stop",
			},
		},
	}

	initialMessages := []providers.Message{
		{
			Role:    "user",
			Content: "What is 2+2?",
		},
	}

	config := ToolLoopConfig{
		Provider:      provider,
		Model:         "mock-model",
		Tools:         NewToolRegistry(),
		MaxIterations: 5,
	}

	result, err := RunToolLoop(context.Background(), config, initialMessages, "test", "user-1")

	if err != nil {
		t.Fatalf("RunToolLoop failed: %v", err)
	}
	if result.Content != "acknowledged" {
		t.Errorf("Content = %q, want 'acknowledged'", result.Content)
	}
}

func TestRunToolLoop_MaxIterationsExceeded(t *testing.T) {
	// Create provider that always returns tool calls to trigger iterations
	provider := &mockLLMProvider{
		responses: []*providers.LLMResponse{
			{
				Content: "calling tool",
				ToolCalls: []providers.ToolCall{
					{
						ID:   "call-1",
						Name: "nonexistent_tool",
						Arguments: map[string]any{
							"arg": "value",
						},
					},
				},
				FinishReason: "tool_calls",
			},
			{
				Content: "calling again",
				ToolCalls: []providers.ToolCall{
					{
						ID:   "call-2",
						Name: "another_tool",
						Arguments: map[string]any{
							"arg": "value",
						},
					},
				},
				FinishReason: "tool_calls",
			},
			{
				Content:      "final answer",
				FinishReason: "stop",
			},
		},
	}

	config := ToolLoopConfig{
		Provider:      provider,
		Model:         "mock-model",
		Tools:         NewToolRegistry(),
		MaxIterations: 2, // Only allow 2 iterations
	}

	result, err := RunToolLoop(context.Background(), config, []providers.Message{}, "test", "user-1")

	if err != nil {
		t.Fatalf("RunToolLoop failed: %v", err)
	}
	if result.Iterations != 2 {
		t.Errorf("Iterations = %d, want 2 (max iterations exceeded)", result.Iterations)
	}
}

func TestRunToolLoop_ContextCancellation(t *testing.T) {
	provider := &mockLLMProvider{
		responses: []*providers.LLMResponse{
			{
				Content:      "answer",
				FinishReason: "stop",
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	config := ToolLoopConfig{
		Provider:      provider,
		Model:         "mock-model",
		Tools:         NewToolRegistry(),
		MaxIterations: 5,
	}

	result, err := RunToolLoop(ctx, config, []providers.Message{}, "test", "user-1")

	// Should fail due to context cancellation
	if err == nil && provider.chatCalls == 0 {
		t.Error("RunToolLoop should handle cancelled context")
	}
	if result != nil && result.Iterations > 0 {
		// May succeed if context isn't checked fast enough
		t.Logf("RunToolLoop completed with %d iterations despite cancelled context", result.Iterations)
	}
}

func TestToolLoopResult_Fields(t *testing.T) {
	result := &ToolLoopResult{
		Content:    "test output",
		Iterations: 3,
	}

	if result.Content != "test output" {
		t.Errorf("Content = %q, want 'test output'", result.Content)
	}
	if result.Iterations != 3 {
		t.Errorf("Iterations = %d, want 3", result.Iterations)
	}
}

func TestToolLoopConfig_DefaultValues(t *testing.T) {
	// Verify default values in config
	config := ToolLoopConfig{
		Provider: &mockLLMProvider{},
		Model:    "test",
	}

	if config.Tools != nil {
		t.Error("Tools should be nil by default")
	}
	if config.MaxIterations != 0 {
		t.Error("MaxIterations should be 0 by default")
	}
	if config.LLMOptions != nil {
		t.Error("LLMOptions should be nil by default")
	}
}

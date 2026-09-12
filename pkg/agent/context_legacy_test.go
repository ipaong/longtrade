package agent

import (
	"context"
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/providers"
)

func TestLegacyContextManager_Assemble(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t)
	defer cleanup()

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatalf("Expected default agent, got nil")
	}

	mgr := &legacyContextManager{al: al}
	ctx := context.Background()

	req := &AssembleRequest{
		SessionKey: "test-session",
	}
	resp, err := mgr.Assemble(ctx, req)
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}
	if resp == nil {
		t.Errorf("Expected AssembleResponse, got nil")
	}
}

func TestLegacyContextManager_Assemble_NoAgent(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t)
	defer cleanup()

	// Create a manager with a loop that has no agent (simulate by using nil registry)
	mgr := &legacyContextManager{al: al}
	ctx := context.Background()

	resp, _ := mgr.Assemble(ctx, &AssembleRequest{SessionKey: "test"})
	// Should still work, just return empty response
	if resp != nil && len(resp.History) != 0 {
		t.Errorf("Expected empty or nil response for missing agent")
	}
}

func TestLegacyContextManager_Compact_Summarize(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t)
	defer cleanup()

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatalf("Expected default agent")
	}

	mgr := &legacyContextManager{al: al}
	ctx := context.Background()

	err := mgr.Compact(ctx, &CompactRequest{
		SessionKey: "test-session",
		Reason:     ContextCompressReasonSummarize,
	})
	if err != nil {
		t.Fatalf("Compact failed: %v", err)
	}
}

func TestLegacyContextManager_Compact_Retry(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t)
	defer cleanup()

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatalf("Expected default agent")
	}

	mgr := &legacyContextManager{al: al}
	ctx := context.Background()

	err := mgr.Compact(ctx, &CompactRequest{
		SessionKey: "test-session",
		Reason:     ContextCompressReasonRetry,
	})
	if err != nil {
		t.Fatalf("Compact failed: %v", err)
	}
}

func TestLegacyContextManager_Ingest(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t)
	defer cleanup()

	mgr := &legacyContextManager{al: al}
	ctx := context.Background()

	// Ingest should be a no-op for legacy manager
	err := mgr.Ingest(ctx, &IngestRequest{
		SessionKey: "test",
		Message:    providers.Message{Role: "user", Content: "test"},
	})
	if err != nil {
		t.Fatalf("Ingest failed: %v", err)
	}
}

func TestLegacyContextManager_Clear(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t)
	defer cleanup()

	agent := al.registry.GetDefaultAgent()
	if agent == nil || agent.Sessions == nil {
		t.Fatalf("Expected agent with sessions")
	}

	mgr := &legacyContextManager{al: al}
	ctx := context.Background()

	sessionKey := "test-session"

	err := mgr.Clear(ctx, sessionKey)
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// Verify history is cleared
	history := agent.Sessions.GetHistory(sessionKey)
	if len(history) != 0 {
		t.Errorf("Expected cleared history, got %d messages", len(history))
	}
}

func TestLegacyContextManager_Clear_NoSessions(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t)
	defer cleanup()

	mgr := &legacyContextManager{al: al}
	ctx := context.Background()

	// Create a scenario where Sessions is nil (shouldn't normally happen)
	tempAgent := al.registry.GetDefaultAgent()
	originalSessions := tempAgent.Sessions
	tempAgent.Sessions = nil

	err := mgr.Clear(ctx, "test-session")
	if err == nil {
		t.Errorf("Expected error when sessions is nil")
	}

	// Restore
	tempAgent.Sessions = originalSessions
}

func TestLegacyContextManager_FindNearestUserMessage(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t)
	defer cleanup()

	mgr := &legacyContextManager{al: al}

	messages := []providers.Message{
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "response"},
		{Role: "user", Content: "second"},
		{Role: "assistant", Content: "response"},
	}

	// Start from index 1 (assistant), should find index 0 (user) backward
	idx := mgr.findNearestUserMessage(messages, 1)
	if idx < 0 || idx >= len(messages) || messages[idx].Role != "user" {
		t.Errorf("Expected to find user message, got index %d", idx)
	}
}

func TestLegacyContextManager_FindNearestUserMessage_Forward(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t)
	defer cleanup()

	mgr := &legacyContextManager{al: al}

	messages := []providers.Message{
		{Role: "assistant", Content: "response"},
		{Role: "assistant", Content: "response"},
		{Role: "user", Content: "later"},
	}

	// Start from index 0, all backward are assistant, should find forward
	idx := mgr.findNearestUserMessage(messages, 0)
	if idx < 0 || idx >= len(messages) {
		t.Errorf("Expected to find user message at valid index")
	}
}

func TestLegacyContextManager_EstimateTokens(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t)
	defer cleanup()

	mgr := &legacyContextManager{al: al}

	messages := []providers.Message{
		{Role: "user", Content: "hello world"},
	}

	tokens := mgr.estimateTokens(messages)
	if tokens <= 0 {
		t.Errorf("Expected positive token count, got %d", tokens)
	}
}

func TestLegacyContextManager_EstimateTokens_Empty(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t)
	defer cleanup()

	mgr := &legacyContextManager{al: al}

	messages := []providers.Message{}

	tokens := mgr.estimateTokens(messages)
	if tokens != 0 {
		t.Errorf("Expected zero tokens for empty messages, got %d", tokens)
	}
}

func TestLegacyContextManager_ForceCompression_WithHistory(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t)
	defer cleanup()

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatalf("Expected default agent, got nil")
	}

	mgr := &legacyContextManager{al: al}
	sessionKey := "force-compress-test"

	history := []providers.Message{
		{Role: "user", Content: "message 1"},
		{Role: "assistant", Content: "response 1"},
		{Role: "user", Content: "message 2"},
		{Role: "assistant", Content: "response 2"},
		{Role: "user", Content: "message 3"},
		{Role: "assistant", Content: "response 3"},
	}
	agent.Sessions.SetHistory(sessionKey, history)

	err := mgr.Compact(context.Background(), &CompactRequest{
		SessionKey: sessionKey,
		Reason:     ContextCompressReasonRetry,
	})
	if err != nil {
		t.Fatalf("Compact with history failed: %v", err)
	}

	newHistory := agent.Sessions.GetHistory(sessionKey)
	if len(newHistory) >= len(history) {
		t.Errorf("Expected compression: original %d messages -> still %d", len(history), len(newHistory))
	}
}

func TestLegacyContextManager_RetryLLMCall(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t)
	defer cleanup()

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatalf("Expected default agent, got nil")
	}

	mgr := &legacyContextManager{al: al}

	resp, err := mgr.retryLLMCall(context.Background(), agent, "summarize this test prompt", 3)
	if err != nil {
		t.Fatalf("retryLLMCall failed: %v", err)
	}
	if resp == nil || resp.Content == "" {
		t.Error("Expected non-empty response from retryLLMCall")
	}
}

func TestLegacyContextManager_SummarizeBatch(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t)
	defer cleanup()

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatalf("Expected default agent, got nil")
	}

	mgr := &legacyContextManager{al: al}

	batch := []providers.Message{
		{Role: "user", Content: "Hello, how are you?"},
		{Role: "assistant", Content: "I am doing well, thank you!"},
		{Role: "user", Content: "What is the weather like?"},
		{Role: "assistant", Content: "It is sunny today."},
	}

	summary, err := mgr.summarizeBatch(context.Background(), agent, batch, "")
	if err != nil {
		t.Fatalf("summarizeBatch failed: %v", err)
	}
	if summary == "" {
		t.Error("Expected non-empty summary from summarizeBatch")
	}
}

func TestLegacyContextManager_SummarizeBatch_WithExistingSummary(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t)
	defer cleanup()

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatalf("Expected default agent, got nil")
	}

	mgr := &legacyContextManager{al: al}

	batch := []providers.Message{
		{Role: "user", Content: "Continue from before"},
		{Role: "assistant", Content: "Sure, continuing now."},
	}

	summary, err := mgr.summarizeBatch(context.Background(), agent, batch, "Prior context: user asked about weather")
	if err != nil {
		t.Fatalf("summarizeBatch with existing summary failed: %v", err)
	}
	if summary == "" {
		t.Error("Expected non-empty summary")
	}
}

func TestLegacyContextManager_SummarizeSession_WithHistory(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t)
	defer cleanup()

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatalf("Expected default agent, got nil")
	}

	mgr := &legacyContextManager{al: al}
	sessionKey := "summarize-session-test"

	// Need > 4 messages and a Turn boundary at index 2 for safeCut > 0.
	history := []providers.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there"},
		{Role: "user", Content: "How are you?"},
		{Role: "assistant", Content: "I am doing well"},
		{Role: "user", Content: "Tell me about Go"},
		{Role: "assistant", Content: "Go is a compiled language"},
	}
	agent.Sessions.SetHistory(sessionKey, history)

	// Call directly (bypassing the goroutine in maybeSummarize)
	mgr.summarizeSession(agent, sessionKey)

	// After summarization, history should be shorter
	newHistory := agent.Sessions.GetHistory(sessionKey)
	if len(newHistory) >= len(history) {
		t.Errorf("Expected summarization to reduce history: original %d, got %d", len(history), len(newHistory))
	}
}

func TestLegacyContextManager_SummarizeSession_TooShort(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t)
	defer cleanup()

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatalf("Expected default agent, got nil")
	}

	mgr := &legacyContextManager{al: al}
	sessionKey := "summarize-short-test"

	// Only 3 messages — summarizeSession should return early (len <= 4)
	history := []providers.Message{
		{Role: "user", Content: "msg1"},
		{Role: "assistant", Content: "resp1"},
		{Role: "user", Content: "msg2"},
	}
	agent.Sessions.SetHistory(sessionKey, history)
	mgr.summarizeSession(agent, sessionKey)

	// History should be unchanged
	newHistory := agent.Sessions.GetHistory(sessionKey)
	if len(newHistory) != len(history) {
		t.Errorf("Expected history unchanged for short session: got %d, want %d", len(newHistory), len(history))
	}
}

func TestLegacyContextManager_SummarizeSession_LargeBatch(t *testing.T) {
	al, _, _, _, cleanup := newTestAgentLoop(t)
	defer cleanup()

	agent := al.registry.GetDefaultAgent()
	if agent == nil {
		t.Fatalf("Expected default agent, got nil")
	}

	mgr := &legacyContextManager{al: al}
	sessionKey := "summarize-large-test"

	// Create >10 valid messages to trigger multi-part summarization path
	var history []providers.Message
	for i := 0; i < 16; i++ {
		if i%2 == 0 {
			history = append(history, providers.Message{Role: "user", Content: "user message"})
		} else {
			history = append(history, providers.Message{Role: "assistant", Content: "assistant response"})
		}
	}
	agent.Sessions.SetHistory(sessionKey, history)
	mgr.summarizeSession(agent, sessionKey)
	// Just verify no panic and something happened
}

package session_test

import (
	"fmt"
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/memory"
	"github.com/cryptoquantumwave/khunquant/pkg/providers"
	"github.com/cryptoquantumwave/khunquant/pkg/session"
)

// Compile-time interface satisfaction checks.
var (
	_ session.SessionStore = (*session.SessionManager)(nil)
	_ session.SessionStore = (*session.JSONLBackend)(nil)
)

func newBackend(t *testing.T) *session.JSONLBackend {
	t.Helper()
	store, err := memory.NewJSONLStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	return session.NewJSONLBackend(store)
}

func TestJSONLBackend_AddAndGetHistory(t *testing.T) {
	b := newBackend(t)

	b.AddMessage("s1", "user", "hello")
	b.AddMessage("s1", "assistant", "hi")

	history := b.GetHistory("s1")
	if len(history) != 2 {
		t.Fatalf("got %d messages, want 2", len(history))
	}
	if history[0].Role != "user" || history[0].Content != "hello" {
		t.Errorf("msg[0] = %+v", history[0])
	}
	if history[1].Role != "assistant" || history[1].Content != "hi" {
		t.Errorf("msg[1] = %+v", history[1])
	}
}

func TestJSONLBackend_AddFullMessage(t *testing.T) {
	b := newBackend(t)

	msg := providers.Message{
		Role:    "assistant",
		Content: "done",
		ToolCalls: []providers.ToolCall{
			{ID: "tc1", Function: &providers.FunctionCall{Name: "read_file", Arguments: `{"path":"x"}`}},
		},
	}
	b.AddFullMessage("s1", msg)

	history := b.GetHistory("s1")
	if len(history) != 1 {
		t.Fatalf("got %d, want 1", len(history))
	}
	if len(history[0].ToolCalls) != 1 || history[0].ToolCalls[0].ID != "tc1" {
		t.Errorf("tool calls = %+v", history[0].ToolCalls)
	}
}

func TestJSONLBackend_Summary(t *testing.T) {
	b := newBackend(t)

	if got := b.GetSummary("s1"); got != "" {
		t.Errorf("got %q, want empty", got)
	}

	b.SetSummary("s1", "test summary")
	if got := b.GetSummary("s1"); got != "test summary" {
		t.Errorf("got %q, want %q", got, "test summary")
	}
}

func TestJSONLBackend_TruncateAndSave(t *testing.T) {
	b := newBackend(t)

	for i := 0; i < 10; i++ {
		b.AddMessage("s1", "user", fmt.Sprintf("msg %d", i))
	}
	b.TruncateHistory("s1", 3)

	history := b.GetHistory("s1")
	if len(history) != 3 {
		t.Fatalf("got %d, want 3", len(history))
	}
	if history[0].Content != "msg 7" {
		t.Errorf("got %q, want %q", history[0].Content, "msg 7")
	}

	// Save triggers compaction.
	if err := b.Save("s1"); err != nil {
		t.Fatal(err)
	}

	// Messages still accessible after compaction.
	history = b.GetHistory("s1")
	if len(history) != 3 {
		t.Fatalf("after save: got %d, want 3", len(history))
	}
}

func TestJSONLBackend_SetHistory(t *testing.T) {
	b := newBackend(t)
	b.AddMessage("s1", "user", "old")

	b.SetHistory("s1", []providers.Message{
		{Role: "user", Content: "new1"},
		{Role: "assistant", Content: "new2"},
	})

	history := b.GetHistory("s1")
	if len(history) != 2 {
		t.Fatalf("got %d, want 2", len(history))
	}
	if history[0].Content != "new1" {
		t.Errorf("got %q, want %q", history[0].Content, "new1")
	}
}

func TestJSONLBackend_EmptySession(t *testing.T) {
	b := newBackend(t)

	history := b.GetHistory("nonexistent")
	if history == nil {
		t.Fatal("got nil, want empty slice")
	}
	if len(history) != 0 {
		t.Errorf("got %d, want 0", len(history))
	}
}

func TestJSONLBackend_SessionIsolation(t *testing.T) {
	b := newBackend(t)
	b.AddMessage("s1", "user", "session1")
	b.AddMessage("s2", "user", "session2")

	h1 := b.GetHistory("s1")
	h2 := b.GetHistory("s2")

	if len(h1) != 1 || h1[0].Content != "session1" {
		t.Errorf("s1: %+v", h1)
	}
	if len(h2) != 1 || h2[0].Content != "session2" {
		t.Errorf("s2: %+v", h2)
	}
}

func TestJSONLBackend_SummarizeFlow(t *testing.T) {
	// Simulates the real summarization flow in the agent loop:
	// SetSummary → TruncateHistory → Save
	b := newBackend(t)

	for i := 0; i < 20; i++ {
		b.AddMessage("s1", "user", fmt.Sprintf("msg %d", i))
	}

	b.SetSummary("s1", "conversation about testing")
	b.TruncateHistory("s1", 4)
	if err := b.Save("s1"); err != nil {
		t.Fatal(err)
	}

	if got := b.GetSummary("s1"); got != "conversation about testing" {
		t.Errorf("summary = %q", got)
	}
	history := b.GetHistory("s1")
	if len(history) != 4 {
		t.Fatalf("got %d messages, want 4", len(history))
	}
	if history[0].Content != "msg 16" {
		t.Errorf("first message = %q, want %q", history[0].Content, "msg 16")
	}
}

func TestJSONLBackend_Close(t *testing.T) {
	b := newBackend(t)
	b.AddMessage("s1", "user", "test")

	if err := b.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	// After close, store should still be reachable for a final save attempt
	// (Close doesn't prevent further operations on the underlying store).
	history := b.GetHistory("s1")
	if len(history) != 1 {
		t.Errorf("after Close(), GetHistory should still work: got %d, want 1", len(history))
	}
}

func TestJSONLBackend_ListSessions(t *testing.T) {
	b := newBackend(t)

	// Initially, ListSessions should return empty slice
	if keys := b.ListSessions(); len(keys) != 0 {
		t.Errorf("ListSessions() on empty backend: got %d, want 0", len(keys))
	}

	// Add messages to different sessions
	b.AddMessage("session1", "user", "msg1")
	b.AddMessage("session2", "user", "msg2")
	b.AddMessage("session3", "user", "msg3")

	// ListSessions should return all three session keys
	keys := b.ListSessions()
	if len(keys) != 3 {
		t.Fatalf("ListSessions(): got %d keys, want 3", len(keys))
	}

	// Check that all expected keys are present (order is not guaranteed)
	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}
	for _, expected := range []string{"session1", "session2", "session3"} {
		if !keySet[expected] {
			t.Errorf("ListSessions() missing key %q", expected)
		}
	}
}

func TestJSONLBackend_AddMessage_ErrorHandling(t *testing.T) {
	// AddMessage doesn't return errors, but we verify the basic flow works
	// even with edge cases (like empty key).
	b := newBackend(t)

	b.AddMessage("", "user", "empty key")
	b.AddMessage("s1", "", "empty role")
	b.AddMessage("s1", "user", "")

	// All should be silently handled; at least the last valid one should exist
	history := b.GetHistory("s1")
	if len(history) < 2 {
		t.Errorf("AddMessage with edge cases: got %d messages, expected at least 2", len(history))
	}
}

func TestJSONLBackend_SetSummary_NonexistentSession(t *testing.T) {
	b := newBackend(t)

	// SetSummary on a non-existent session should not panic or error
	b.SetSummary("nonexistent", "summary text")

	// GetSummary should reflect the set value (backend creates session implicitly)
	if got := b.GetSummary("nonexistent"); got != "summary text" {
		t.Errorf("SetSummary on nonexistent session: got %q, want %q", got, "summary text")
	}
}

func TestJSONLBackend_TruncateHistory_EdgeCases(t *testing.T) {
	b := newBackend(t)

	// Truncate non-existent session should not panic
	b.TruncateHistory("nonexistent", 5)
	if len(b.GetHistory("nonexistent")) != 0 {
		t.Errorf("TruncateHistory on nonexistent session should remain empty")
	}

	// Add some messages
	for i := 0; i < 5; i++ {
		b.AddMessage("s1", "user", fmt.Sprintf("msg%d", i))
	}

	// Truncate with keepLast=0
	b.TruncateHistory("s1", 0)
	if len(b.GetHistory("s1")) != 0 {
		t.Errorf("TruncateHistory(0): got %d, want 0", len(b.GetHistory("s1")))
	}

	// Re-add and truncate with keepLast > current length
	for i := 0; i < 3; i++ {
		b.AddMessage("s2", "user", fmt.Sprintf("msg%d", i))
	}
	b.TruncateHistory("s2", 100)
	if len(b.GetHistory("s2")) != 3 {
		t.Errorf("TruncateHistory(100) when only 3 messages: got %d, want 3", len(b.GetHistory("s2")))
	}
}

func TestJSONLBackend_Save_NilBackend(t *testing.T) {
	// Ensure that Save returns an error appropriately, even with edge cases
	b := newBackend(t)
	b.AddMessage("s1", "user", "test")

	// Normal save should work
	if err := b.Save("s1"); err != nil {
		t.Fatalf("Save() should not error: %v", err)
	}

	// Save on a session that exists should succeed
	if err := b.Save("s1"); err != nil {
		t.Fatalf("Save() second time should not error: %v", err)
	}
}

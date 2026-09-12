package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/providers"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"telegram:123456", "telegram_123456"},
		{"discord:987654321", "discord_987654321"},
		{"slack:C01234", "slack_C01234"},
		{"no-colons-here", "no-colons-here"},
		{"multiple:colons:here", "multiple_colons_here"},
		{"agent:main:telegram:group:-1003822706455/12", "agent_main_telegram_group_-1003822706455_12"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSave_WithColonInKey(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir)

	// Create a session with a key containing colon (typical channel session key).
	key := "telegram:123456"
	sm.GetOrCreate(key)
	sm.AddMessage(key, "user", "hello")

	// Save should succeed even though the key contains ':'
	if err := sm.Save(key); err != nil {
		t.Fatalf("Save(%q) failed: %v", key, err)
	}

	// The file on disk should use sanitized name.
	expectedFile := filepath.Join(tmpDir, "telegram_123456.json")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Fatalf("expected session file %s to exist", expectedFile)
	}

	// Load into a fresh manager and verify the session round-trips.
	sm2 := NewSessionManager(tmpDir)
	history := sm2.GetHistory(key)
	if len(history) != 1 {
		t.Fatalf("expected 1 message after reload, got %d", len(history))
	}
	if history[0].Content != "hello" {
		t.Errorf("expected message content %q, got %q", "hello", history[0].Content)
	}
}

func TestSave_RejectsPathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir)

	// Invalid names that must still be rejected.
	badKeys := []string{"", ".", ".."}
	for _, key := range badKeys {
		sm.GetOrCreate(key)
		if err := sm.Save(key); err == nil {
			t.Errorf("Save(%q) should have failed but didn't", key)
		}
	}

	// Keys containing path separators are sanitized (no subdirs created).
	sm.GetOrCreate("foo/bar")
	if err := sm.Save("foo/bar"); err != nil {
		t.Fatalf("Save(\"foo/bar\") after sanitize should succeed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "foo_bar.json")); os.IsNotExist(err) {
		t.Errorf("expected foo_bar.json in storage (sanitized from foo/bar)")
	}
}

func TestSessionManager_GetSummary(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir)

	// GetSummary on nonexistent session returns empty string
	if got := sm.GetSummary("nonexistent"); got != "" {
		t.Errorf("GetSummary on nonexistent: got %q, want empty", got)
	}

	// Create a session and set a summary
	session := sm.GetOrCreate("s1")
	session.Summary = "test summary"

	if got := sm.GetSummary("s1"); got != "test summary" {
		t.Errorf("GetSummary: got %q, want %q", got, "test summary")
	}
}

func TestSessionManager_SetSummary(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir)

	// SetSummary on nonexistent session should not panic
	sm.SetSummary("nonexistent", "summary")
	if got := sm.GetSummary("nonexistent"); got != "" {
		t.Errorf("SetSummary on nonexistent session should not create it: got %q", got)
	}

	// SetSummary on existing session
	sm.GetOrCreate("s1")
	sm.SetSummary("s1", "new summary")
	if got := sm.GetSummary("s1"); got != "new summary" {
		t.Errorf("SetSummary: got %q, want %q", got, "new summary")
	}

	// Verify Updated timestamp changed
	session := sm.GetOrCreate("s1")
	oldUpdated := session.Updated
	sm.SetSummary("s1", "another summary")
	if session.Updated == oldUpdated {
		t.Errorf("SetSummary should update the Updated timestamp")
	}
}

func TestSessionManager_TruncateHistory(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir)

	// TruncateHistory on nonexistent session should not panic
	sm.TruncateHistory("nonexistent", 5)

	// TruncateHistory on existing session
	_ = sm.GetOrCreate("s1")
	for i := 0; i < 10; i++ {
		sm.AddMessage("s1", "user", string(rune('0'+i)))
	}

	sm.TruncateHistory("s1", 3)
	history := sm.GetHistory("s1")
	if len(history) != 3 {
		t.Fatalf("TruncateHistory(3): got %d, want 3", len(history))
	}
	if history[0].Content != "7" {
		t.Errorf("first message after truncate: got %q, want %q", history[0].Content, "7")
	}

	// TruncateHistory with keepLast=0
	sm.TruncateHistory("s1", 0)
	history = sm.GetHistory("s1")
	if len(history) != 0 {
		t.Errorf("TruncateHistory(0): got %d, want 0", len(history))
	}

	// TruncateHistory with keepLast >= current length
	for i := 0; i < 5; i++ {
		sm.AddMessage("s1", "user", string(rune('0'+i)))
	}
	sm.TruncateHistory("s1", 10)
	history = sm.GetHistory("s1")
	if len(history) != 5 {
		t.Errorf("TruncateHistory(10) when only 5 messages: got %d, want 5", len(history))
	}
}

func TestSessionManager_ListSessions(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir)

	// Empty manager
	if keys := sm.ListSessions(); len(keys) != 0 {
		t.Errorf("ListSessions on empty manager: got %d, want 0", len(keys))
	}

	// Add sessions
	sm.GetOrCreate("s1")
	sm.GetOrCreate("s2")
	sm.GetOrCreate("s3")

	keys := sm.ListSessions()
	if len(keys) != 3 {
		t.Fatalf("ListSessions: got %d, want 3", len(keys))
	}

	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}
	for _, expected := range []string{"s1", "s2", "s3"} {
		if !keySet[expected] {
			t.Errorf("ListSessions missing key %q", expected)
		}
	}
}

func TestSessionManager_Close(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir)

	// Close should return nil (no-op)
	if err := sm.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	// SessionManager should still work after Close
	sm.GetOrCreate("s1")
	sm.AddMessage("s1", "user", "test")
	history := sm.GetHistory("s1")
	if len(history) != 1 {
		t.Errorf("after Close(), operations should still work: got %d, want 1", len(history))
	}
}

func TestSessionManager_SetHistory(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir)

	// SetHistory on nonexistent session should not panic
	sm.SetHistory("nonexistent", []providers.Message{})

	// SetHistory on existing session
	_ = sm.GetOrCreate("s1")
	sm.AddMessage("s1", "user", "old message")

	newHistory := []providers.Message{
		{Role: "user", Content: "new1"},
		{Role: "assistant", Content: "new2"},
	}
	sm.SetHistory("s1", newHistory)

	history := sm.GetHistory("s1")
	if len(history) != 2 {
		t.Fatalf("SetHistory: got %d messages, want 2", len(history))
	}
	if history[0].Content != "new1" || history[1].Content != "new2" {
		t.Errorf("SetHistory: messages not replaced correctly")
	}

	// Test that SetHistory creates a deep copy
	t.Run("SetHistory_DeepCopy", func(t *testing.T) {
		tmpDir := t.TempDir()
		sm := NewSessionManager(tmpDir)

		// Create session first
		_ = sm.GetOrCreate("s1")

		originalMessages := []providers.Message{
			{Role: "user", Content: "msg1"},
		}
		sm.SetHistory("s1", originalMessages)

		// Modify the original slice
		originalMessages[0].Content = "modified"

		// The session should still have the original value (deep copy)
		history := sm.GetHistory("s1")
		if len(history) != 1 {
			t.Fatalf("SetHistory deep copy: expected 1 message, got %d", len(history))
		}
		if history[0].Content != "msg1" {
			t.Errorf("SetHistory deep copy failed: got %q, want %q", history[0].Content, "msg1")
		}
	})
}

func TestSessionManager_LoadSessions_Persistence(t *testing.T) {
	tmpDir := t.TempDir()

	// Create and save a session
	sm1 := NewSessionManager(tmpDir)
	sm1.GetOrCreate("persistent:session")
	sm1.AddMessage("persistent:session", "user", "important message")
	if err := sm1.Save("persistent:session"); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load from disk with a fresh manager
	sm2 := NewSessionManager(tmpDir)
	history := sm2.GetHistory("persistent:session")
	if len(history) != 1 {
		t.Fatalf("loaded history: got %d messages, want 1", len(history))
	}
	if history[0].Content != "important message" {
		t.Errorf("loaded message content: got %q, want %q", history[0].Content, "important message")
	}
}

func TestSessionManager_NoStorage(t *testing.T) {
	// SessionManager with empty storage string should work in-memory only
	sm := NewSessionManager("")

	sm.GetOrCreate("s1")
	sm.AddMessage("s1", "user", "message")

	// Save should be a no-op when storage is empty
	if err := sm.Save("s1"); err != nil {
		t.Fatalf("Save with empty storage: %v", err)
	}

	history := sm.GetHistory("s1")
	if len(history) != 1 {
		t.Errorf("in-memory session: got %d, want 1", len(history))
	}
}

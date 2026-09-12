package agent

import (
	"path/filepath"
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/bus"
	"github.com/cryptoquantumwave/khunquant/pkg/providers"
)

func TestComputeContextUsage_NilAgent(t *testing.T) {
	result := computeContextUsage(nil, "test")
	if result != nil {
		t.Errorf("Expected nil for nil agent, got %v", result)
	}
}

func TestComputeContextUsage_NoSessions(t *testing.T) {
	agent := &AgentInstance{
		ID:            "test",
		ContextWindow: 4096,
	}
	result := computeContextUsage(agent, "test")
	if result != nil {
		t.Errorf("Expected nil for nil sessions, got %v", result)
	}
}

func TestComputeContextUsage_InvalidContextWindow(t *testing.T) {
	agent := &AgentInstance{
		ID:            "test",
		ContextWindow: 0,
		Sessions:      initSessionStore(t.TempDir()),
	}
	result := computeContextUsage(agent, "test")
	if result != nil {
		t.Errorf("Expected nil for invalid context window, got %v", result)
	}
}

func TestComputeContextUsage_EmptySession(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions")
	sessions := initSessionStore(sessionsDir)
	agent := &AgentInstance{
		ID:            "test",
		ContextWindow: 4096,
		MaxTokens:     1024,
		Sessions:      sessions,
	}

	result := computeContextUsage(agent, "test-session")
	if result == nil {
		t.Fatalf("Expected ContextUsage, got nil")
	}

	if result.TotalTokens != 4096 {
		t.Errorf("Expected TotalTokens=4096, got %d", result.TotalTokens)
	}
	if result.CompressAtTokens != 3072 { // 4096 - 1024
		t.Errorf("Expected CompressAtTokens=3072, got %d", result.CompressAtTokens)
	}
}

func TestComputeContextUsage_WithHistory(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions")
	sessions := initSessionStore(sessionsDir)

	sessionKey := "test-session"
	history := []providers.Message{
		{Role: "user", Content: "hello world"},
		{Role: "assistant", Content: "hi there"},
	}
	sessions.SetHistory(sessionKey, history)

	agent := &AgentInstance{
		ID:            "test",
		ContextWindow: 4096,
		MaxTokens:     1024,
		Sessions:      sessions,
	}

	result := computeContextUsage(agent, sessionKey)
	if result == nil {
		t.Fatalf("Expected ContextUsage, got nil")
	}

	if result.UsedTokens <= 0 {
		t.Errorf("Expected positive UsedTokens, got %d", result.UsedTokens)
	}
	if result.UsedPercent < 0 || result.UsedPercent > 100 {
		t.Errorf("Expected UsedPercent 0-100, got %d", result.UsedPercent)
	}
}

func TestComputeContextUsage_WithContextBuilder(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions")
	sessions := initSessionStore(sessionsDir)
	contextBuilder := NewContextBuilder(tmpDir)

	sessionKey := "test-session"
	history := []providers.Message{
		{Role: "user", Content: "test message"},
	}
	sessions.SetHistory(sessionKey, history)
	sessions.SetSummary(sessionKey, "Test summary")

	agent := &AgentInstance{
		ID:             "test",
		ContextWindow:  4096,
		MaxTokens:      1024,
		Sessions:       sessions,
		ContextBuilder: contextBuilder,
	}

	result := computeContextUsage(agent, sessionKey)
	if result == nil {
		t.Fatalf("Expected ContextUsage, got nil")
	}

	if result.TotalTokens != 4096 {
		t.Errorf("Expected TotalTokens=4096, got %d", result.TotalTokens)
	}
	if result.UsedTokens <= 0 {
		t.Errorf("Expected positive UsedTokens, got %d", result.UsedTokens)
	}
}

func TestComputeContextUsage_EffectiveWindow(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions")
	sessions := initSessionStore(sessionsDir)

	agent := &AgentInstance{
		ID:            "test",
		ContextWindow: 2000,
		MaxTokens:     500,
		Sessions:      sessions,
	}

	result := computeContextUsage(agent, "test")
	if result == nil {
		t.Fatalf("Expected ContextUsage, got nil")
	}

	expectedEffective := 1500 // 2000 - 500
	if result.CompressAtTokens != expectedEffective {
		t.Errorf("Expected CompressAtTokens=%d, got %d", expectedEffective, result.CompressAtTokens)
	}
}

func TestComputeContextUsage_MaxTokensGreaterThanWindow(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions")
	sessions := initSessionStore(sessionsDir)

	agent := &AgentInstance{
		ID:            "test",
		ContextWindow: 1000,
		MaxTokens:     2000,
		Sessions:      sessions,
	}

	result := computeContextUsage(agent, "test")
	if result == nil {
		t.Fatalf("Expected ContextUsage, got nil")
	}

	// Should use full context window as fallback
	if result.CompressAtTokens != 1000 {
		t.Errorf("Expected fallback CompressAtTokens=1000, got %d", result.CompressAtTokens)
	}
}

func TestComputeContextUsage_UsedPercentCalculation(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions")
	sessions := initSessionStore(sessionsDir)

	// Add history that uses some tokens
	sessionKey := "test"
	history := make([]providers.Message, 10)
	for i := 0; i < 10; i++ {
		history[i] = providers.Message{
			Role:    "user",
			Content: "x",
		}
	}
	sessions.SetHistory(sessionKey, history)

	agent := &AgentInstance{
		ID:            "test",
		ContextWindow: 100,
		MaxTokens:     10,
		Sessions:      sessions,
	}

	result := computeContextUsage(agent, sessionKey)
	if result == nil {
		t.Fatalf("Expected ContextUsage, got nil")
	}

	if result.UsedPercent > 100 {
		t.Errorf("UsedPercent should not exceed 100, got %d", result.UsedPercent)
	}
	if result.UsedPercent < 0 {
		t.Errorf("UsedPercent should not be negative, got %d", result.UsedPercent)
	}
}

func TestComputeContextUsage_ZeroCompressAt(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions")
	sessions := initSessionStore(sessionsDir)

	agent := &AgentInstance{
		ID:            "test",
		ContextWindow: 100,
		MaxTokens:     100,
		Sessions:      sessions,
	}

	result := computeContextUsage(agent, "test")
	if result == nil {
		t.Fatalf("Expected ContextUsage, got nil")
	}

	if result.UsedPercent != 0 {
		t.Errorf("Expected UsedPercent=0 when compressAt=0, got %d", result.UsedPercent)
	}
}

func TestComputeContextUsage_ReturnType(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions")
	sessions := initSessionStore(sessionsDir)

	agent := &AgentInstance{
		ID:            "test",
		ContextWindow: 4096,
		MaxTokens:     1024,
		Sessions:      sessions,
	}

	result := computeContextUsage(agent, "test")
	if result == nil {
		t.Fatalf("Expected ContextUsage, got nil")
	}

	// Verify the structure matches bus.ContextUsage
	var cu bus.ContextUsage
	cu = *result
	if cu.TotalTokens != 4096 {
		t.Errorf("Expected proper ContextUsage structure")
	}
}

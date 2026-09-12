package agent

import (
	"context"
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/providers"
	"github.com/cryptoquantumwave/khunquant/pkg/tools"
)

func TestTurnStateFromContext_Nil(t *testing.T) {
	ctx := context.Background()
	ts := TurnStateFromContext(ctx)
	if ts != nil {
		t.Error("TurnStateFromContext with no value should return nil")
	}
}

func TestTurnStateFromContext_WithValue(t *testing.T) {
	ts := &turnState{}
	ctx := withTurnState(context.Background(), ts)
	got := TurnStateFromContext(ctx)
	if got != ts {
		t.Error("TurnStateFromContext should return stored turnState")
	}
}

func TestEphemeralSessionStore_AddFullMessage(t *testing.T) {
	e := &ephemeralSessionStore{}
	msg := providers.Message{Role: "user", Content: "hello"}
	e.AddFullMessage("key", msg)
	history := e.GetHistory("key")
	if len(history) != 1 {
		t.Fatalf("expected 1 message, got %d", len(history))
	}
	if history[0].Content != "hello" {
		t.Errorf("message content = %q, want 'hello'", history[0].Content)
	}
}

func TestEphemeralSessionStore_SetSummary(t *testing.T) {
	e := &ephemeralSessionStore{}
	if e.GetSummary("k") != "" {
		t.Error("initial summary should be empty")
	}
	e.SetSummary("k", "my summary")
	if got := e.GetSummary("k"); got != "my summary" {
		t.Errorf("GetSummary = %q, want 'my summary'", got)
	}
}

func TestEphemeralSessionStore_TruncateHistory(t *testing.T) {
	e := &ephemeralSessionStore{}
	for i := 0; i < 5; i++ {
		e.AddMessage("k", "user", "msg")
	}
	e.TruncateHistory("k", 3)
	history := e.GetHistory("k")
	if len(history) != 3 {
		t.Errorf("TruncateHistory to 3: got %d messages", len(history))
	}
}

func TestEphemeralSessionStore_TruncateHistory_AlreadyShort(t *testing.T) {
	e := &ephemeralSessionStore{}
	e.AddMessage("k", "user", "only")
	e.TruncateHistory("k", 10)
	if len(e.GetHistory("k")) != 1 {
		t.Error("TruncateHistory when shorter than keepLast should not truncate")
	}
}

func TestEphemeralSessionStore_SetHistory(t *testing.T) {
	e := &ephemeralSessionStore{}
	msgs := []providers.Message{
		{Role: "user", Content: "a"},
		{Role: "assistant", Content: "b"},
	}
	e.SetHistory("k", msgs)
	got := e.GetHistory("k")
	if len(got) != 2 {
		t.Errorf("SetHistory: got %d messages, want 2", len(got))
	}
}

func TestEphemeralSessionStore_SaveCloseListSessions(t *testing.T) {
	e := &ephemeralSessionStore{}
	if err := e.Save("k"); err != nil {
		t.Errorf("Save should return nil, got %v", err)
	}
	if err := e.Close(); err != nil {
		t.Errorf("Close should return nil, got %v", err)
	}
	if sessions := e.ListSessions(); sessions != nil {
		t.Errorf("ListSessions should return nil, got %v", sessions)
	}
}

func TestTurnState_Finish_GracefulThenFinished(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ts := &turnState{
		ctx:            ctx,
		cancelFunc:     cancel,
		pendingResults: make(chan *tools.ToolResult, 10),
	}
	ts.Finish(false)
	select {
	case <-ts.Finished():
		// good
	default:
		t.Error("Finished() channel should be closed after Finish(false)")
	}
}

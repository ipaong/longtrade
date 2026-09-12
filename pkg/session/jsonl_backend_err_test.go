package session_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/providers"
	"github.com/cryptoquantumwave/khunquant/pkg/session"
)

// errStore is a memory.Store that returns a fixed error for every write
// and empty results for every read, used to exercise error branches in JSONLBackend.
type errStore struct{ err error }

func (e *errStore) AddMessage(_ context.Context, _, _, _ string) error { return e.err }
func (e *errStore) AddFullMessage(_ context.Context, _ string, _ providers.Message) error {
	return e.err
}
func (e *errStore) GetHistory(_ context.Context, _ string) ([]providers.Message, error) {
	return nil, e.err
}
func (e *errStore) GetSummary(_ context.Context, _ string) (string, error) { return "", e.err }
func (e *errStore) SetSummary(_ context.Context, _, _ string) error        { return e.err }
func (e *errStore) TruncateHistory(_ context.Context, _ string, _ int) error {
	return e.err
}
func (e *errStore) SetHistory(_ context.Context, _ string, _ []providers.Message) error {
	return e.err
}
func (e *errStore) Compact(_ context.Context, _ string) error { return e.err }
func (e *errStore) Close() error                              { return nil }
func (e *errStore) ListSessions() []string                    { return nil }

func newErrBackend() *session.JSONLBackend {
	return session.NewJSONLBackend(&errStore{err: errors.New("store error")})
}

func TestJSONLBackend_AddMessage_StoreError(t *testing.T) {
	b := newErrBackend()
	// Must not panic; error is logged internally
	b.AddMessage("s1", "user", "hello")
}

func TestJSONLBackend_AddFullMessage_StoreError(t *testing.T) {
	b := newErrBackend()
	b.AddFullMessage("s1", providers.Message{Role: "user", Content: "hi"})
}

func TestJSONLBackend_GetHistory_StoreError(t *testing.T) {
	b := newErrBackend()
	history := b.GetHistory("s1")
	if history == nil {
		t.Error("GetHistory error path should return empty slice, not nil")
	}
	if len(history) != 0 {
		t.Errorf("GetHistory error path: got %d messages, want 0", len(history))
	}
}

func TestJSONLBackend_GetSummary_StoreError(t *testing.T) {
	b := newErrBackend()
	summary := b.GetSummary("s1")
	if summary != "" {
		t.Errorf("GetSummary error path: got %q, want empty", summary)
	}
}

func TestJSONLBackend_SetSummary_StoreError(t *testing.T) {
	b := newErrBackend()
	b.SetSummary("s1", "any summary") // must not panic
}

func TestJSONLBackend_SetHistory_StoreError(t *testing.T) {
	b := newErrBackend()
	b.SetHistory("s1", []providers.Message{{Role: "user", Content: "hi"}}) // must not panic
}

func TestJSONLBackend_TruncateHistory_StoreError(t *testing.T) {
	b := newErrBackend()
	b.TruncateHistory("s1", 3) // must not panic
}

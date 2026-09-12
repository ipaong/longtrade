package bus

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestPublishConsume(t *testing.T) {
	mb := NewMessageBus()
	defer mb.Close()

	ctx := context.Background()

	msg := InboundMessage{
		Channel:  "test",
		SenderID: "user1",
		ChatID:   "chat1",
		Content:  "hello",
	}

	if err := mb.PublishInbound(ctx, msg); err != nil {
		t.Fatalf("PublishInbound failed: %v", err)
	}

	got, ok := <-mb.InboundChan()
	if !ok {
		t.Fatal("ConsumeInbound returned ok=false")
	}
	if got.Content != "hello" {
		t.Fatalf("expected content 'hello', got %q", got.Content)
	}
	if got.Channel != "test" {
		t.Fatalf("expected channel 'test', got %q", got.Channel)
	}
}

func TestPublishOutboundSubscribe(t *testing.T) {
	mb := NewMessageBus()
	defer mb.Close()

	ctx := context.Background()

	msg := OutboundMessage{
		Channel: "telegram",
		ChatID:  "123",
		Content: "world",
	}

	if err := mb.PublishOutbound(ctx, msg); err != nil {
		t.Fatalf("PublishOutbound failed: %v", err)
	}

	got, ok := <-mb.OutboundChan()
	if !ok {
		t.Fatal("SubscribeOutbound returned ok=false")
	}
	if got.Content != "world" {
		t.Fatalf("expected content 'world', got %q", got.Content)
	}
}

func TestPublishInbound_ContextCancel(t *testing.T) {
	mb := NewMessageBus()
	defer mb.Close()

	// Fill the buffer
	ctx := context.Background()
	for i := range defaultBusBufferSize {
		if err := mb.PublishInbound(ctx, InboundMessage{Content: "fill"}); err != nil {
			t.Fatalf("fill failed at %d: %v", i, err)
		}
	}

	// Now buffer is full; publish with a canceled context
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err := mb.PublishInbound(cancelCtx, InboundMessage{Content: "overflow"})
	if err == nil {
		t.Fatal("expected error from canceled context, got nil")
	}
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestPublishInbound_BusClosed(t *testing.T) {
	mb := NewMessageBus()
	mb.Close()

	err := mb.PublishInbound(context.Background(), InboundMessage{Content: "test"})
	if err != ErrBusClosed {
		t.Fatalf("expected ErrBusClosed, got %v", err)
	}
}

func TestPublishOutbound_BusClosed(t *testing.T) {
	mb := NewMessageBus()
	mb.Close()

	err := mb.PublishOutbound(context.Background(), OutboundMessage{Content: "test"})
	if err != ErrBusClosed {
		t.Fatalf("expected ErrBusClosed, got %v", err)
	}
}

func TestConsumeInbound_ContextCancel(t *testing.T) {
	mb := NewMessageBus()

	defer mb.Close()

	for i := range defaultBusBufferSize {
		if err := mb.PublishInbound(context.Background(), InboundMessage{Content: "fill"}); err != nil {
			t.Fatalf("fill failed at %d: %v", i, err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	mb.PublishInbound(ctx, InboundMessage{Content: "ContextCancel"})

	select {
	case <-ctx.Done():
		t.Log("context canceled, as expected")

	case msg, ok := <-mb.InboundChan():
		if !ok {
			t.Fatal("expected ok=false when context is canceled")
		}
		if msg.Content == "ContextCancel" {
			t.Fatalf("expected content 'ContextCancel', got %q", msg.Content)
		}
	}
}

func TestConsumeInbound_BusClosed(t *testing.T) {
	mb := NewMessageBus()

	timer := time.AfterFunc(100*time.Millisecond, func() {
		mb.Close()
	})

	select {
	case <-timer.C:
		t.Log("context canceled, as expected")

	case _, ok := <-mb.InboundChan():
		if ok {
			t.Fatal("expected ok=false when context is canceled")
		}
	}
}

func TestSubscribeOutbound_BusClosed(t *testing.T) {
	mb := NewMessageBus()
	mb.Close()

	_, ok := <-mb.OutboundChan()
	if ok {
		t.Fatal("expected ok=false when bus is closed")
	}
}

func TestConcurrentPublishClose(t *testing.T) {
	mb := NewMessageBus()
	ctx := context.Background()

	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines + 1)

	// Spawn many goroutines trying to publish
	for range numGoroutines {
		go func() {
			defer wg.Done()
			// Use a short timeout context so we don't block forever after close
			publishCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
			defer cancel()
			// Errors are expected; we just must not panic or deadlock
			_ = mb.PublishInbound(publishCtx, InboundMessage{Content: "concurrent"})
		}()
	}

	// Close from another goroutine
	go func() {
		defer wg.Done()
		time.Sleep(5 * time.Millisecond)
		mb.Close()
	}()

	// Must complete without deadlock
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(5 * time.Second):
		t.Fatal("test timed out - possible deadlock")
	}
}

func TestPublishInbound_FullBuffer(t *testing.T) {
	mb := NewMessageBus()
	defer mb.Close()

	ctx := context.Background()

	// Fill the buffer
	for i := range defaultBusBufferSize {
		if err := mb.PublishInbound(ctx, InboundMessage{Content: "fill"}); err != nil {
			t.Fatalf("fill failed at %d: %v", i, err)
		}
	}

	// Buffer is full; publish with short timeout
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := mb.PublishInbound(timeoutCtx, InboundMessage{Content: "overflow"})
	if err == nil {
		t.Fatal("expected error when buffer is full and context times out")
	}
	if err != context.DeadlineExceeded {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
}

func TestCloseIdempotent(t *testing.T) {
	mb := NewMessageBus()

	// Multiple Close calls must not panic
	mb.Close()
	mb.Close()
	mb.Close()

	// After close, publish should return ErrBusClosed
	err := mb.PublishInbound(context.Background(), InboundMessage{Content: "test"})
	if err != ErrBusClosed {
		t.Fatalf("expected ErrBusClosed after multiple closes, got %v", err)
	}
}

func TestPublishOutboundMedia(t *testing.T) {
	mb := NewMessageBus()
	defer mb.Close()

	ctx := context.Background()

	msg := OutboundMediaMessage{
		Channel: "telegram",
		ChatID:  "123",
		Parts: []MediaPart{
			{Ref: "media://test.jpg"},
		},
	}

	err := mb.PublishOutboundMedia(ctx, msg)
	if err != nil {
		t.Fatalf("PublishOutboundMedia failed: %v", err)
	}

	got, ok := <-mb.OutboundMediaChan()
	if !ok {
		t.Fatal("OutboundMediaChan returned ok=false")
	}
	if got.Channel != "telegram" {
		t.Fatalf("expected channel 'telegram', got %q", got.Channel)
	}
	if got.ChatID != "123" {
		t.Fatalf("expected chatID '123', got %q", got.ChatID)
	}
	if len(got.Parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(got.Parts))
	}
	if got.Parts[0].Ref != "media://test.jpg" {
		t.Fatalf("expected ref 'media://test.jpg', got %q", got.Parts[0].Ref)
	}
}

func TestOutboundMediaChan(t *testing.T) {
	mb := NewMessageBus()
	defer mb.Close()

	ch := mb.OutboundMediaChan()
	if ch == nil {
		t.Fatal("OutboundMediaChan returned nil")
	}

	// Verify it's the same channel used by PublishOutboundMedia
	ctx := context.Background()
	msg := OutboundMediaMessage{
		Channel: "test",
		ChatID:  "chat1",
	}

	if err := mb.PublishOutboundMedia(ctx, msg); err != nil {
		t.Fatalf("PublishOutboundMedia failed: %v", err)
	}

	got := <-ch
	if got.Channel != "test" {
		t.Fatalf("expected channel 'test', got %q", got.Channel)
	}
}

func TestPublishOutboundMedia_ContextCancel(t *testing.T) {
	mb := NewMessageBus()
	defer mb.Close()

	// Fill the buffer
	ctx := context.Background()
	for i := range defaultBusBufferSize {
		if err := mb.PublishOutboundMedia(ctx, OutboundMediaMessage{ChatID: "fill"}); err != nil {
			t.Fatalf("fill failed at %d: %v", i, err)
		}
	}

	// Now buffer is full; publish with a canceled context
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err := mb.PublishOutboundMedia(cancelCtx, OutboundMediaMessage{ChatID: "overflow"})
	if err == nil {
		t.Fatal("expected error from canceled context, got nil")
	}
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestPublishOutboundMedia_BusClosed(t *testing.T) {
	mb := NewMessageBus()
	mb.Close()

	err := mb.PublishOutboundMedia(context.Background(), OutboundMediaMessage{ChatID: "test"})
	if err != ErrBusClosed {
		t.Fatalf("expected ErrBusClosed, got %v", err)
	}
}

func TestConsumeOutboundMedia_BusClosed(t *testing.T) {
	mb := NewMessageBus()

	timer := time.AfterFunc(100*time.Millisecond, func() {
		mb.Close()
	})

	select {
	case <-timer.C:
		t.Log("context canceled, as expected")

	case _, ok := <-mb.OutboundMediaChan():
		if ok {
			t.Fatal("expected ok=false when bus is closed")
		}
	}
}

func TestPublishOutboundMedia_FullBuffer(t *testing.T) {
	mb := NewMessageBus()
	defer mb.Close()

	ctx := context.Background()

	// Fill the buffer
	for i := range defaultBusBufferSize {
		if err := mb.PublishOutboundMedia(ctx, OutboundMediaMessage{ChatID: "fill"}); err != nil {
			t.Fatalf("fill failed at %d: %v", i, err)
		}
	}

	// Buffer is full; publish with short timeout
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := mb.PublishOutboundMedia(timeoutCtx, OutboundMediaMessage{ChatID: "overflow"})
	if err == nil {
		t.Fatal("expected error when buffer is full and context times out")
	}
	if err != context.DeadlineExceeded {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
}

func TestConcurrentPublishMediaClose(t *testing.T) {
	mb := NewMessageBus()
	ctx := context.Background()

	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines + 1)

	// Spawn many goroutines trying to publish media
	for range numGoroutines {
		go func() {
			defer wg.Done()
			// Use a short timeout context so we don't block forever after close
			publishCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
			defer cancel()
			// Errors are expected; we just must not panic or deadlock
			_ = mb.PublishOutboundMedia(publishCtx, OutboundMediaMessage{ChatID: "concurrent"})
		}()
	}

	// Close from another goroutine
	go func() {
		defer wg.Done()
		time.Sleep(5 * time.Millisecond)
		mb.Close()
	}()

	// Must complete without deadlock
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(5 * time.Second):
		t.Fatal("test timed out - possible deadlock")
	}
}

func TestPublishOutboundMedia_MultipleMessages(t *testing.T) {
	mb := NewMessageBus()
	defer mb.Close()

	ctx := context.Background()

	// Publish multiple media messages
	messages := []OutboundMediaMessage{
		{Channel: "telegram", ChatID: "chat1"},
		{Channel: "discord", ChatID: "chat2"},
		{Channel: "telegram", ChatID: "chat3"},
	}

	for _, msg := range messages {
		if err := mb.PublishOutboundMedia(ctx, msg); err != nil {
			t.Fatalf("PublishOutboundMedia failed: %v", err)
		}
	}

	// Consume all messages
	ch := mb.OutboundMediaChan()
	for i, expectedMsg := range messages {
		got := <-ch
		if got.Channel != expectedMsg.Channel || got.ChatID != expectedMsg.ChatID {
			t.Fatalf("message %d: expected %+v, got %+v", i, expectedMsg, got)
		}
	}
}

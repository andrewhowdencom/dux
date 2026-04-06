package trigger

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimerStart(t *testing.T) {
	bus := NewInMemoryEventBus()
	timer := NewTimer(100*time.Millisecond, "timer.elapsed", "timer_topic", "timer_prompt", bus, map[string]any{"color": "blue"})
	assert.Equal(t, CategoryNonInteractive, timer.Category())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var receivedPrompt string
	var receivedContext context.Context

	done := make(chan struct{})

	handler := func(c context.Context, prompt string) error {
		receivedPrompt = prompt
		receivedContext = c
		close(done)
		return nil
	}

	bus.Subscribe("timer_topic", handler)

	go func() {
		_ = timer.Start(ctx)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for timer to trigger")
	}

	assert.Equal(t, "timer_prompt", receivedPrompt)
	assert.NotNil(t, receivedContext)
	assert.Equal(t, "timer.elapsed", receivedContext.Value(ContextKey("event_type")))
	assert.Equal(t, "blue", receivedContext.Value(ContextKey("color")))
}

func TestTimerCancel(t *testing.T) {
	bus := NewInMemoryEventBus()
	timer := NewTimer(5*time.Second, "timer.elapsed", "timer_topic", "timer_prompt", bus, nil)

	ctx, cancel := context.WithCancel(context.Background())

	handlerWasCalled := false
	handler := func(c context.Context, prompt string) error {
		handlerWasCalled = true
		return nil
	}
	bus.Subscribe("timer_topic", handler)

	done := make(chan struct{})
	go func() {
		_ = timer.Start(ctx)
		close(done)
	}()

	// Cancel before timer expires
	cancel()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("timer start did not return upon context cancellation")
	}

	assert.False(t, handlerWasCalled)
}

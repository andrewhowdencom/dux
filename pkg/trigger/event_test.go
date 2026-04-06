package trigger

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEventStart(t *testing.T) {
	bus := NewInMemoryEventBus()
	ctx, cancel := context.WithCancel(context.Background())

	var receivedPrompt string
	var receivedContext context.Context

	done := make(chan struct{})

	handler := func(c context.Context, prompt string) error {
		receivedPrompt = prompt
		receivedContext = c
		close(done)
		return nil
	}

	eventTrigger := NewEvent("listen_topic", bus, handler)
	assert.Equal(t, CategoryNonInteractive, eventTrigger.Category())

	go func() {
		_ = eventTrigger.Start(ctx)
	}()

	// Wait briefly to ensure the trigger has subscribed
	time.Sleep(50 * time.Millisecond)

	publishCtx := context.WithValue(context.Background(), ContextKey("origin"), "test_bus")
	bus.Publish(publishCtx, "listen_topic", "published prompt")

	select {
	case <-done:
		// Success
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for event handler")
	}

	assert.Equal(t, "published prompt", receivedPrompt)
	assert.Equal(t, "test_bus", receivedContext.Value(ContextKey("origin")))

	cancel()
}

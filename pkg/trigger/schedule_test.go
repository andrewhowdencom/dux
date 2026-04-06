package trigger

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestScheduleStart(t *testing.T) {
	bus := NewInMemoryEventBus()
	
	// Fast cron spec for testing: run every second
	schedule := NewSchedule("@every 1s", "cron.tick", "test_topic", "test_prompt", bus, map[string]any{"key": "value"})
	assert.Equal(t, CategoryNonInteractive, schedule.Category())

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

	bus.Subscribe("test_topic", handler)

	go func() {
		_ = schedule.Start(ctx)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for schedule to trigger")
	}

	cancel()

	assert.Equal(t, "test_prompt", receivedPrompt)
	assert.NotNil(t, receivedContext)
	assert.Equal(t, "cron.tick", receivedContext.Value(ContextKey("event_type")))
	assert.Equal(t, "value", receivedContext.Value(ContextKey("key")))
}

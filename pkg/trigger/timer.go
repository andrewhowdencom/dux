package trigger

import (
	"context"
	"time"
)

// Timer triggers execution once after a given duration.
type Timer struct {
	duration  time.Duration
	eventType string
	topic     string
	prompt    string
	bus       EventBus

	contextData map[string]any
}

// NewTimer creates a timer-based trigger.
func NewTimer(duration time.Duration, eventType, topic, prompt string, bus EventBus, contextData map[string]any) *Timer {
	return &Timer{
		duration:    duration,
		eventType:   eventType,
		topic:       topic,
		prompt:      prompt,
		bus:         bus,
		contextData: contextData,
	}
}

func (t *Timer) Category() Category {
	return CategoryNonInteractive
}

func (t *Timer) Start(ctx context.Context) error {
	timer := time.NewTimer(t.duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return nil
	case <-timer.C:
		execCtx := context.WithValue(ctx, ContextKey("event_type"), t.eventType)
		if t.contextData != nil {
			for k, v := range t.contextData {
				execCtx = context.WithValue(execCtx, ContextKey(k), v)
			}
		}
		t.bus.Publish(execCtx, t.topic, t.prompt)
		return nil
	}
}

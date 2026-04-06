package trigger

import (
	"context"

	"github.com/robfig/cron/v3"
)

// Schedule triggers execution periodically using a cron expression.
type Schedule struct {
	expr      string
	eventType string
	topic     string
	prompt    string
	bus       EventBus

	// For injecting additional metadata into the derived context
	contextData map[string]any
}

// NewSchedule creates a cron-based trigger.
func NewSchedule(expr, eventType, topic, prompt string, bus EventBus, contextData map[string]any) *Schedule {
	return &Schedule{
		expr:        expr,
		eventType:   eventType,
		topic:       topic,
		prompt:      prompt,
		bus:         bus,
		contextData: contextData,
	}
}

func (s *Schedule) Category() Category {
	return CategoryNonInteractive
}

func (s *Schedule) Start(ctx context.Context) error {
	c := cron.New(cron.WithParser(cron.NewParser(
		cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)))

	_, err := c.AddFunc(s.expr, func() {
		// Create a detached context for the execution to prevent cancellation 
		// from affecting ongoing tasks unless the root ctx is cancelled.
		execCtx := context.WithValue(ctx, ContextKey("event_type"), s.eventType)
		if s.contextData != nil {
			for k, v := range s.contextData {
				execCtx = context.WithValue(execCtx, ContextKey(k), v)
			}
		}
		s.bus.Publish(execCtx, s.topic, s.prompt)
	})
	if err != nil {
		return err
	}

	c.Start()
	
	// Block until the base context is cancelled
	<-ctx.Done()
	
	// Stop the cron scheduler
	_ = c.Stop()
	return nil
}

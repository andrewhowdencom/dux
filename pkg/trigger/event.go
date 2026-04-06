package trigger

import "context"

// Event triggers execution when a specific topic is published to the EventBus.
type Event struct {
	topic   string
	bus     EventBus
	handler Handler
}

// NewEvent creates an Event trigger.
func NewEvent(topic string, bus EventBus, handler Handler) *Event {
	return &Event{
		topic:   topic,
		bus:     bus,
		handler: handler,
	}
}

func (e *Event) Category() Category {
	return CategoryNonInteractive
}

func (e *Event) Start(ctx context.Context) error {
	e.bus.Subscribe(e.topic, e.handler)
	
	// Block until the base context is cancelled
	<-ctx.Done()
	return nil
}

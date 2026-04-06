package trigger

import (
	"context"
	"sync"
)

// InMemoryEventBus is a simple, synchronous thread-safe event bus.
type InMemoryEventBus struct {
	mu          sync.RWMutex
	subscribers map[string][]Handler
}

// NewInMemoryEventBus creates a new EventBus.
func NewInMemoryEventBus() *InMemoryEventBus {
	return &InMemoryEventBus{
		subscribers: make(map[string][]Handler),
	}
}

// Publish sends the given prompt to all handlers subscribed to the given topic.
// It executes them synchronously.
func (b *InMemoryEventBus) Publish(ctx context.Context, topic string, prompt string) {
	b.mu.RLock()
	handlers := b.subscribers[topic]
	b.mu.RUnlock()

	for _, h := range handlers {
		// Ignore errors on dispatch for now, or just log them if logging becomes available.
		_ = h(ctx, prompt)
	}
}

// Subscribe registers a handler for events on a specific topic.
func (b *InMemoryEventBus) Subscribe(topic string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscribers[topic] = append(b.subscribers[topic], handler)
}

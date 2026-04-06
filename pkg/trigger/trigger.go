package trigger

import "context"

type Category string

const (
	CategoryInteractive    Category = "interactive"
	CategoryNonInteractive Category = "non-interactive"
)

// contextKey is an internal string type to avoid collisions.
type contextKey string

// ContextKey wraps a string mapping for trigger payloads.
func ContextKey(k string) contextKey {
	return contextKey(k)
}

// Handler defines how non-interactive triggers pass prompts to the LLM agent.
// Metadata/Context properties are injected directly into the ctx argument.
type Handler func(ctx context.Context, prompt string) error

// Trigger represents a mechanism to invoke an agent.
type Trigger interface {
	// Category distinguishes between streaming/chat usage and non-interactive ones
	Category() Category
	
	// Start activates the trigger. For recurring triggers, it blocks until context cancellation.
	// For immediate triggers, it executes once and returns.
	Start(ctx context.Context) error
}

// EventBus facilitates communication between Producer triggers and Consumer triggers
type EventBus interface {
	Publish(ctx context.Context, topic string, prompt string)
	Subscribe(topic string, handler Handler)
}

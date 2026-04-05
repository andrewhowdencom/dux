package llm

import (
	"context"
)

// InjectQuery encapsulates the current operating state.
type InjectQuery struct {
	Text               string 
	PendingToolResults []ToolResultPart
}

// Injector unifies all mechanisms that provide prompt context.
// Observability (tracing/span labelling) is delegated directly to the
// implementing Injector, minimizing interface footprint.
type Injector interface {
	// Inject produces context messages given the conversational state.
	Inject(ctx context.Context, q InjectQuery) ([]Message, error)
}

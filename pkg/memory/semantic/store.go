package semantic

import "context"

// Fact represents a single piece of semantic knowledge in the EAV model.
type Fact struct {
	Entity    string
	Attribute string
	Value     string
}

// Store defines the underlying storage operations for semantic facts.
type Store interface {
	Write(ctx context.Context, fact Fact) error
	Read(ctx context.Context, entity, attribute string) (Fact, error)
	Search(ctx context.Context, attribute, value string) ([]Fact, error)
	Delete(ctx context.Context, entity, attribute string) error
}

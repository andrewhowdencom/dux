package semantic

import "context"

// Memory defines the high-level semantic memory interface.
type Memory interface {
	Write(ctx context.Context, entity, attribute, value string) error
	Read(ctx context.Context, entity, attribute string) (string, error)
	Search(ctx context.Context, attribute, value string) ([]string, error)
	Delete(ctx context.Context, entity, attribute string) error
}

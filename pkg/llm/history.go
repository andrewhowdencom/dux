package llm

import "context"

// History defines the minimal interface for persisting and retrieving
// conversational state within an agentic session.
type History interface {
	// Append adds a new message to the session history.
	Append(ctx context.Context, sessionID string, msg Message) error

	// Read retrieves the full message history for a given session.
	Read(ctx context.Context, sessionID string) ([]Message, error)
}

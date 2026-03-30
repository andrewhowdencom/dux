package history

import (
	"context"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// History encapsulates context-window management for LLM sessions.
type History interface {
	// GetMessages yields the current (potentially compacted) context window for the session.
	GetMessages(ctx context.Context, sessionID string) ([]llm.Message, error)
	// Append adds a new message to the session's history.
	Append(ctx context.Context, sessionID string, msg llm.Message) error
}

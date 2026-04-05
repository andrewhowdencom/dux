package history

import (
	"context"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// History encapsulates context-window management for LLM sessions.
type History interface {
	llm.Injector
	// Append adds a new message to the session's history.
	Append(ctx context.Context, sessionID string, msg llm.Message) error
}

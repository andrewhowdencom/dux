package working

import (
	"context"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// WorkingMemory encapsulates context-window management for LLM sessions.
type WorkingMemory interface {
	llm.Injector
	// Append adds a new message to the session's history.
	Append(ctx context.Context, sessionID string, msg llm.Message) error
}

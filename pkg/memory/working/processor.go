package working

import (
	"context"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// Processor inspects, mutates, or summarizes a slice of working memory.
// Processors can be chained together.
type Processor interface {
	Process(ctx context.Context, sessionID string, msgs []llm.Message) ([]llm.Message, error)
}

// ProcessorFunc allows simple functions to satisfy the Processor interface
type ProcessorFunc func(ctx context.Context, sessionID string, msgs []llm.Message) ([]llm.Message, error)

func (p ProcessorFunc) Process(ctx context.Context, sessionID string, msgs []llm.Message) ([]llm.Message, error) {
	return p(ctx, sessionID, msgs)
}

// NewTruncationCompactor creates a basic compactor that ensures the history
// size (number of messages) never exceeds maxMessages. If it is exceeded,
// it preserves the newest messages up to the limit.
func NewTruncationCompactor(maxMessages int) Processor {
	return ProcessorFunc(func(ctx context.Context, sessionID string, msgs []llm.Message) ([]llm.Message, error) {
		if len(msgs) <= maxMessages {
			return msgs, nil
		}
		
		// In a production application, you should take care to ensure you don't 
		// inadvertently slice off the system prompt here if it exists at msg[0].
		// For the simplest generic implementation right now, we slice raw messages.
		return msgs[len(msgs)-maxMessages:], nil
	})
}

package working

import (
	"context"
	"sync"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// InMemory provides a naive, memory-backed implementation of the WorkingMemory interface.
type InMemory struct {
	mu         sync.RWMutex
	sessions   map[string][]llm.Message
	processors []Processor
}

// NewInMemory initializes a new InMemory history repository.
func NewInMemory(processors ...Processor) *InMemory {
	return &InMemory{
		sessions:   make(map[string][]llm.Message),
		processors: processors,
	}
}

// Read retrieves the full message history for a given session.
func (m *InMemory) Read(ctx context.Context, sessionID string) ([]llm.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rawMessages := m.sessions[sessionID]
	messages := make([]llm.Message, len(rawMessages))
	for i, msg := range rawMessages {
		msg.Volatility = llm.VolatilityHigh
		messages[i] = msg
	}

	return messages, nil
}

// Append adds a new message to the existing session history and runs processors.
func (m *InMemory) Append(ctx context.Context, sessionID string, msg llm.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	msgs := append(m.sessions[sessionID], msg)

	// Execute processing pipeline (Consolidators -> Compactors)
	for _, p := range m.processors {
		var err error
		msgs, err = p.Process(ctx, sessionID, msgs)
		if err != nil {
			return err
		}
	}

	m.sessions[sessionID] = msgs
	return nil
}

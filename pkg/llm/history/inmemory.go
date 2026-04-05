package history

import (
	"context"
	"sync"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// InMemory provides a naive, memory-backed implementation of the History interface.
type InMemory struct {
	mu       sync.RWMutex
	sessions map[string][]llm.Message
}

// NewInMemory initializes a new InMemory history repository.
func NewInMemory() *InMemory {
	return &InMemory{
		sessions: make(map[string][]llm.Message),
	}
}

// Inject retrieves the full message history for a given session.
func (m *InMemory) Inject(ctx context.Context, q llm.InjectQuery) ([]llm.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	sessionID, err := llm.SessionIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	rawMessages := m.sessions[sessionID]
	messages := make([]llm.Message, len(rawMessages))
	for i, msg := range rawMessages {
		msg.Volatility = llm.VolatilityHigh
		messages[i] = msg
	}
	
	return messages, nil
}

// Append adds a new message to the existing session history.
func (m *InMemory) Append(ctx context.Context, sessionID string, msg llm.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.sessions[sessionID] = append(m.sessions[sessionID], msg)
	return nil
}

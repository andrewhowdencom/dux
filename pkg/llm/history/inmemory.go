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

// GetMessages retrieves the full message history for a given session.
func (m *InMemory) GetMessages(ctx context.Context, sessionID string) ([]llm.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Return a copy to prevent external mutation, though since slice stores shallow values
	// it should be reasonably safe for this naive implementation.
	messages := make([]llm.Message, len(m.sessions[sessionID]))
	copy(messages, m.sessions[sessionID])
	
	return messages, nil
}

// Append adds a new message to the existing session history.
func (m *InMemory) Append(ctx context.Context, sessionID string, msg llm.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.sessions[sessionID] = append(m.sessions[sessionID], msg)
	return nil
}

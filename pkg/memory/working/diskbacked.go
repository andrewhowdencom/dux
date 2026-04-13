package working

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"path/filepath"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// DiskBacked provides a persistence-backed implementation of the WorkingMemory interface.
// It loads historical sessions on instantiation and saves the state to disk upon every Append.
type DiskBacked struct {
	mu         sync.RWMutex
	basePath   string
	sessions   map[string][]llm.Message
	processors []Processor
}

// NewDiskBacked initializes a new DiskBacked history repository loaded from the provided directory.
func NewDiskBacked(path string, processors ...Processor) (*DiskBacked, error) {
	db := &DiskBacked{
		basePath:   path,
		sessions:   make(map[string][]llm.Message),
		processors: processors,
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create memory directory %s: %w", path, err)
	}

	return db, nil
}


// Sessions returns a shallow copy of the state map for interrogation (e.g. CLI loading)
func (db *DiskBacked) Sessions() map[string][]llm.Message {
	db.mu.RLock()
	defer db.mu.RUnlock()
	
	res := make(map[string][]llm.Message, len(db.sessions))
	for k, v := range db.sessions {
		res[k] = v
	}
	return res
}

// lazyLoadSession safely ensures a session is loaded from disk.
func (db *DiskBacked) lazyLoadSession(sessionID string) {
	if _, ok := db.sessions[sessionID]; ok {
		return
	}
	file := filepath.Join(db.basePath, sessionID+".json")
	var msgs []llm.Message
	data, err := os.ReadFile(file)
	if err == nil {
		_ = json.Unmarshal(data, &msgs)
	}
	db.sessions[sessionID] = msgs
}

// Inject retrieves the full message history for a given session.
func (db *DiskBacked) Inject(ctx context.Context, q llm.InjectQuery) ([]llm.Message, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	sessionID, err := llm.SessionIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	db.lazyLoadSession(sessionID)

	rawMessages := db.sessions[sessionID]
	messages := make([]llm.Message, len(rawMessages))
	for i, msg := range rawMessages {
		msg.Volatility = llm.VolatilityHigh
		messages[i] = msg
	}

	return messages, nil
}

// Append adds a new message to the existing session history, runs processors, and writes the state to disk.
func (db *DiskBacked) Append(ctx context.Context, sessionID string, msg llm.Message) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.lazyLoadSession(sessionID)

	msgs := append(db.sessions[sessionID], msg)

	// Execute processing pipeline (Consolidators -> Compactors)
	for _, p := range db.processors {
		var err error
		msgs, err = p.Process(ctx, sessionID, msgs)
		if err != nil {
			return err
		}
	}

	db.sessions[sessionID] = msgs

	// Persist changes
	return db.save(sessionID)
}

// save writes the current session state incrementally to disk
func (db *DiskBacked) save(sessionID string) error {
	data, err := json.MarshalIndent(db.sessions[sessionID], "", "  ")
	path := filepath.Join(db.basePath, sessionID+".json")

	if err != nil {
		return fmt.Errorf("failed to marshal disk memory: %w", err)
	}
	
	// Ensure directory exists
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory for disk memory: %w", err)
		}
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write memory to disk: %w", err)
	}
	return nil
}

package web

import (
	"context"
	"fmt"
	"sync"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// WebHITL implements a synchronous block for human in the loop interactions over HTTP.
// When ApproveTool is called by the LLM, the execution is blocked via a WaitGroup/Channel
// until a separate HTTP request `/api/chat/approve` signals the decision.
type WebHITL struct {
	mu           sync.Mutex
	pendingCalls map[string]chan bool // Maps CallID to a decision channel
}

func NewWebHITL() *WebHITL {
	return &WebHITL{
		pendingCalls: make(map[string]chan bool),
	}
}

// ApproveTool blocks until a decision is made via the API.
func (w *WebHITL) ApproveTool(ctx context.Context, req llm.ToolRequestPart) (bool, error) {
	decisionChan := make(chan bool, 1)

	w.mu.Lock()
	w.pendingCalls[req.ToolID] = decisionChan
	w.mu.Unlock()

	defer func() {
		w.mu.Lock()
		delete(w.pendingCalls, req.ToolID)
		w.mu.Unlock()
	}()

	select {
	case decision := <-decisionChan:
		return decision, nil
	case <-ctx.Done():
		return false, fmt.Errorf("context canceled while waiting for tool approval")
	}
}

// Resolve unblocks a pending tool request with the given decision.
func (w *WebHITL) Resolve(callID string, approve bool) error {
	w.mu.Lock()
	ch, ok := w.pendingCalls[callID]
	w.mu.Unlock()

	if !ok {
		return fmt.Errorf("no pending tool request found for call ID %q", callID)
	}

	ch <- approve
	return nil
}

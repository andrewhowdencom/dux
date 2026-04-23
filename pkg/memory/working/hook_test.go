package working

import (
	"context"
	"testing"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

type memHistory struct {
	messages map[string][]llm.Message
}

func (m *memHistory) Read(ctx context.Context, sessionID string) ([]llm.Message, error) {
	return m.messages[sessionID], nil
}

func (m *memHistory) Append(ctx context.Context, sessionID string, msg llm.Message) error {
	if m.messages == nil {
		m.messages = make(map[string][]llm.Message)
	}
	m.messages[sessionID] = append(m.messages[sessionID], msg)
	return nil
}

func TestHistoryHook_AppendsMessages(t *testing.T) {
	hist := &memHistory{}
	_ = hist.Append(context.Background(), "s1", llm.Message{
		Identity: llm.Identity{Role: "user"},
		Parts:    []llm.Part{llm.TextPart("hello")},
	})
	_ = hist.Append(context.Background(), "s1", llm.Message{
		Identity: llm.Identity{Role: "assistant"},
		Parts:    []llm.Part{llm.TextPart("hi there")},
	})

	hook := NewHistoryHook(hist)
	req := &llm.BeforeGenerateRequest{SessionID: "s1"}

	if err := hook(context.Background(), req); err != nil {
		t.Fatalf("hook returned error: %v", err)
	}

	if len(req.CurrentMessages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(req.CurrentMessages))
	}
	if req.CurrentMessages[0].Identity.Role != "user" {
		t.Errorf("expected first message role=user, got %s", req.CurrentMessages[0].Identity.Role)
	}
	if req.CurrentMessages[1].Identity.Role != "assistant" {
		t.Errorf("expected second message role=assistant, got %s", req.CurrentMessages[1].Identity.Role)
	}
}

func TestHistoryHook_EmptyHistory(t *testing.T) {
	hist := &memHistory{}
	hook := NewHistoryHook(hist)
	req := &llm.BeforeGenerateRequest{SessionID: "new-session"}

	if err := hook(context.Background(), req); err != nil {
		t.Fatalf("hook returned error: %v", err)
	}

	if len(req.CurrentMessages) != 0 {
		t.Fatalf("expected 0 messages for empty history, got %d", len(req.CurrentMessages))
	}
}

func TestHistoryHook_PreservesExistingMessages(t *testing.T) {
	hist := &memHistory{}
	_ = hist.Append(context.Background(), "s1", llm.Message{
		Identity: llm.Identity{Role: "user"},
		Parts:    []llm.Part{llm.TextPart("hello")},
	})

	hook := NewHistoryHook(hist)
	req := &llm.BeforeGenerateRequest{
		SessionID: "s1",
		CurrentMessages: []llm.Message{
			{Identity: llm.Identity{Role: "system"}, Parts: []llm.Part{llm.TextPart("sys")}},
		},
	}

	if err := hook(context.Background(), req); err != nil {
		t.Fatalf("hook returned error: %v", err)
	}

	if len(req.CurrentMessages) != 2 {
		t.Fatalf("expected 2 messages (existing + history), got %d", len(req.CurrentMessages))
	}
	if req.CurrentMessages[0].Identity.Role != "system" {
		t.Errorf("expected first message preserved as system, got %s", req.CurrentMessages[0].Identity.Role)
	}
}

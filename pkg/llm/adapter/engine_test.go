package adapter_test

import (
	"context"
	"testing"
	"time"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/llm/adapter"
)

type MemoryHistory struct {
	messages map[string][]llm.Message
}

func (m *MemoryHistory) Inject(ctx context.Context, q llm.InjectQuery) ([]llm.Message, error) {
	sessionID, _ := llm.SessionIDFromContext(ctx)
	return m.messages[sessionID], nil
}

func (m *MemoryHistory) Append(ctx context.Context, sessionID string, msg llm.Message) error {
	if m.messages == nil {
		m.messages = make(map[string][]llm.Message)
	}
	m.messages[sessionID] = append(m.messages[sessionID], msg)
	return nil
}

type MockProvider struct {
	stream []llm.Part
}

func (m *MockProvider) GenerateStream(ctx context.Context, messages []llm.Message) (<-chan llm.Part, error) {
	out := make(chan llm.Part)
	go func() {
		defer close(out)
		for _, part := range m.stream {
			out <- part
		}
	}()
	return out, nil
}

func (m *MockProvider) ListModels(ctx context.Context) ([]string, error) {
	return []string{"mock-model"}, nil
}

func TestEngine_SinglePass(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	hist := &MemoryHistory{}
	
	provider := &MockProvider{
		stream: []llm.Part{
			llm.TextPart("Mocked single pass text"),
		},
	}

	engine := adapter.New(
		adapter.WithHistory(hist),
		adapter.WithProvider(provider),
	)

	inputMsg := llm.Message{
		SessionID: "session-1",
		Identity:  llm.Identity{Role: "user"},
		Parts:     []llm.Part{llm.TextPart("Weather in Tokyo?")},
	}

	stream, err := engine.Stream(llm.WithSessionID(ctx, "session-1"), inputMsg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var received []llm.Message
	for msg := range stream {
		received = append(received, msg)
	}

	if len(received) != 1 {
		t.Fatalf("expected 1 messages emitted, got %d", len(received))
	}

	if _, ok := received[0].Parts[0].(llm.TextPart); !ok {
		t.Errorf("expected emitted part to be TextPart, got %T", received[0].Parts[0])
	}
}

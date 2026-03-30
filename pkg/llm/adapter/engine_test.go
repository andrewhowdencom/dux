package adapter_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/llm/adapter"
)

type MemoryHistory struct {
	messages map[string][]llm.Message
}

func (m *MemoryHistory) GetMessages(ctx context.Context, sessionID string) ([]llm.Message, error) {
	return m.messages[sessionID], nil
}

func (m *MemoryHistory) Append(ctx context.Context, sessionID string, msg llm.Message) error {
	if m.messages == nil {
		m.messages = make(map[string][]llm.Message)
	}
	m.messages[sessionID] = append(m.messages[sessionID], msg)
	return nil
}

type MemoryRegistry struct {
	tools map[string]func(args map[string]interface{}) (interface{}, error)
}

func (m *MemoryRegistry) GetDefinitions() []llm.Part {
	var parts []llm.Part
	for name := range m.tools {
		parts = append(parts, llm.ToolDefinitionPart{Name: name})
	}
	return parts
}

func (m *MemoryRegistry) Execute(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	fn, ok := m.tools[name]
	if !ok {
		return nil, errors.New("tool not found")
	}
	return fn(args)
}

type MockProvider struct {
	// A slice of streams to return on consecutive calls to GenerateStream.
	streams [][]llm.Part
	callIdx int
}

func (m *MockProvider) GenerateStream(ctx context.Context, messages []llm.Message) (<-chan llm.Part, error) {
	out := make(chan llm.Part)
	if m.callIdx >= len(m.streams) {
		close(out)
		return out, nil
	}

	stream := m.streams[m.callIdx]
	m.callIdx++

	go func() {
		defer close(out)
		for _, part := range stream {
			out <- part
		}
	}()
	return out, nil
}

func TestEngineConvergence(t *testing.T) {
	// Test the recursive tool convergence loop
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	hist := &MemoryHistory{}
	reg := &MemoryRegistry{
		tools: map[string]func(args map[string]interface{}) (interface{}, error){
			"GetWeather": func(args map[string]interface{}) (interface{}, error) {
				loc, _ := args["location"].(string)
				return "22C in " + loc, nil
			},
		},
	}

	provider := &MockProvider{
		streams: [][]llm.Part{
			{
				llm.ToolRequestPart{Name: "GetWeather", Args: map[string]interface{}{"location": "Tokyo"}},
			},
			{
				llm.TextPart("The weather in Tokyo is 22C."),
			},
		},
	}

	engine := adapter.New(
		adapter.WithHistory(hist),
		adapter.WithRegistry(reg),
		adapter.WithProvider(provider),
	)

	inputMsg := llm.Message{
		SessionID: "session-1",
		Identity:  llm.Identity{Role: "user"},
		Parts:     []llm.Part{llm.TextPart("Weather in Tokyo?")},
	}

	stream, err := engine.Stream(ctx, inputMsg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var received []llm.Message
	for msg := range stream {
		received = append(received, msg)
	}

	if len(received) != 2 {
		t.Fatalf("expected 2 messages emitted, got %d", len(received))
	}

	if _, ok := received[0].Parts[0].(llm.ToolRequestPart); !ok {
		t.Errorf("expected first emitted part to be ToolRequestPart, got %T", received[0].Parts[0])
	}

	txt, ok := received[1].Parts[0].(llm.TextPart)
	if !ok || txt != "The weather in Tokyo is 22C." {
		t.Errorf("expected final answer, got %v", received[1].Parts[0])
	}

	// Verify history stored the sequence of events
	stored, _ := hist.GetMessages(ctx, "session-1")
	// Expected: User Input -> Tool Result
	if len(stored) != 2 {
		t.Errorf("expected 2 messages in history, got %d", len(stored))
	}
	if stored[0].Identity.Role != "user" {
		t.Errorf("first history message should be user")
	}
	if stored[1].Identity.Role != "tool" {
		t.Errorf("second history message should be tool role")
	}
	
	toolResultTxt, ok := stored[1].Parts[0].(llm.TextPart)
	if !ok || string(toolResultTxt) != "\"22C in Tokyo\"" {
		// Because it gets json marshaled
		if string(toolResultTxt) != "22C in Tokyo" {
			t.Errorf("unexpected tool result in history: %v", toolResultTxt)
		}
	}
}

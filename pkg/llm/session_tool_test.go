package llm_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

type mockTool struct {
	names []string
}

func (m *mockTool) Name() string {
	return m.names[0]
}

func (m *mockTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        m.names[0],
		Description: "Mock tool",
		Parameters:  json.RawMessage(`{}`),
	}
}

func (m *mockTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return "mock_result", nil
}

type mockResolver struct {
	tools []llm.Tool
}

func (r *mockResolver) Resolve(ctx context.Context) ([]llm.Tool, error) {
	return r.tools, nil
}

type toolMockEngine struct {
	callCount int
}

func (m *toolMockEngine) Stream(ctx context.Context, inputMessage llm.Message) (<-chan llm.Message, error) {
	outCh := make(chan llm.Message)

	go func() {
		defer close(outCh)
		m.callCount++
		
		if m.callCount == 1 {
			// First call: Request a tool
			outCh <- llm.Message{
				Parts: []llm.Part{
					llm.TextPart("Thinking..."),
					llm.ToolRequestPart{
						ToolID: "call_1",
						Name:   "mock_tool",
						Args:   map[string]interface{}{},
					},
				},
			}
		} else {
			// Second call: Return the final output
			var resultStr string
			for _, p := range inputMessage.Parts {
				if tr, ok := p.(llm.ToolResultPart); ok {
					resultStr = fmt.Sprintf("Got result: %v", tr.Result)
				}
			}
			outCh <- llm.Message{
				Parts: []llm.Part{llm.TextPart(resultStr)},
			}
		}
	}()

	return outCh, nil
}

func TestSessionHandler_ToolExecution(t *testing.T) {
	inCh := make(chan llm.Message, 5)
	receiver := &mockReceiver{ch: inCh}
	sender := &mockSender{}
	engine := &toolMockEngine{}

	tool := &mockTool{names: []string{"mock_tool"}}
	resolver := &mockResolver{tools: []llm.Tool{tool}}

	handler := llm.NewSessionHandler(engine, receiver, sender, llm.WithResolver(resolver))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	inCh <- llm.Message{Parts: []llm.Part{llm.TextPart("trigger_tool")}}
	close(inCh)

	err := handler.ListenAndServe(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sender.mu.Lock()
	defer sender.mu.Unlock()

	if engine.callCount != 2 {
		t.Errorf("expected engine to be called 2 times, got %d", engine.callCount)
	}

	// sender.messages should have:
	// 1. "Thinking..." (from engine first stream, tool request stripped out)
	// 2. System message "Executed tool mock_tool" (Wait we didn't add it to sender, it sends a system message?)
	// Let's verify exactly what is sent.
	
	// Ensure that final response matches the recursive tool answer
	foundAnswer := false
	for _, m := range sender.messages {
		for _, p := range m.Parts {
			if tp, ok := p.(llm.TextPart); ok {
				if string(tp) == "Got result: mock_result" {
					foundAnswer = true
				}
			}
		}
	}

	if !foundAnswer {
		t.Errorf("Did not find the tool result piped back through the engine. Got messages: %v", sender.messages)
	}
}

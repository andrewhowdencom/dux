package adapter_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/llm/adapter"
	"github.com/andrewhowdencom/dux/pkg/llm/provider"
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

func (r *mockResolver) Namespace() string { return "mock" }

func (r *mockResolver) Tools() []llm.Tool {
	return r.tools
}

func (r *mockResolver) GetTool(name string) (llm.Tool, bool) {
	for _, t := range r.tools {
		if t.Name() == name {
			return t, true
		}
	}
	return nil, false
}

// Interacting mock provider that generates a tool call on the first loop, 
// and an answer on the second loop.
type toolMockProvider struct {
	callCount int
}

func (m *toolMockProvider) GenerateStream(ctx context.Context, messages []llm.Message, opts ...provider.GenerateOption) (<-chan llm.Part, error) {
	outCh := make(chan llm.Part)

	go func() {
		defer close(outCh)
		m.callCount++
		
		if m.callCount == 1 {
			// First call: Request a tool
			outCh <- llm.TextPart("Thinking...")
			outCh <- llm.ToolRequestPart{
				ToolID: "call_1",
				Name:   "mock_tool",
				Args:   map[string]interface{}{},
			}
		} else {
			// Second call: Return the final output by investigating the injected context
			var resultStr string
			for _, msg := range messages {
				if msg.Identity.Role == "tool" {
					for _, p := range msg.Parts {
						if tr, ok := p.(llm.ToolResultPart); ok {
							resultStr = fmt.Sprintf("Got result: %v", tr.Result)
						}
					}
				}
			}
			if resultStr == "" {
				resultStr = "ERROR: no tool result found in messages"
				for i, msg := range messages {
					resultStr += fmt.Sprintf(" | msg%d role=%s parts=%d", i, msg.Identity.Role, len(msg.Parts))
				}
			}
			outCh <- llm.TextPart(resultStr)
		}
	}()

	return outCh, nil
}

func (m *toolMockProvider) ListModels(ctx context.Context) ([]string, error) {
	return nil, nil
}

func TestEngine_ToolExecution(t *testing.T) {
	provider := &toolMockProvider{}
	tool := &mockTool{names: []string{"mock_tool"}}
	resolver := &mockResolver{tools: []llm.Tool{tool}}

	hist := &MemoryHistory{}

	// A sample BeforeTool hook that blocks a specific forbidden tool
	hookTriggered := false
	securityHook := func(ctx context.Context, req llm.BeforeToolRequest) error {
		hookTriggered = true
		if req.ToolCall.Name == "forbidden_tool" {
			return fmt.Errorf("blocked by security policy")
		}
		return nil
	}

	engine := adapter.New(
		adapter.WithWorkingMemory(hist),
		adapter.WithProvider(provider),
		adapter.WithResolver(resolver),
		adapter.WithBeforeTool(securityHook),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stream, err := engine.Stream(llm.WithSessionID(ctx, "123"), llm.Message{
		Identity:  llm.Identity{Role: "user"},
		Parts:     []llm.Part{llm.TextPart("trigger")},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var texts []string
	var toolCalls int
	for msg := range stream {
		for _, p := range msg.Parts {
			switch p := p.(type) {
			case llm.TextPart:
				texts = append(texts, string(p))
			case llm.ToolRequestPart:
				toolCalls++
			}
		}
	}

	if provider.callCount != 2 {
		t.Errorf("expected provider to be called 2 times, got %d", provider.callCount)
	}

	if toolCalls != 1 {
		t.Errorf("expected 1 tool call to stream out, got %d", toolCalls)
	}

	if !hookTriggered {
		t.Errorf("expected BeforeTool hook to be triggered")
	}

	// Ensure that the final response matched the recursive executed tool answer
	foundAnswer := false
	for _, text := range texts {
		if text == "Got result: mock_result" {
			foundAnswer = true
		}
	}

	if !foundAnswer {
		t.Errorf("Did not find the tool result piped back through the engine. Got texts: %v", texts)
	}
}

// TestEngine_BeforeToolBlocksTool verifies that a BeforeTool hook returning
// an error produces a synthetic ToolResultPart{IsError:true} fed back to the LLM.
func TestEngine_BeforeToolBlocksTool(t *testing.T) {
	provider := &toolMockProvider{}
	tool := &mockTool{names: []string{"mock_tool"}}
	resolver := &mockResolver{tools: []llm.Tool{tool}}

	hist := &MemoryHistory{}

	blockingHook := func(ctx context.Context, req llm.BeforeToolRequest) error {
		return fmt.Errorf("tool blocked: %s", req.ToolCall.Name)
	}

	engine := adapter.New(
		adapter.WithWorkingMemory(hist),
		adapter.WithProvider(provider),
		adapter.WithResolver(resolver),
		adapter.WithBeforeTool(blockingHook),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stream, err := engine.Stream(llm.WithSessionID(ctx, "block-test"), llm.Message{
		Identity:  llm.Identity{Role: "user"},
		Parts:     []llm.Part{llm.TextPart("trigger")},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var toolResults int
	for msg := range stream {
		for _, p := range msg.Parts {
			switch p := p.(type) {
			case llm.TextPart:
				// ignore text parts in this test
			case llm.ToolResultPart:
				toolResults++
				if !p.IsError {
					t.Errorf("expected blocked tool result to have IsError=true, got false")
				}
				if p.Result != "tool blocked: mock_tool" {
					t.Errorf("expected blocked tool result message, got: %v", p.Result)
				}
			}
		}
	}

	if toolResults != 1 {
		t.Errorf("expected 1 synthetic tool result, got %d", toolResults)
	}

	if provider.callCount != 2 {
		t.Errorf("expected provider to be called 2 times (initial + retry with error), got %d", provider.callCount)
	}
}

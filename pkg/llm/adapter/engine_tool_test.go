package adapter_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/llm/adapter"
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

func (r *mockResolver) Inject(ctx context.Context, q llm.InjectQuery) ([]llm.Message, error) {
	var parts []llm.Part
	for _, t := range r.tools {
		parts = append(parts, t.Definition())
	}
	if len(parts) == 0 {
		return nil, nil
	}
	return []llm.Message{{
		Identity:   llm.Identity{Role: "system"},
		Parts:      parts,
		Volatility: llm.VolatilityHigh,
	}}, nil
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

func (m *toolMockProvider) GenerateStream(ctx context.Context, messages []llm.Message) (<-chan llm.Part, error) {
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

	// A sample security middleware that intercepts a specific secret argument
	middlewareTriggered := false
	securityMW := func(ctx context.Context, req llm.ToolRequestPart, next func(context.Context) (interface{}, error)) (interface{}, error) {
		middlewareTriggered = true
		if req.Name == "forbidden_tool" {
			return nil, fmt.Errorf("blocked by security policy")
		}
		return next(ctx)
	}

	engine := adapter.New(
		adapter.WithHistory(hist),
		adapter.WithProvider(provider),
		adapter.WithResolver(resolver),
		adapter.WithToolMiddleware(securityMW),
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

	if !middlewareTriggered {
		t.Errorf("expected middleware to be triggered")
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

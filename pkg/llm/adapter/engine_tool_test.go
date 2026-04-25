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
	ns    string
	tools []llm.Tool
}

func (r *mockResolver) Namespace() string { return r.ns }

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
	toolName  string // defaults to "mock_tool"
}

func (m *toolMockProvider) GenerateStream(ctx context.Context, messages []llm.Message, opts ...provider.GenerateOption) (<-chan llm.Part, error) {
	outCh := make(chan llm.Part)

	toolName := m.toolName
	if toolName == "" {
		toolName = "mock_tool"
	}

	go func() {
		defer close(outCh)
		m.callCount++
		
		if m.callCount == 1 {
			// First call: Request a tool
			outCh <- llm.TextPart("Thinking...")
			outCh <- llm.ToolRequestPart{
				ToolID: "call_1",
				Name:   toolName,
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
	resolver := &mockResolver{ns: "mock", tools: []llm.Tool{tool}}

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
	resolver := &mockResolver{ns: "mock", tools: []llm.Tool{tool}}

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

// TestEngine_NamespaceInBeforeToolContext verifies that the engine resolves
// the tool's namespace and injects it into the context before calling
// BeforeTool hooks. This is a targeted regression test for the bug where
// HITL policy hooks could not look up namespace-specific CEL expressions
// because the namespace was missing from the context.
func TestEngine_NamespaceInBeforeToolContext(t *testing.T) {
	// Resolver scoped to "filesystem" namespace.
	fsResolver := &mockResolver{ns: "filesystem", tools: []llm.Tool{&mockTool{names: []string{"file_list"}}}}

	var namespaceFromHook string
	hook := func(ctx context.Context, req llm.BeforeToolRequest) error {
		ns, _ := ctx.Value(llm.ContextKeyNamespace).(string)
		namespaceFromHook = ns
		if ns != "filesystem" {
			return fmt.Errorf("expected namespace 'filesystem', got '%s'", ns)
		}
		return nil
	}

	// Provider that requests file_list on the first call and answers on the second.
	prv := &toolMockProvider{toolName: "file_list"}

	hist := &MemoryHistory{}
	engine := adapter.New(
		adapter.WithWorkingMemory(hist),
		adapter.WithProvider(prv),
		adapter.WithResolver(fsResolver),
		adapter.WithBeforeTool(hook),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stream, err := engine.Stream(llm.WithSessionID(ctx, "ns-test"), llm.Message{
		Identity: llm.Identity{Role: "user"},
		Parts:    []llm.Part{llm.TextPart("list files")},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var finalText string
	for msg := range stream {
		for _, p := range msg.Parts {
			if tp, ok := p.(llm.TextPart); ok {
				finalText = string(tp)
			}
		}
	}

	if namespaceFromHook == "" {
		t.Fatalf("BeforeTool hook did not receive a namespace in context")
	}
	if namespaceFromHook != "filesystem" {
		t.Errorf("expected namespace 'filesystem' in hook context, got '%s'", namespaceFromHook)
	}

	// Because the hook didn't error, the tool should have executed and
	// the provider's second pass should have seen the result.
	if finalText != "Got result: mock_result" {
		t.Errorf("expected successful tool result echoed back, got: %v", finalText)
	}
}

// recordingHITL records every ApproveTool call so tests can assert
// which tools triggered human-in-the-loop supervision.
type recordingHITL struct {
	calls []string // tool names that reached ApproveTool
}

func (r *recordingHITL) ApproveTool(ctx context.Context, req llm.ToolRequestPart) (bool, error) {
	r.calls = append(r.calls, req.Name)
	return false, nil // always deny to keep tests deterministic
}

// multiToolMockProvider simulates an LLM that requests multiple tools
// across consecutive generate calls before producing a final text answer.
type multiToolMockProvider struct {
	callCount int
}

func (m *multiToolMockProvider) GenerateStream(ctx context.Context, messages []llm.Message, opts ...provider.GenerateOption) (<-chan llm.Part, error) {
	outCh := make(chan llm.Part)

	go func() {
		defer close(outCh)
		m.callCount++

		// Collect the most recent tool result from the message history.
		lastResult := ""
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Identity.Role == "tool" {
				for _, p := range messages[i].Parts {
					if tr, ok := p.(llm.ToolResultPart); ok {
						lastResult = fmt.Sprintf("%v", tr.Result)
						break
					}
				}
				break
			}
		}

		switch m.callCount {
		case 1:
			// First generate: request file_read.
			outCh <- llm.TextPart("I'll check the docs and then update them.")
			outCh <- llm.ToolRequestPart{
				ToolID: "call_1",
				Name:   "file_read",
				Args:   map[string]interface{}{"path": "docs/README.md"},
			}
		case 2:
			// Second generate: received file_read result, now request file_write.
			outCh <- llm.TextPart(fmt.Sprintf("Read result: %s. Now I'll write.", lastResult))
			outCh <- llm.ToolRequestPart{
				ToolID: "call_2",
				Name:   "file_write",
				Args:   map[string]interface{}{"path": "docs/README.md", "content": "# New"},
			}
		case 3:
			// Third generate: received file_write result (or error), answer.
			outCh <- llm.TextPart(fmt.Sprintf("Final state: %s", lastResult))
		}
	}()

	return outCh, nil
}

func (m *multiToolMockProvider) ListModels(ctx context.Context) ([]string, error) {
	return nil, nil
}

// TestEngine_HITLNamespacePolicy wires the real llm.NewHITLHook through the
// engine and verifies that CEL policies keyed by namespace are evaluated
// correctly. file_read should auto-approve; file_write should trigger HITL.
func TestEngine_HITLNamespacePolicy(t *testing.T) {
	// Resolver scoped to "filesystem" namespace with two tools.
	fsResolver := &mockResolver{
		ns: "filesystem",
		tools: []llm.Tool{
			&mockTool{names: []string{"file_read"}},
			&mockTool{names: []string{"file_write"}},
		},
	}

	handler := &recordingHITL{}
	// CEL policy: only file_write requires supervision.
	policies := map[string]interface{}{
		"filesystem": "tool_name == 'file_write'",
	}
	hitlHook := llm.NewHITLHook(handler, policies, false)

	prv := &multiToolMockProvider{}
	hist := &MemoryHistory{}
	engine := adapter.New(
		adapter.WithWorkingMemory(hist),
		adapter.WithProvider(prv),
		adapter.WithResolver(fsResolver),
		adapter.WithBeforeTool(hitlHook),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stream, err := engine.Stream(llm.WithSessionID(ctx, "hitl-test"), llm.Message{
		Identity: llm.Identity{Role: "user"},
		Parts:    []llm.Part{llm.TextPart("update docs")},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var texts []string
	var toolResults []llm.ToolResultPart
	for msg := range stream {
		for _, p := range msg.Parts {
			switch v := p.(type) {
			case llm.TextPart:
				texts = append(texts, string(v))
			case llm.ToolResultPart:
				toolResults = append(toolResults, v)
			}
		}
	}

	// file_read should NOT have triggered HITL (auto-approved by CEL).
	var fileReadHitl bool
	for _, name := range handler.calls {
		if name == "file_read" {
			fileReadHitl = true
		}
	}
	if fileReadHitl {
		t.Errorf("file_read should not have triggered HITL, but ApproveTool was called for it")
	}

	// file_write SHOULD have triggered HITL (CEL returns true, handler denies).
	var fileWriteHitl bool
	for _, name := range handler.calls {
		if name == "file_write" {
			fileWriteHitl = true
		}
	}
	if !fileWriteHitl {
		t.Errorf("file_write should have triggered HITL, but ApproveTool was never called for it")
	}

	// We expect exactly 2 tool results: one success (file_read) and one
	// synthetic error (file_write blocked by HITL denial).
	if len(toolResults) != 2 {
		t.Fatalf("expected 2 tool results, got %d", len(toolResults))
	}

	// file_read result should be successful.
	if toolResults[0].IsError {
		t.Errorf("expected file_read result to be successful, got error: %v", toolResults[0].Result)
	}
	if toolResults[0].Result != "mock_result" {
		t.Errorf("expected file_read result 'mock_result', got: %v", toolResults[0].Result)
	}

	// file_write result should be a synthetic HITL denial error.
	if !toolResults[1].IsError {
		t.Errorf("expected file_write result to be an error (HITL denied), got success")
	}
	if toolResults[1].Result != "user denied tool execution: please ask the user why they denied the tool and request follow-up instructions" {
		t.Errorf("expected HITL denial message, got: %v", toolResults[1].Result)
	}

	// Provider should have been called 3 times:
	// 1. initial request (file_read)
	// 2. retry with file_read result (file_write)
	// 3. retry with file_write error (final answer)
	if prv.callCount != 3 {
		t.Errorf("expected provider to be called 3 times, got %d", prv.callCount)
	}
}

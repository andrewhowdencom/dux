package librarian

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// Provider implements llm.ToolProvider for context discovery tools.
type Provider struct {
	globalMem llm.History
	tools     map[string]llm.Tool
}

// NewProvider creates a new context discovery provider with read_working_memory.
func NewProvider(globalMem llm.History) *Provider {
	p := &Provider{
		globalMem: globalMem,
		tools:     make(map[string]llm.Tool),
	}

	readMemDef := llm.ToolDefinitionPart{
		Name:        "read_working_memory",
		Description: "Read the overarching conversation history from the global Orchestrator's working memory. Use this to explicitly search for user constraints, tool executions, or context that is missing from your isolated context.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {}
		}`),
	}

	p.tools["read_working_memory"] = &genericTool{
		def: readMemDef,
		execute: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			if p.globalMem == nil {
				return nil, fmt.Errorf("global memory not available")
			}

			// Perform lookup
			sessionID, err := llm.SessionIDFromContext(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to read session ID: %w", err)
			}
			msgs, err := p.globalMem.Read(ctx, sessionID)
			if err != nil {
				return nil, fmt.Errorf("failed to read global memory: %w", err)
			}

			// Format output
			var b strings.Builder
			b.WriteString(fmt.Sprintf("Global Memory Context (%d messages):\n", len(msgs)))
			for i, msg := range msgs {
				b.WriteString(fmt.Sprintf("--- Turn %d [%s] ---\n", i+1, msg.Identity.Role))
				for _, part := range msg.Parts {
					if text, ok := part.(llm.TextPart); ok {
						b.WriteString(string(text))
						b.WriteString("\n")
					} else if tp, ok := part.(llm.ToolRequestPart); ok {
						argsRaw, _ := json.Marshal(tp.Args)
						b.WriteString(fmt.Sprintf("Tool Call: %s(%s)\n", tp.Name, string(argsRaw)))
					} else if tr, ok := part.(llm.ToolResultPart); ok {
						b.WriteString(fmt.Sprintf("Tool Result [%s]: %v\n", tr.Name, tr.Result))
					} else if ts, ok := part.(llm.TransitionSignalPart); ok {
						b.WriteString(fmt.Sprintf("Transition [%s] -> %s: %s\n", msg.Identity.Role, ts.TargetMode, ts.Message))
					}
				}
			}

			return b.String(), nil
		},
	}

	return p
}

// Namespace implements llm.ToolProvider
func (p *Provider) Namespace() string {
	return "librarian"
}

func (p *Provider) Tools() []llm.Tool {
	var tools []llm.Tool
	for _, t := range p.tools {
		tools = append(tools, t)
	}
	return tools
}

// GetTool implements llm.ToolProvider
func (p *Provider) GetTool(name string) (llm.Tool, bool) {
	t, ok := p.tools[name]
	return t, ok
}

type genericTool struct {
	def     llm.ToolDefinitionPart
	execute func(context.Context, map[string]interface{}) (interface{}, error)
}

func (g *genericTool) Name() string {
	return g.def.Name
}

func (g *genericTool) Definition() llm.ToolDefinitionPart {
	return g.def
}

func (g *genericTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return g.execute(ctx, args)
}

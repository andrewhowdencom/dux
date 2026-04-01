package llm

import "context"

// Tool defines an atomic executable unit that the Agent can invoke.
type Tool interface {
	// Name returns the specific identifier for this tool.
	Name() string
	// Definition returns the JSON Schema outlining arguments.
	Definition() ToolDefinitionPart
	// Execute performs the underlying tool logic.
	Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)
}

// ToolResolver defines how a Session or Agent discovers tools at runtime.
type ToolResolver interface {
	// Resolve queries for available tools asynchronously.
	// For MCP this entails a live fetch. For static tools, this returns instantly.
	Resolve(ctx context.Context) ([]Tool, error)
}

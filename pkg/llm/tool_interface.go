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
// For MCP this entails a live fetch. For static tools, this returns instantly.
type ToolResolver interface {
	Resolve(ctx context.Context) ([]Tool, error)
}

// ToolMiddleware provides an interceptor interface wrapping tool invocations.
// Middleware can observe arguments, enforce security, prompt for Human-In-The-Loop approval,
// or completely hijack tool execution.
type ToolMiddleware func(ctx context.Context, req ToolRequestPart, next func(ctx context.Context) (interface{}, error)) (interface{}, error)

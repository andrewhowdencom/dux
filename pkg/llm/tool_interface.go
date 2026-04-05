package llm

import (
	"context"
)

// Tool defines an atomic executable unit that the Agent can invoke.
type Tool interface {
	// Name returns the specific identifier for this tool.
	Name() string
	// Definition returns the JSON Schema outlining arguments.
	Definition() ToolDefinitionPart
	// Execute performs the underlying tool logic.
	Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)
}

// ToolProvider represents an authorized source of executable tools.
// It injects ToolDefinitions into the context but also provides the backing implementation.
type ToolProvider interface {
	Injector
	// GetTool allows the engine to resolve an execution callback when the LLM triggers a tool.
	GetTool(name string) (Tool, bool)
}

// ToolMiddleware provides an interceptor interface wrapping tool invocations.
// Middleware can observe arguments, enforce security, prompt for Human-In-The-Loop approval,
// or completely hijack tool execution.
type ToolMiddleware func(ctx context.Context, req ToolRequestPart, next func(ctx context.Context) (interface{}, error)) (interface{}, error)

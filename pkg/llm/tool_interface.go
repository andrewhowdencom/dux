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

const ContextKeyNamespace contextKey = "tool_namespace"

// ToolProvider represents an authorized source of executable tools.
type ToolProvider interface {
	Namespace() string
	// Tools returns all tools provided by this resolver.
	Tools() []Tool
	// GetTool allows the engine to resolve an execution callback when the LLM triggers a tool.
	GetTool(name string) (Tool, bool)
}

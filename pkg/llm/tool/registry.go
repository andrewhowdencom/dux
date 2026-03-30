package tool

import (
	"context"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// Registry acts as a centralized store for tool discovery and execution.
type Registry interface {
	// GetDefinitions returns tool definitions wrapped as Parts (typically ToolDefinitionPart)
	// to inject into the provider request.
	GetDefinitions() []llm.Part
	// Execute runs the underlying Go code for a given tool name.
	Execute(ctx context.Context, name string, args map[string]interface{}) (interface{}, error)
}

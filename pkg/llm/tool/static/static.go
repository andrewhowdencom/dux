package static

import (
	"context"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// Resolver returns a fixed slice of instantiated tools.
// Useful for built-in framework tools or programmatic library integrations.
type Resolver struct {
	tools []llm.Tool
}

// New returns a resolver explicitly scoped to the provided tools.
func New(tools ...llm.Tool) *Resolver {
	return &Resolver{
		tools: tools,
	}
}

// Resolve perfectly implements the llm.ToolResolver interface for static definitions.
func (r *Resolver) Resolve(ctx context.Context) ([]llm.Tool, error) {
	return r.tools, nil
}

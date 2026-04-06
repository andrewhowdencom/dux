package static

import (
	"context"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// Resolver returns a fixed slice of instantiated tools.
// Useful for built-in framework tools or programmatic library integrations.
type Resolver struct {
	namespace string
	tools     []llm.Tool
}

// New returns a resolver explicitly scoped to the provided tools.
func New(namespace string, tools ...llm.Tool) *Resolver {
	return &Resolver{
		namespace: namespace,
		tools:     tools,
	}
}

func (r *Resolver) Namespace() string {
	return r.namespace
}

// Resolve perfectly implements the llm.ToolProvider interface for static definitions.
func (r *Resolver) Inject(ctx context.Context, q llm.InjectQuery) ([]llm.Message, error) {
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

func (r *Resolver) GetTool(name string) (llm.Tool, bool) {
	for _, t := range r.tools {
		if t.Name() == name {
			return t, true
		}
	}
	return nil, false
}

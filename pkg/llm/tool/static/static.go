package static

import (
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

func (r *Resolver) Tools() []llm.Tool {
	return r.tools
}

func (r *Resolver) GetTool(name string) (llm.Tool, bool) {
	for _, t := range r.tools {
		if t.Name() == name {
			return t, true
		}
	}
	return nil, false
}

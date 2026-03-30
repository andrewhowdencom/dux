package factory

import (
	"fmt"

	"github.com/andrewhowdencom/dux/pkg/llm/provider"
	"github.com/andrewhowdencom/dux/pkg/llm/provider/ollama"
	"github.com/andrewhowdencom/dux/pkg/llm/provider/openai"
	"github.com/andrewhowdencom/dux/pkg/llm/provider/static"
)

// New builds a new LLM provider recursively based on its type definition and config envelope.
func New(cfg provider.InstanceConfig) (provider.Provider, error) {
	switch cfg.Type {
	case "static":
		return static.New(cfg.Config)
	case "ollama":
		return ollama.New(cfg.Config)
	case "openai", "litellm":
		return openai.New(cfg.Config)
	default:
		return nil, fmt.Errorf("unknown or unsupported provider type: %q (id: %q)", cfg.Type, cfg.ID)
	}
}

package provider

import (
	"context"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// Provider represents a generic LLM client (e.g., OpenAI, Vertex).
type Provider interface {
	// GenerateStream yields complete Part operations (e.g., fully formed text
	// parts or complete tool requests) from the LLM based on the conversation messages.
	GenerateStream(ctx context.Context, messages []llm.Message) (<-chan llm.Part, error)

	// ListModels returns a list of available models for this provider
	ListModels(ctx context.Context) ([]string, error)
}

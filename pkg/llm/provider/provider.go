package provider

import (
	"context"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// ModelLister defines standard operations to retrieve available generation models.
type ModelLister interface {
	// ListModels returns a list of available models for this provider
	ListModels(ctx context.Context) ([]string, error)
}

// ChatGenerator defines text generation and tool invocation mechanics for standard chat flows.
type ChatGenerator interface {
	// GenerateStream yields complete Part operations (e.g., fully formed text
	// parts or complete tool requests) from the LLM based on the conversation messages.
	GenerateStream(ctx context.Context, messages []llm.Message) (<-chan llm.Part, error)
}

// Embedder identifies providers capable of generating high-density vector representations.
type Embedder interface {
	GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
}

// Provider represents a generic LLM client (e.g., OpenAI, Vertex) combining multiple modality interfaces.
// It is intended for backwards compatibility; prefer the specific segregated interfaces where possible.
type Provider interface {
	ChatGenerator
	ModelLister
}

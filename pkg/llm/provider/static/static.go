package static

import (
	"context"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/llm/provider"
)

// Provider implements the LLM Provider stream by yielding hardcoded messages.
type Provider struct {
	responses []llm.Message
}

// New creates a static Provider.
func New(config map[string]interface{}) (provider.Provider, error) {
	// Future config maps can decode here! For now, generic fallback:
	responses := []llm.Message{
		{
			SessionID: "static",
			Identity:  llm.Identity{Role: "assistant"},
			Parts:     []llm.Part{llm.TextPart("I am a static LLM. I'm operating within the generic Provider pipeline!")},
		},
	}
	return &Provider{responses: responses}, nil
}

// GenerateStream immediately yields all specified message parts directly into the output stream.
func (s *Provider) GenerateStream(ctx context.Context, messages []llm.Message) (<-chan llm.Part, error) {
	out := make(chan llm.Part)

	go func() {
		defer close(out)
		for _, msg := range s.responses {
			for _, part := range msg.Parts {
				select {
				case <-ctx.Done():
					return
				case out <- part:
				}
			}
		}
	}()

	return out, nil
}

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

func New(config map[string]interface{}) (provider.Provider, error) {
	responseText := "I am a static LLM. I'm operating within the generic Provider pipeline!"
	if text, ok := config["text"].(string); ok {
		responseText = text
	}

	responses := []llm.Message{
		{
			SessionID: "static",
			Identity:  llm.Identity{Role: "assistant"},
			Parts:     []llm.Part{llm.TextPart(responseText)},
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

// ListModels returns a static list of models for the Static provider.
func (s *Provider) ListModels(ctx context.Context) ([]string, error) {
	return []string{"static-model"}, nil
}

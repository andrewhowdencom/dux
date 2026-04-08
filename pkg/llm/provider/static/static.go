package static

import (
	"context"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/llm/provider"
)

// Option allows functional configuration of the Static Provider.
type Option func(*Provider)

// WithText sets the static message text yielded by the provider.
func WithText(text string) Option {
	return func(p *Provider) {
		p.text = text
	}
}

// Provider implements the LLM Provider stream by yielding hardcoded messages.
type Provider struct {
	text      string
	responses []llm.Message
}

func New(opts ...Option) (*Provider, error) {
	p := &Provider{
		text: "I am a static LLM. I'm operating within the generic Provider pipeline!",
	}

	for _, opt := range opts {
		opt(p)
	}

	p.responses = []llm.Message{
		{
			SessionID: "static",
			Identity:  llm.Identity{Role: "assistant"},
			Parts:     []llm.Part{llm.TextPart(p.text)},
		},
	}
	return p, nil
}

func (s *Provider) Capabilities() provider.Capabilities {
	return provider.Capabilities{
		SupportsSystemPrompt:     false,
		SupportsToolCalling:      false,
		SupportsImages:           false,
		SupportsStructuredOutput: false,
		MaxContextWindow:         0,
	}
}

// GenerateStream immediately yields all specified message parts directly into the output stream.
func (s *Provider) GenerateStream(ctx context.Context, messages []llm.Message, opts ...provider.GenerateOption) (<-chan llm.Part, error) {
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

// GenerateEmbeddings returns a mock set of empty vectors.
func (s *Provider) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
    result := make([][]float32, len(texts))
    for i := range texts {
        result[i] = []float32{0.0}
    }
    return result, nil
}

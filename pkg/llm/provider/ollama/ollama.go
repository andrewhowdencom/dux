package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/google/uuid"
	api "github.com/ollama/ollama/api"
)

// Option allows functional configuration of the Ollama Provider.
type Option func(*Provider)

// WithAddress overrides the default Ollama API address.
func WithAddress(address string) Option {
	return func(p *Provider) {
		p.address = address
	}
}

// WithModel sets the target chat completion model.
func WithModel(model string) Option {
	return func(p *Provider) {
		p.model = model
	}
}

// WithNumCtx explicitly sets the context window length for the model.
func WithNumCtx(ctx int) Option {
	return func(p *Provider) {
		if p.options == nil {
			p.options = make(map[string]any)
		}
		p.options["num_ctx"] = ctx
	}
}

type Provider struct {
	client  *api.Client
	model   string
	address string
	options map[string]any
}

// New constructs an Ollama compatible provider utilizing functional options.
func New(opts ...Option) (*Provider, error) {
	p := &Provider{
		address: "http://localhost:11434",
		model:   "llama3",
	}

	for _, opt := range opts {
		opt(p)
	}

	u, err := url.Parse(p.address)
	if err != nil {
		return nil, fmt.Errorf("invalid address for ollama: %w", err)
	}

	p.client = api.NewClient(u, http.DefaultClient)
	return p, nil
}

func (o *Provider) GenerateStream(ctx context.Context, messages []llm.Message) (<-chan llm.Part, error) {
	out := make(chan llm.Part)

	reqMessages, reqTools := buildOllamaRequest(messages)

	go func() {
		defer close(out)

		t := true
		req := &api.ChatRequest{
			Model:    o.model,
			Messages: reqMessages,
			Tools:    reqTools,
			Stream:   &t,
			Options:  o.options,
		}

		err := o.client.Chat(ctx, req, func(resp api.ChatResponse) error {
			if resp.Message.Content != "" {
				out <- llm.TextPart(resp.Message.Content)
			}
			if resp.Message.Thinking != "" {
				out <- llm.ReasoningPart(resp.Message.Thinking)
			}
			for _, tc := range resp.Message.ToolCalls {
				out <- llm.ToolRequestPart{
					ToolID: uuid.NewString(),
					Name: tc.Function.Name,
					Args: tc.Function.Arguments.ToMap(),
				}
			}
			if resp.Done {
				out <- llm.TelemetryPart{
					InputTokens:  resp.PromptEvalCount,
					OutputTokens: resp.EvalCount,
					Duration:     resp.TotalDuration,
				}
			}
			return nil
		})

		if err != nil {
			out <- llm.TextPart(fmt.Sprintf("\n[Ollama Provider Error: %v]", err))
		}
	}()

	return out, nil
}

// GenerateEmbeddings implements the Embedder interface for Ollama.
func (o *Provider) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	req := &api.EmbedRequest{
		Model:   o.model,
		Input:   texts,
		Options: o.options,
	}

	resp, err := o.client.Embed(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ollama embeddings: %w", err)
	}

	var embeddings [][]float32
	// Ensure we don't return null if Embeddings is empty but not nil
	if resp.Embeddings != nil {
		embeddings = make([][]float32, len(resp.Embeddings))
		copy(embeddings, resp.Embeddings)
	}

	return embeddings, nil
}

// ListModels returns a list of available models from the Ollama instance.
func (o *Provider) ListModels(ctx context.Context) ([]string, error) {
	resp, err := o.client.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list ollama models: %w", err)
	}

	var models []string
	for _, m := range resp.Models {
		models = append(models, m.Name)
	}
	return models, nil
}

// buildOllamaRequest extracts abstract llm.Message structures into Ollama's specific api layout.
func buildOllamaRequest(messages []llm.Message) ([]api.Message, api.Tools) {
	var reqMessages []api.Message
	var reqTools api.Tools

	for _, m := range messages {
		var textContent string
		var thinkingContent string
		var toolCalls []api.ToolCall

		for _, p := range m.Parts {
			switch part := p.(type) {
			case llm.TextPart:
				textContent += string(part)
			case llm.ReasoningPart:
				thinkingContent += string(part)
			case llm.ToolRequestPart:
				args := api.NewToolCallFunctionArguments()
				for k, v := range part.Args {
					args.Set(k, v)
				}

				toolCalls = append(toolCalls, api.ToolCall{
					Function: api.ToolCallFunction{
						Name:      part.Name,
						Arguments: args,
					},
				})
			case llm.ToolDefinitionPart:
				var params api.ToolFunctionParameters
				if len(part.Parameters) > 0 {
					_ = json.Unmarshal(part.Parameters, &params)
				}
				reqTools = append(reqTools, api.Tool{
					Type: "function",
					Function: api.ToolFunction{
						Name:        part.Name,
						Description: part.Description,
						Parameters:  params,
					},
				})
			case llm.ToolResultPart:
				b, _ := json.Marshal(part.Result)
				textContent += string(b)
			}
		}

		if textContent != "" || thinkingContent != "" || len(toolCalls) > 0 {
			reqMessages = append(reqMessages, api.Message{
				Role:      m.Identity.Role,
				Content:   textContent,
				Thinking:  thinkingContent,
				ToolCalls: toolCalls,
			})
		}
	}
	return reqMessages, reqTools
}

package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"net/url"
	"time"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/llm/provider"
	openai "github.com/sashabaranov/go-openai"
)

type Option func(*Provider)

// WithBaseURL overrides the default OpenAI API base URL.
func WithBaseURL(url string) Option {
	return func(p *Provider) {
		p.baseURL = url
	}
}

// WithModel sets the target chat completion model.
func WithModel(model string) Option {
	return func(p *Provider) {
		p.model = model
	}
}

type Provider struct {
	client  *openai.Client
	model   string
	baseURL string
}

// New constructs an OpenAI compatible provider utilizing functional options.
func New(apiKey string, opts ...Option) (*Provider, error) {
	p := &Provider{
		model: openai.GPT4o,
	}

	for _, opt := range opts {
		opt(p)
	}

	clientConfig := openai.DefaultConfig(apiKey)
	if p.baseURL != "" {
		_, err := url.ParseRequestURI(p.baseURL)
		if err != nil {
			return nil, fmt.Errorf("invalid base_url: %w", err)
		}
		clientConfig.BaseURL = p.baseURL
	}

	p.client = openai.NewClientWithConfig(clientConfig)

	return p, nil
}

func (p *Provider) Capabilities() provider.Capabilities {
	return provider.Capabilities{
		SupportsSystemPrompt:     true,
		SupportsToolCalling:      true,
		SupportsImages:           true,
		SupportsStructuredOutput: true,
		MaxContextWindow:         0,
	}
}

func (p *Provider) GenerateStream(ctx context.Context, messages []llm.Message, opts ...provider.GenerateOption) (<-chan llm.Part, error) {
	out := make(chan llm.Part)

	config := &provider.GenerateConfig{}
	for _, opt := range opts {
		opt(config)
	}

	reqMessages, reqTools := buildOpenAIRequest(messages)

	go func() {
		defer close(out)

		req := openai.ChatCompletionRequest{
			Model:    p.model,
			Messages: reqMessages,
			Stream:   true,
			StreamOptions: &openai.StreamOptions{
				IncludeUsage: true,
			},
		}

		if config.Temperature != nil {
			req.Temperature = *config.Temperature
		}
		
		if config.MaxTokens != nil {
			req.MaxTokens = *config.MaxTokens
		}

		if len(config.JSONSchema) > 0 {
			req.ResponseFormat = &openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeJSONObject,
			}
		}

		if len(reqTools) > 0 {
			req.Tools = reqTools
		}

		startTime := time.Now()
		stream, err := p.client.CreateChatCompletionStream(ctx, req)
		if err != nil {
			var apiErr *openai.APIError
			if errors.As(err, &apiErr) {
				if apiErr.HTTPStatusCode == 429 {
					out <- llm.TextPart(fmt.Sprintf("\n[OpenAI Provider Error: %v - %v]", provider.ErrRateLimitExceeded, apiErr.Message))
				} else {
					out <- llm.TextPart(fmt.Sprintf("\n[OpenAI Provider Error: %v - %v]", provider.ErrProviderUnavailable, apiErr.Message))
				}
			} else {
				out <- llm.TextPart(fmt.Sprintf("\n[OpenAI Provider Error: %v - %v]", provider.ErrProviderUnavailable, err))
			}
			return
		}
		defer func() {
			_ = stream.Close()
		}()

		// Tool calls can arrive in chunks. We need to accumulate them.
		type toolCallBuilder struct {
			ID        string
			Name      string
			Arguments string
		}
		toolCallBuilders := make(map[int]*toolCallBuilder)

		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				// Flush accumulated tool calls
				for _, b := range toolCallBuilders {
					var args map[string]interface{}
					if b.Arguments != "" {
						_ = json.Unmarshal([]byte(b.Arguments), &args)
					}
					out <- llm.ToolRequestPart{
						ToolID: b.ID,
						Name:   b.Name,
						Args:   args,
					}
				}
				return
			}
			if err != nil {
				out <- llm.TextPart(fmt.Sprintf("\n[OpenAI Provider Stream Error: %v - %v]", provider.ErrProviderUnavailable, err))
				return
			}

			if len(response.Choices) > 0 {
				choice := response.Choices[0]

				if choice.Delta.Content != "" {
					out <- llm.TextPart(choice.Delta.Content)
				}
				if choice.Delta.ReasoningContent != "" {
					out <- llm.ReasoningPart(choice.Delta.ReasoningContent)
				}

				for _, tc := range choice.Delta.ToolCalls {
					// We need to accumulate
					if tc.Index != nil {
						idx := *tc.Index
						if _, exists := toolCallBuilders[idx]; !exists {
							toolCallBuilders[idx] = &toolCallBuilder{}
						}
						b := toolCallBuilders[idx]
						if tc.ID != "" {
							b.ID = tc.ID
						}
						if tc.Function.Name != "" {
							b.Name += tc.Function.Name
						}
						if tc.Function.Arguments != "" {
							b.Arguments += tc.Function.Arguments
						}
					}
				}
			}

			if response.Usage != nil {
				reasoningTokens := 0
				if response.Usage.CompletionTokensDetails != nil {
					reasoningTokens = response.Usage.CompletionTokensDetails.ReasoningTokens
				}
				out <- llm.TelemetryPart{
					InputTokens:     response.Usage.PromptTokens,
					OutputTokens:    response.Usage.CompletionTokens,
					ReasoningTokens: reasoningTokens,
					Duration:        time.Since(startTime),
				}
			}
		}
	}()

	return out, nil
}

// GenerateEmbeddings implements the Embedder interface for OpenAI.
func (p *Provider) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	req := openai.EmbeddingRequest{
		Input: texts,
		Model: openai.EmbeddingModel(p.model),
	}

	resp, err := p.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate openai embeddings: %w", err)
	}

	embeddings := make([][]float32, len(resp.Data))
	for i, d := range resp.Data {
		embeddings[i] = d.Embedding
	}

	return embeddings, nil
}

// ListModels returns a list of available models from the OpenAI-compatible endpoint.
func (p *Provider) ListModels(ctx context.Context) ([]string, error) {
	resp, err := p.client.ListModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list openai models: %w", err)
	}

	var models []string
	for _, m := range resp.Models {
		models = append(models, m.ID)
	}
	return models, nil
}

// buildOpenAIRequest extracts abstract llm.Messages into OpenAI's specific api layout.
func buildOpenAIRequest(messages []llm.Message) ([]openai.ChatCompletionMessage, []openai.Tool) {
	var reqMessages []openai.ChatCompletionMessage
	var reqTools []openai.Tool

	for _, m := range messages {
		var textContent string
		var thinkingContent string
		var toolCalls []openai.ToolCall
		var toolResults []llm.ToolResultPart

		// Translate dux roles to openai roles
		role := m.Identity.Role
		switch role {
		case "tool":
			role = openai.ChatMessageRoleTool
		case "model", "assistant":
			role = openai.ChatMessageRoleAssistant
		case "user":
			role = openai.ChatMessageRoleUser
		case "system":
			role = openai.ChatMessageRoleSystem
		}

		for _, p := range m.Parts {
			switch part := p.(type) {
			case llm.TextPart:
				textContent += string(part)
			case llm.ReasoningPart:
				thinkingContent += string(part)
			case llm.ToolRequestPart:
				argsJSON, _ := json.Marshal(part.Args)
				toolCalls = append(toolCalls, openai.ToolCall{
					ID:   part.ToolID,
					Type: openai.ToolTypeFunction,
					Function: openai.FunctionCall{
						Name:      part.Name,
						Arguments: string(argsJSON),
					},
				})
			case llm.ToolResultPart:
				toolResults = append(toolResults, part)
			case llm.ToolDefinitionPart:
				// Map ToolDefinitionPart to reqTools
				var parameters interface{}
				if len(part.Parameters) > 0 {
					_ = json.Unmarshal(part.Parameters, &parameters)
				} else {
					// Fallback to empty object schema
					parameters = map[string]interface{}{
						"type":       "object",
						"properties": map[string]interface{}{},
					}
				}
				reqTools = append(reqTools, openai.Tool{
					Type: openai.ToolTypeFunction,
					Function: &openai.FunctionDefinition{
						Name:        part.Name,
						Description: part.Description,
						Parameters:  parameters,
					},
				})
			}
		}

		if len(toolResults) > 0 {
			for _, tr := range toolResults {
				var content string
				if s, ok := tr.Result.(string); ok {
					content = s
				} else {
					b, _ := json.Marshal(tr.Result)
					content = string(b)
				}
				
				if content == "" || content == "null" {
					content = "{}"
				}

				reqMessages = append(reqMessages, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					Content:    content,
					ToolCallID: tr.ToolID,
				})
			}

			if textContent != "" || thinkingContent != "" {
				reqMessages = append(reqMessages, openai.ChatCompletionMessage{
					Role:             openai.ChatMessageRoleUser,
					Content:          textContent,
					ReasoningContent: thinkingContent,
				})
			}
		} else if textContent != "" || thinkingContent != "" || len(toolCalls) > 0 {
			msg := openai.ChatCompletionMessage{
				Role:             role,
				Content:          textContent,
				ReasoningContent: thinkingContent,
				ToolCalls:        toolCalls,
			}
			reqMessages = append(reqMessages, msg)
		}
	}
	return reqMessages, reqTools
}

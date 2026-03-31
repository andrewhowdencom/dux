package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"net/url"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/llm/provider"
	"github.com/mitchellh/mapstructure"
	openai "github.com/sashabaranov/go-openai"
)

type Config struct {
	BaseURL string `mapstructure:"base_url"`
	APIKey  string `mapstructure:"api_key"`
	Model   string `mapstructure:"model"`
}

type Provider struct {
	client *openai.Client
	model  string
}

func New(rawConfig map[string]interface{}) (provider.Provider, error) {
	var cfg Config
	if err := mapstructure.Decode(rawConfig, &cfg); err != nil {
		return nil, fmt.Errorf("failed to decode openai config: %w", err)
	}

	clientConfig := openai.DefaultConfig(cfg.APIKey)
	if cfg.BaseURL != "" {
		_, err := url.ParseRequestURI(cfg.BaseURL)
		if err != nil {
			return nil, fmt.Errorf("invalid base_url: %w", err)
		}
		clientConfig.BaseURL = cfg.BaseURL
	}

	client := openai.NewClientWithConfig(clientConfig)

	model := cfg.Model
	if model == "" {
		model = openai.GPT4o
	}

	return &Provider{
		client: client,
		model:  model,
	}, nil
}

func (p *Provider) GenerateStream(ctx context.Context, messages []llm.Message) (<-chan llm.Part, error) {
	out := make(chan llm.Part)

	var reqMessages []openai.ChatCompletionMessage
	var reqTools []openai.Tool

	for _, m := range messages {
		var textContent string
		var toolCalls []openai.ToolCall

		// Translate dux roles to openai roles
		role := m.Identity.Role
		if role == "tool" {
			role = openai.ChatMessageRoleTool
		} else if role == "model" || role == "assistant" {
			role = openai.ChatMessageRoleAssistant
		} else if role == "user" {
			role = openai.ChatMessageRoleUser
		} else if role == "system" {
			role = openai.ChatMessageRoleSystem
		}

		for _, p := range m.Parts {
			switch part := p.(type) {
			case llm.TextPart:
				textContent += string(part)
			case llm.ToolRequestPart:
				argsJSON, _ := json.Marshal(part.Args)
				toolCalls = append(toolCalls, openai.ToolCall{
					Type: openai.ToolTypeFunction,
					Function: openai.FunctionCall{
						Name:      part.Name,
						Arguments: string(argsJSON),
					},
				})
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

		if textContent != "" || len(toolCalls) > 0 {
			msg := openai.ChatCompletionMessage{
				Role:      role,
				Content:   textContent,
				ToolCalls: toolCalls,
			}
			reqMessages = append(reqMessages, msg)
		}
	}

	go func() {
		defer close(out)

		req := openai.ChatCompletionRequest{
			Model:    p.model,
			Messages: reqMessages,
			Stream:   true,
		}

		if len(reqTools) > 0 {
			req.Tools = reqTools
		}

		stream, err := p.client.CreateChatCompletionStream(ctx, req)
		if err != nil {
			var apiErr *openai.APIError
			if errors.As(err, &apiErr) {
				out <- llm.TextPart(fmt.Sprintf("\n[OpenAI Provider Error: %v]", apiErr.Message))
			} else {
				out <- llm.TextPart(fmt.Sprintf("\n[OpenAI Provider Error: %v]", err))
			}
			return
		}
		defer stream.Close()

		// Tool calls can arrive in chunks. We need to accumulate them.
		type toolCallBuilder struct {
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
						Name: b.Name,
						Args: args,
					}
				}
				return
			}
			if err != nil {
				out <- llm.TextPart(fmt.Sprintf("\n[OpenAI Provider Stream Error: %v]", err))
				return
			}

			if len(response.Choices) > 0 {
				choice := response.Choices[0]

				if choice.Delta.Content != "" {
					out <- llm.TextPart(choice.Delta.Content)
				}

				for _, tc := range choice.Delta.ToolCalls {
					// We need to accumulate
					if tc.Index != nil {
						idx := *tc.Index
						if _, exists := toolCallBuilders[idx]; !exists {
							toolCallBuilders[idx] = &toolCallBuilder{}
						}
						b := toolCallBuilders[idx]
						if tc.Function.Name != "" {
							b.Name += tc.Function.Name
						}
						if tc.Function.Arguments != "" {
							b.Arguments += tc.Function.Arguments
						}
					}
				}
			}
		}
	}()

	return out, nil
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

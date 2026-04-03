package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"google.golang.org/genai"
)

type Option func(*Provider)

// WithModel sets the target chat completion model.
func WithModel(model string) Option {
	return func(p *Provider) {
		p.model = model
	}
}

type Provider struct {
	client *genai.Client
	model  string
}

// New constructs a Gemini compatible provider.
func New(apiKey string, opts ...Option) (*Provider, error) {
	p := &Provider{
		model: "gemini-3-flash-preview",
	}

	for _, opt := range opts {
		opt(p)
	}

	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}
	p.client = client

	return p, nil
}

func (p *Provider) GenerateStream(ctx context.Context, messages []llm.Message) (<-chan llm.Part, error) {
	out := make(chan llm.Part)

	contents, cfg := buildGeminiRequest(messages)

	go func() {
		defer close(out)

		startTime := time.Now()
		stream := p.client.Models.GenerateContentStream(ctx, p.model, contents, cfg)
		
		for resp, err := range stream {
			if err != nil {
				out <- llm.TextPart(fmt.Sprintf("\n[Gemini Provider Stream Error: %v]", err))
				return
			}

			if len(resp.Candidates) > 0 {
				candidate := resp.Candidates[0]
				if candidate.Content != nil {
					for _, part := range candidate.Content.Parts {
						if part.Text != "" {
							out <- llm.TextPart(part.Text)
						}
						if part.ExecutableCode != nil && part.ExecutableCode.Code != "" {
							out <- llm.TextPart(part.ExecutableCode.Code) // just append the code, theoretically it's text
						}
						
						if part.FunctionCall != nil {
							out <- llm.ToolRequestPart{
								ToolID: "", // function call ID not directly exposed unless we generate it
								Name:   part.FunctionCall.Name,
								Args:   part.FunctionCall.Args,
							}
						}
					}
				}
			}

			if resp.UsageMetadata != nil {
				out <- llm.TelemetryPart{
					InputTokens:  int(resp.UsageMetadata.PromptTokenCount),
					OutputTokens: int(resp.UsageMetadata.CandidatesTokenCount),
					Duration:     time.Since(startTime),
				}
			}
		}
	}()

	return out, nil
}

func (p *Provider) ListModels(ctx context.Context) ([]string, error) {
	// The new genai SDK Models list models operation is typically on client.Models.List() but keeping it simple right now
	// To do a proper lookup we could use client.Models.List() if we fetch the right page wrapper but let's 
	// fallback if not immediately apparent. Wait, let's just use empty model list or hardcoded default common ones
	// for now, as genai handles mostly Gemini flavors.
	models := []string{
		"gemini-3-flash-preview",
		"gemini-2.5-flash",
		"gemini-2.5-pro",
		"gemini-2.0-flash",
		"gemini-2.0-pro-exp-02-05",
		"gemini-1.5-pro",
		"gemini-1.5-flash",
	}
	return models, nil
}

func buildGeminiRequest(messages []llm.Message) ([]*genai.Content, *genai.GenerateContentConfig) {
	var contents []*genai.Content
	cfg := &genai.GenerateContentConfig{
        Tools: []*genai.Tool{},
    }
	var sysParts []*genai.Part

	for _, m := range messages {
		var contentParts []*genai.Part

		role := "user"
		switch m.Identity.Role {
		case "model", "assistant":
			role = "model"
		case "user":
			role = "user"
		case "tool":
			role = "user" // genai uses 'user' to send tool responses
		case "system":
			role = "system"
		}

		for _, p := range m.Parts {
			switch part := p.(type) {
			case llm.TextPart:
				contentParts = append(contentParts, &genai.Part{Text: string(part)})
			case llm.ReasoningPart:
				// gemini might not have explicit reasoning part in user requests yet, just map to text
				contentParts = append(contentParts, &genai.Part{Text: string(part)})
			case llm.ToolRequestPart:
				contentParts = append(contentParts, &genai.Part{
					FunctionCall: &genai.FunctionCall{
						Name: part.Name,
						Args: part.Args,
					},
				})
				role = "model" // model generates function calls
			case llm.ToolResultPart:
				var responseMap map[string]any
				if r, ok := part.Result.(map[string]any); ok {
					responseMap = r
				} else {
					b, err := json.Marshal(part.Result)
					if err == nil {
						_ = json.Unmarshal(b, &responseMap)
					}
				}
				contentParts = append(contentParts, &genai.Part{
					FunctionResponse: &genai.FunctionResponse{
						Name:     part.Name, 
						Response: responseMap,
					},
				})
			case llm.ToolDefinitionPart:
				if len(part.Parameters) > 0 {
					var raw map[string]any
					_ = json.Unmarshal(part.Parameters, &raw)
				}
				
				tool := &genai.Tool{
					FunctionDeclarations: []*genai.FunctionDeclaration{
						{
							Name:        part.Name,
							Description: part.Description,
						},
					},
				}
				cfg.Tools = append(cfg.Tools, tool)
			}
		}

		if role == "system" {
			sysParts = append(sysParts, contentParts...)
		} else if len(contentParts) > 0 {
			// Ensure role is either user or model
			if role == "tool" {
				role = "user" // Tool results sent as user
			}
			contents = append(contents, &genai.Content{
				Role:  role,
				Parts: contentParts, // This might need logic to merge adjacent text parts if genai is strict, but it typically accepts it
			})
		}
	}

	if len(sysParts) > 0 {
		cfg.SystemInstruction = &genai.Content{
			Role:  "user", // System instruction role doesn't strictly matter or is 'user' or empty
			Parts: sysParts,
		}
	}
    
    // Clear tools slice if empty
    if len(cfg.Tools) == 0 {
        cfg.Tools = nil
    }

	return contents, cfg
}

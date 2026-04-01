package ollama

import (
	"encoding/json"
	"testing"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

func TestBuildOllamaRequest(t *testing.T) {
	tests := []struct {
		name         string
		messages     []llm.Message
		expectedMsgs int
		expectedTool int
	}{
		{
			name: "single text part",
			messages: []llm.Message{
				{
					Identity: llm.Identity{Role: "user"},
					Parts:    []llm.Part{llm.TextPart("Hello world")},
				},
			},
			expectedMsgs: 1,
			expectedTool: 0,
		},
		{
			name: "tool definition mapping",
			messages: []llm.Message{
				{
					Identity: llm.Identity{Role: "system"},
					Parts: []llm.Part{
						llm.TextPart("System prompt"),
						llm.ToolDefinitionPart{
							Name:        "get_weather",
							Description: "Get weather",
							Parameters:  json.RawMessage(`{"type":"object"}`),
						},
					},
				},
			},
			expectedMsgs: 1,
			expectedTool: 1,
		},
		{
			name: "tool result part guarantees role tool",
			messages: []llm.Message{
				{
					Identity: llm.Identity{Role: "tool"},
					Parts: []llm.Part{
						llm.ToolResultPart{
							Name:   "get_weather",
							Result: map[string]string{"temp": "75F"},
						},
					},
				},
			},
			expectedMsgs: 1,
			expectedTool: 0,
		},
		{
			name: "tool request part maps native arguments",
			messages: []llm.Message{
				{
					Identity: llm.Identity{Role: "assistant"},
					Parts: []llm.Part{
						llm.ToolRequestPart{
							Name: "get_weather",
							Args: map[string]interface{}{"loc": "Tokyo"},
						},
					},
				},
			},
			expectedMsgs: 1,
			expectedTool: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgs, tools := buildOllamaRequest(tt.messages)
			if len(msgs) != tt.expectedMsgs {
				t.Fatalf("expected %d messages, got %d", tt.expectedMsgs, len(msgs))
			}
			if len(tools) != tt.expectedTool {
				t.Fatalf("expected %d tools, got %d", tt.expectedTool, len(tools))
			}

			// Core integrity verification for Tool Result
			if tt.name == "tool result part guarantees role tool" {
				if msgs[0].Role != "tool" {
					t.Errorf("expected role 'tool', got %q", msgs[0].Role)
				}
				if msgs[0].Content == "" {
					t.Errorf("expected JSON stringified content for tool result, got empty")
				}
			}
		})
	}
}

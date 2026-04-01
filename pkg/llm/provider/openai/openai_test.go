package openai

import (
	"encoding/json"
	"testing"

	"github.com/andrewhowdencom/dux/pkg/llm"
	openai "github.com/sashabaranov/go-openai"
)

func TestBuildOpenAIRequest(t *testing.T) {
	tests := []struct {
		name         string
		messages     []llm.Message
		expectedMsgs int
		expectedTool int
	}{
		{
			name: "single text part resolves to simple chat message",
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
			name: "tool definition maps to functional payload schema",
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
			name: "tool result strictly maps tool call identifier back",
			messages: []llm.Message{
				{
					Identity: llm.Identity{Role: "tool"},
					Parts: []llm.Part{
						llm.ToolResultPart{
							ToolID: "call_abc123",
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
			name: "tool request part binds out correct id array",
			messages: []llm.Message{
				{
					Identity: llm.Identity{Role: "assistant"},
					Parts: []llm.Part{
						llm.ToolRequestPart{
							ToolID: "call_req_456",
							Name:   "get_weather",
							Args:   map[string]interface{}{"loc": "Tokyo"},
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
			msgs, tools := buildOpenAIRequest(tt.messages)
			if len(msgs) != tt.expectedMsgs {
				t.Fatalf("expected %d messages, got %d", tt.expectedMsgs, len(msgs))
			}
			if len(tools) != tt.expectedTool {
				t.Fatalf("expected %d tools, got %d", tt.expectedTool, len(tools))
			}

			// Sub-property test for OpenAI strict requirements
			if tt.name == "tool result strictly maps tool call identifier back" {
				if msgs[0].Role != openai.ChatMessageRoleTool {
					t.Errorf("expected role 'tool', got %q", msgs[0].Role)
				}
				if msgs[0].ToolCallID != "call_abc123" {
					t.Errorf("expected ToolCallID to traverse perfectly back, got %q", msgs[0].ToolCallID)
				}
			}

			if tt.name == "tool request part binds out correct id array" {
				if len(msgs[0].ToolCalls) == 0 {
					t.Fatalf("Expected toolcalls to be populated")
				}
				if msgs[0].ToolCalls[0].ID != "call_req_456" {
					t.Errorf("Expected outbound ToolCall array ID bindings to stay intact")
				}
			}
		})
	}
}

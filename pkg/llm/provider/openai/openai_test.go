package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

func TestOpenAINewConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid config",
			config: map[string]interface{}{
				"api_key": "test_key",
				"model":   "gpt-4o",
			},
			wantErr: false,
		},
		{
			name: "custom base url",
			config: map[string]interface{}{
				"base_url": "http://localhost:8080/v1",
			},
			wantErr: false,
		},
		{
			name: "invalid base url",
			config: map[string]interface{}{
				"base_url": "::invalid::url",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOpenAIGenerateStream(t *testing.T) {
	// Start a local HTTP server that mocks the OpenAI API stream response.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Send mock chunks
		chunk1 := `{"id":"chatcmpl-123","choices":[{"delta":{"content":"Hello "}}]}`
		chunk2 := `{"id":"chatcmpl-123","choices":[{"delta":{"content":"World!"}}]}`

		_, _ = w.Write([]byte("data: " + chunk1 + "\n\n"))
		_, _ = w.Write([]byte("data: " + chunk2 + "\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	// Initialize the provider with the mock server URL
	provider, err := New(map[string]interface{}{
		"api_key":  "test_key",
		"base_url": server.URL,
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx := context.Background()
	messages := []llm.Message{
		{
			Identity: llm.Identity{Role: "user"},
			Parts: []llm.Part{
				llm.TextPart("Say hello"),
			},
		},
	}

	stream, err := provider.GenerateStream(ctx, messages)
	if err != nil {
		t.Fatalf("Failed to generate stream: %v", err)
	}

	var results []string
	for part := range stream {
		if textPart, ok := part.(llm.TextPart); ok {
			results = append(results, string(textPart))
		}
	}

	expected := "Hello World!"
	actual := strings.Join(results, "")
	if actual != expected {
		t.Errorf("Expected output %q, got %q", expected, actual)
	}
}

func TestOpenAIToolCallParsing(t *testing.T) {
	// Start a local HTTP server that mocks the OpenAI API stream response.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")

		// Send a mock chunk containing a tool call
		chunk1 := `{"id":"chatcmpl-123","choices":[{"delta":{"tool_calls":[{"index":0,"function":{"name":"get_weather"}}]}}]}`
		chunk2 := `{"id":"chatcmpl-123","choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"location\":\""}}]}}]}`
		chunk3 := `{"id":"chatcmpl-123","choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"Tokyo\"}"}}]}}]}`

		_, _ = w.Write([]byte("data: " + chunk1 + "\n\n"))
		_, _ = w.Write([]byte("data: " + chunk2 + "\n\n"))
		_, _ = w.Write([]byte("data: " + chunk3 + "\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	// Initialize the provider with the mock server URL
	provider, err := New(map[string]interface{}{
		"api_key":  "test_key",
		"base_url": server.URL,
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx := context.Background()
	params, _ := json.Marshal(map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"location": map[string]interface{}{
				"type": "string",
			},
		},
	})
	messages := []llm.Message{
		{
			Identity: llm.Identity{Role: "user"},
			Parts: []llm.Part{
				llm.TextPart("What's the weather in Tokyo?"),
				llm.ToolDefinitionPart{
					Name:        "get_weather",
					Description: "Get weather",
					Parameters:  params,
				},
			},
		},
	}

	stream, err := provider.GenerateStream(ctx, messages)
	if err != nil {
		t.Fatalf("Failed to generate stream: %v", err)
	}

	var toolCalls []llm.ToolRequestPart
	for part := range stream {
		if reqPart, ok := part.(llm.ToolRequestPart); ok {
			toolCalls = append(toolCalls, reqPart)
		}
	}

	if len(toolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(toolCalls))
	}

	if toolCalls[0].Name != "get_weather" {
		t.Errorf("Expected tool name 'get_weather', got '%s'", toolCalls[0].Name)
	}

	if location, ok := toolCalls[0].Args["location"].(string); !ok || location != "Tokyo" {
		t.Errorf("Expected argument location='Tokyo', got '%v'", toolCalls[0].Args["location"])
	}
}

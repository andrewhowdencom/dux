package ollama_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/llm/provider/ollama"
	api "github.com/ollama/ollama/api"
)

func TestOllamaNew(t *testing.T) {
	// 1. Default fallback parameters
	prv, err := ollama.New(nil)
	if err != nil {
		t.Fatalf("expected no error for empty config, got %v", err)
	}
	if prv == nil {
		t.Fatalf("expected provider pointer to be returned")
	}

	// 2. Explicit overriding parameters
	explicitCfg := map[string]interface{}{
		"address": "http://127.0.0.1:9090",
		"model":   "llama3:latest",
	}
	_, err = ollama.New(explicitCfg)
	if err != nil {
		t.Fatalf("expected no error for mapped config, got %v", err)
	}

	// 3. Guaranteed schema decode errors checking
	invalidType := map[string]interface{}{
		"model": 8493, // Supposed to be string
	}
	_, err = ollama.New(invalidType)
	if err == nil {
		t.Fatalf("expected mapstructure decoding error for mismatched schema")
	}
}

func TestOllamaGenerateStream(t *testing.T) {
	// Create a generic HTTP server acting as the REST backend for the Ollama chat module
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/api/chat") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/x-ndjson")

		// Push two JSONlines stream chunks
		_ = json.NewEncoder(w).Encode(api.ChatResponse{
			Model:   "llama3",
			Message: api.Message{Content: "Hello "},
			Done:    false,
		})
		_ = json.NewEncoder(w).Encode(api.ChatResponse{
			Model:   "llama3",
			Message: api.Message{Content: "World!"},
			Done:    true,
		})
	}))
	defer srv.Close()

	cfg := map[string]interface{}{
		"address": srv.URL,
		"model":   "llama3",
	}

	prv, err := ollama.New(cfg)
	if err != nil {
		t.Fatalf("unexpected error creating mapped provider: %v", err)
	}

	out, err := prv.GenerateStream(context.Background(), []llm.Message{
		{
			Identity: llm.Identity{Role: "user"},
			Parts:    []llm.Part{llm.TextPart("Say hello!")},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error initializing stream: %v", err)
	}

	var chunks []string
	for part := range out {
		if text, ok := part.(llm.TextPart); ok {
			chunks = append(chunks, string(text))
		}
	}

	joined := strings.Join(chunks, "")
	if joined != "Hello World!" {
		t.Errorf("expected joined stream chunks to equal 'Hello World!', got %q", joined)
	}
}

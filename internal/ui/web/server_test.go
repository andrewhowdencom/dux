package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/andrewhowdencom/dux/internal/config"
	"github.com/andrewhowdencom/dux/pkg/llm"
)

type mockStreamer struct {
	messages []llm.Message
	err      error
}

func (m *mockStreamer) Stream(ctx context.Context, inputMessage llm.Message) (<-chan llm.Message, error) {
	if m.err != nil {
		return nil, m.err
	}
	out := make(chan llm.Message)
	go func() {
		defer close(out)
		for _, msg := range m.messages {
			select {
			case <-ctx.Done():
				return
			case out <- msg:
				time.Sleep(10 * time.Millisecond) // Simulate stream delay
			}
		}
	}()
	return out, nil
}

func TestHandleChat_StreamingNDJSON(t *testing.T) {
	mockEng := &mockStreamer{
		messages: []llm.Message{
			{Parts: []llm.Part{llm.TextPart("Hello")}},
			{Parts: []llm.Part{llm.TextPart(" World")}},
			{Parts: []llm.Part{llm.ToolRequestPart{ToolID: "123", Name: "test_tool", Args: map[string]interface{}{"foo": "bar"}}}},
		},
	}

	factory := func(ctx context.Context, agentName, providerID, agentsFilePath string, hitl llm.HITLHandler, unsafeAllTools bool) (Streamer, *config.InstanceConfig, func(), error) {
		return mockEng, &config.InstanceConfig{}, func() {}, nil
	}

	key := make([]byte, 32)
	enc, _ := encryptSessionID(key, "test-session")

	srv := &Server{
		hitl:          NewWebHITL(),
		engineFactory: factory,
		sessionKey:    key,
		sessions:      make(map[string]*Session),
		toolDisplayCfg: config.ToolDisplayConfig{
			Enabled:     true,
			DefaultIcon: "🔧",
		},
	}

	payload := map[string]string{
		"agent":  "test-agent",
		"prompt": "Say hello",
	}
	b, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(b))
	req.AddCookie(&http.Cookie{Name: "dux_session", Value: enc})
	rec := httptest.NewRecorder()

	srv.handleChat(rec, req)

	res := rec.Result()

	if res.Header.Get("Content-Type") != "application/x-ndjson" {
		t.Errorf("Expected Content-Type application/x-ndjson, got %s", res.Header.Get("Content-Type"))
	}

	lines := strings.Split(strings.TrimSpace(rec.Body.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("Expected 4 stream lines (2 text + 1 tool_call + 1 hitl_request), got %d", len(lines))
	}

	// Verify line 1
	var out1 map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &out1); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}
	if out1["type"] != "text" || out1["content"] != "Hello" {
		t.Errorf("Line 1 mismatch: %v", out1)
	}

	// Verify line 2
	var out2 map[string]interface{}
	if err := json.Unmarshal([]byte(lines[1]), &out2); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}
	if out2["type"] != "text" || out2["content"] != " World" {
		t.Errorf("Line 2 mismatch: %v", out2)
	}

	// Verify line 3 (tool_call)
	var out3 map[string]interface{}
	if err := json.Unmarshal([]byte(lines[2]), &out3); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}
	if out3["type"] != "tool_call" || out3["tool"] != "test_tool" {
		t.Errorf("Line 3 mismatch: %v", out3)
	}

	// Verify line 4
	var out4 map[string]interface{}
	if err := json.Unmarshal([]byte(lines[3]), &out4); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}
	if out4["type"] != "hitl_request" || out4["call_id"] != "123" || out4["tool"] != "test_tool" {
		t.Errorf("Line 4 mismatch: %v", out4)
	}
}

func TestHandleChat_EngineError(t *testing.T) {
	factory := func(ctx context.Context, agentName, providerID, agentsFilePath string, hitl llm.HITLHandler, unsafeAllTools bool) (Streamer, *config.InstanceConfig, func(), error) {
		return nil, nil, nil, errors.New("engine bootstrap error")
	}

	key := make([]byte, 32)
	enc, _ := encryptSessionID(key, "test-session")

	srv := &Server{
		hitl:          NewWebHITL(),
		engineFactory: factory,
		sessionKey:    key,
		sessions:      make(map[string]*Session),
	}

	payload := map[string]string{"agent": "test-agent", "prompt": "Say hello"}
	b, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(b))
	req.AddCookie(&http.Cookie{Name: "dux_session", Value: enc})
	rec := httptest.NewRecorder()

	srv.handleChat(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500 status code, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "engine bootstrap error") {
		t.Errorf("Expected error to be written to body, got %s", rec.Body.String())
	}
}

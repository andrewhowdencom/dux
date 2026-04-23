package adapter_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/llm/adapter"
	"github.com/andrewhowdencom/dux/pkg/llm/enrich"
	"github.com/andrewhowdencom/dux/pkg/llm/provider/static"
	"github.com/andrewhowdencom/dux/pkg/memory/working"
)

func TestEngineE2EProgrammaticGo(t *testing.T) {
	// 1. Initialize the Provider
	// The static provider yields exactly what it is configured to yield.
	expectedReply := "Hello from the pure Go library!"
	prv, err := static.New(static.WithText(expectedReply))
	if err != nil {
		t.Fatalf("failed to create static provider: %v", err)
	}

	// 2. Initialize History storage
	memHistory := working.NewInMemory()

	// 3. Initialize the Engine using Variadic Options
	engine := adapter.New(
		adapter.WithProvider(prv),
		adapter.WithHistory(memHistory),
		adapter.WithSystemPrompt("You are a helpful agent testing programmatic initialization."),
		adapter.WithEnrichers(
			[]llm.BeforeGenerateHook{
				enrich.NewTime(),
				enrich.NewOS(),
			},
		),
	)

	// 4. Stream Results
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	sessionID := "e2e-session-uuid"
	msg := llm.Message{
		SessionID: sessionID,
		Identity:  llm.Identity{Role: "user"},
		Parts:     []llm.Part{llm.TextPart("Can you verify this setup?")},
	}

	stream, err := engine.Stream(llm.WithSessionID(ctx, sessionID), msg)
	if err != nil {
		t.Fatalf("failed to stream: %v", err)
	}

	var output strings.Builder
	for yield := range stream {
		for _, part := range yield.Parts {
			if textPart, ok := part.(llm.TextPart); ok {
				output.WriteString(string(textPart))
			}
		}
	}

	// 5. Assert the result matches the static provider configuration
	if output.String() != expectedReply {
		t.Errorf("expected engine stream output %q, got %q", expectedReply, output.String())
	}

	// 6. Assert history captured correctly
	// Note: System prompts and Enrichments are mapped onto the `sessionID` invisibly at initialization/execution.
	// Therefore, the persistent history should contain User Msg -> Assistant Output
	storedMsgs, err := memHistory.Read(llm.WithSessionID(ctx, sessionID), sessionID)
	if err != nil {
		t.Fatalf("failed fetching history: %v", err)
	}

	// History may contain specific roles. We check the bounds.
	if len(storedMsgs) < 2 {
		t.Fatalf("expected at least 2 history messages (1 user prompt, 1 reply), got %v", len(storedMsgs))
	}

	lastRole := storedMsgs[len(storedMsgs)-1].Identity.Role
	if lastRole != "assistant" {
		t.Errorf("expected last history message role 'assistant', got %q", lastRole)
	}
	

	// If the system prompt was stored first, it's 'system', otherwise 'user'.
	// Usually, the system prompt doesn't get saved to persistent store but sent out every time.
	// But let's verify either way:
	foundUserMsg := false
	for _, m := range storedMsgs {
		if m.Identity.Role == "user" {
			foundUserMsg = true
			break
		}
	}
	if !foundUserMsg {
		t.Errorf("expected user input message in history")
	}
}

package llm_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

type mockReceiver struct {
	ch chan llm.Message
}

func (m *mockReceiver) Receive(ctx context.Context) (<-chan llm.Message, error) {
	return m.ch, nil
}

type mockSender struct {
	mu       sync.Mutex
	messages []llm.Message
}

func (m *mockSender) Send(ctx context.Context, msg llm.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
	return nil
}

type mockEngine struct {
	mu           sync.Mutex
	processDelay time.Duration
	handled      []llm.Message
}

func (m *mockEngine) Stream(ctx context.Context, inputMessage llm.Message) (<-chan llm.Message, error) {
	m.mu.Lock()
	m.handled = append(m.handled, inputMessage)
	m.mu.Unlock()

	outCh := make(chan llm.Message)

	// Spin up routine to mock stream behavior
	go func() {
		defer close(outCh)
        
        // Ensure we handle Parts properly when extracting content for mock response
        inText := "unknown"
        if len(inputMessage.Parts) > 0 {
            if tp, ok := inputMessage.Parts[0].(llm.TextPart); ok {
                inText = string(tp)
            }
        }

		select {
		case <-time.After(m.processDelay):
			// Yield intermediate update
			outCh <- llm.Message{
				Parts: []llm.Part{llm.TextPart(fmt.Sprintf("Status: processing %s", inText))},
			}
			// Yield final update
			outCh <- llm.Message{
				Parts: []llm.Part{llm.TextPart(fmt.Sprintf("Response to: %s", inText))},
			}
		case <-ctx.Done():
            // If killed mid-flight, just exit cleanly
			return 
		}
	}()

	return outCh, nil
}

func TestSessionHandler_StrictQueuing_WithStreams(t *testing.T) {
	inCh := make(chan llm.Message, 5)
	receiver := &mockReceiver{ch: inCh}
	sender := &mockSender{}
	
	// Engine with a slight delay to simulate inference time
	engine := &mockEngine{processDelay: 50 * time.Millisecond}

	handler := llm.NewSessionHandler(engine, receiver, sender)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Push two messages into the channel rapidly
	inCh <- llm.Message{Parts: []llm.Part{llm.TextPart("m1")}}
	inCh <- llm.Message{Parts: []llm.Part{llm.TextPart("m2")}}
	close(inCh) // Closing stops the listener gracefully once drained

	err := handler.ListenAndServe(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sender.mu.Lock()
	defer sender.mu.Unlock()

	// 2 messages in * 2 stream parts per message = 4 expected outbound messages
	if len(sender.messages) != 4 {
		t.Fatalf("expected 4 stream responses, got %d", len(sender.messages))
	}

	if string(sender.messages[0].Parts[0].(llm.TextPart)) != "Status: processing m1" {
		t.Errorf("expected Status: processing m1, got %s", sender.messages[0].Parts[0])
	}
	if string(sender.messages[1].Parts[0].(llm.TextPart)) != "Response to: m1" {
		t.Errorf("expected Response to: m1, got %s", sender.messages[1].Parts[0])
	}
	if string(sender.messages[2].Parts[0].(llm.TextPart)) != "Status: processing m2" {
		t.Errorf("expected Status: processing m2, got %s", sender.messages[2].Parts[0])
	}
}

func TestSessionHandler_ContextCancellation(t *testing.T) {
	inCh := make(chan llm.Message, 5)
	receiver := &mockReceiver{ch: inCh}
	sender := &mockSender{}
	
	// Very slow engine to test cancellation
	engine := &mockEngine{processDelay: 5 * time.Second}

	handler := llm.NewSessionHandler(engine, receiver, sender)

	ctx, cancel := context.WithCancel(context.Background())

	inCh <- llm.Message{Parts: []llm.Part{llm.TextPart("cancel_me")}}

	// Cancel context quickly before engine finishes
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := handler.ListenAndServe(ctx)
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

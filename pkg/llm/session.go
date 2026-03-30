package llm

import "context"

// Engine represents the core LLM execution block (which wraps internal agentic tool convergence)
// The engine handles its own back-and-forth reasoning inside its implementation.
type Engine interface {
	// Stream spins up internal resources and returns an unbuffered channel that yields
	// intermediate states or final Messages. The Engine is responsible for closing the channel.
	Stream(ctx context.Context, inputMessage Message) (<-chan Message, error)
}

// SessionHandler wireframes the core Input/Output loop.
// It enforces strict request queuing and propagates Contexts for graceful cancellation.
type SessionHandler struct {
	engine   Engine
	receiver Receiver
	sender   Sender
}

// ListenAndServe blocks to process incoming inputs sequentially.
// Under strict queuing, it ranges over the Engine's Stream until it hits EOF,
// ensuring the current prompt finishes entirely before dequeuing the next prompt.
func (s *SessionHandler) ListenAndServe(ctx context.Context) error {
	inCh, err := s.receiver.Receive(ctx)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			// Explicit parent session cancellation
			return ctx.Err()
		case msg, ok := <-inCh:
			if !ok {
				// Input stream closed. Stop the session natively.
				return nil
			}

			// Setup execution-specific context mapping (so inference can be explicitly aborted)
			inferCtx, cancelInference := context.WithCancel(ctx)

			streamCh, err := s.engine.Stream(inferCtx, msg)
			if err != nil {
				// If the engine instantiation fails, push an error trace.
				_ = s.sender.Send(inferCtx, Message{
					SessionID: msg.SessionID,
					Identity:  Identity{Role: "system"},
					Parts:     []Part{TextPart("Error generating response: " + err.Error())},
				})
			} else {
				// The strict queue blocks here, ranging over the engine's stream
				// until it closes, feeding every intermediate update to the Sender sequentially.
				for outMsg := range streamCh {
					_ = s.sender.Send(inferCtx, outMsg)
				}
			}

			// Cleanup the context block before returning to the core select loop
			cancelInference()
		}
	}
}

// NewSessionHandler creates a fully configured session orchestrator.
func NewSessionHandler(engine Engine, receiver Receiver, sender Sender) *SessionHandler {
	return &SessionHandler{
		engine:   engine,
		receiver: receiver,
		sender:   sender,
	}
}

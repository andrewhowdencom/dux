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
	engine    Engine
	receiver  Receiver
	sender    Sender
	resolvers []ToolResolver
}

type SessionOption func(*SessionHandler)

// WithResolver injects dynamic capabilities into the session loop.
func WithResolver(r ToolResolver) SessionOption {
	return func(s *SessionHandler) {
		s.resolvers = append(s.resolvers, r)
	}
}

// NewSessionHandler creates a fully configured session orchestrator.
func NewSessionHandler(engine Engine, receiver Receiver, sender Sender, opts ...SessionOption) *SessionHandler {
	s := &SessionHandler{
		engine:   engine,
		receiver: receiver,
		sender:   sender,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
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

			s.processMessage(inferCtx, msg)

			// Cleanup the context block before returning to the core select loop
			cancelInference()
		}
	}
}

func (s *SessionHandler) processMessage(ctx context.Context, msg Message) {
	tools, err := s.resolveTools(ctx)
	if err != nil {
		s.sendError(ctx, msg.SessionID, "Error resolving tools: "+err.Error())
		return
	}

	// Always inject definitions so providers know what tools are available downstream
	// We avoid duplicating them if the sender already provided them.
	hasToolDefs := false
	for _, part := range msg.Parts {
		if part.Type() == TypeToolDefinition {
			hasToolDefs = true
			break
		}
	}
	if !hasToolDefs {
		for _, t := range tools {
			msg.Parts = append(msg.Parts, t.Definition())
		}
	}

	streamCh, err := s.engine.Stream(ctx, msg)
	if err != nil {
		s.sendError(ctx, msg.SessionID, "Error generating response: "+err.Error())
		return
	}

	var pendingToolCalls []ToolRequestPart

	// The strict queue blocks here, ranging over the engine's stream
	// until it closes, feeding every intermediate update to the Sender sequentially.
	for outMsg := range streamCh {
		var filteredParts []Part
		for _, part := range outMsg.Parts {
			if tr, ok := part.(ToolRequestPart); ok {
				pendingToolCalls = append(pendingToolCalls, tr)
			} else {
				filteredParts = append(filteredParts, part)
			}
		}

		if len(filteredParts) > 0 {
			outMsg.Parts = filteredParts
			_ = s.sender.Send(ctx, outMsg)
		}
	}

	// If tools were requested, execute them and recursively call processMessage
	if len(pendingToolCalls) > 0 {
		var resultParts []Part
		for _, tc := range pendingToolCalls {
			resultPart := ToolResultPart{
				ToolID: tc.ToolID,
				Name:   tc.Name,
			}

			var targetTool Tool
			for _, t := range tools {
				if t.Name() == tc.Name {
					targetTool = t
					break
				}
			}

			if targetTool == nil {
				resultPart.Result = "Error: Tool not found"
				resultPart.IsError = true
			} else {
				res, err := targetTool.Execute(ctx, tc.Args)
				if err != nil {
					resultPart.Result = err.Error()
					resultPart.IsError = true
				} else {
					resultPart.Result = res
				}
			}
			resultParts = append(resultParts, resultPart)
		}

		// Re-invoke the Engine with tool answers
		toolMsg := Message{
			SessionID: msg.SessionID,
			Identity:  Identity{Role: "tool"},
			Parts:     resultParts,
		}
		s.processMessage(ctx, toolMsg)
	}
}

func (s *SessionHandler) resolveTools(ctx context.Context) ([]Tool, error) {
	var allTools []Tool
	for _, r := range s.resolvers {
		tools, err := r.Resolve(ctx)
		if err != nil {
			return nil, err
		}
		allTools = append(allTools, tools...)
	}
	return allTools, nil
}

func (s *SessionHandler) sendError(ctx context.Context, sessionID, text string) {
	_ = s.sender.Send(ctx, Message{
		SessionID: sessionID,
		Identity:  Identity{Role: "system"},
		Parts:     []Part{TextPart(text)},
	})
}

package adapter

import (
	"context"
	"log/slog"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/llm/enrich"
	"github.com/andrewhowdencom/dux/pkg/llm/history"
	"github.com/andrewhowdencom/dux/pkg/llm/provider"
)

// Engine orchestrates the convergence loop between the LLM provider,
// tools, and conversation history.
type Engine struct {
	history      history.History
	provider     provider.Provider
	systemPrompt string
	enrichers    []enrich.Enricher
}

// Option configures the Engine via the functional options pattern.
type Option func(*Engine)

// WithHistory sets the engine's history backend.
func WithHistory(h history.History) Option {
	return func(e *Engine) {
		e.history = h
	}
}

// WithProvider sets the core LLM inference provider.
func WithProvider(p provider.Provider) Option {
	return func(e *Engine) {
		e.provider = p
	}
}



// WithSystemPrompt sets an overarching system prompt injected dynamically at stream time.
func WithSystemPrompt(prompt string) Option {
	return func(e *Engine) {
		e.systemPrompt = prompt
	}
}

// WithEnrichers sets the dynamic context enrichers to be evaluated before streaming.
func WithEnrichers(enrichers []enrich.Enricher) Option {
	return func(e *Engine) {
		e.enrichers = enrichers
	}
}

// New creates a new Engine configured with the provided options.
func New(opts ...Option) *Engine {
	e := &Engine{}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Stream executes the recursive convergence loop.
func (e *Engine) Stream(ctx context.Context, inputMessage llm.Message) (<-chan llm.Message, error) {
	out := make(chan llm.Message)

	// Append initial input to history
	if e.history != nil {
		if err := e.history.Append(ctx, inputMessage.SessionID, inputMessage); err != nil {
			return nil, err
		}
	}

	go func() {
		defer close(out)

		// 1. Fetch current context
		var msgs []llm.Message
		if e.history != nil {
			var err error
			msgs, err = e.history.GetMessages(ctx, inputMessage.SessionID)
			if err != nil {
				e.sendError(ctx, out, err, inputMessage.SessionID)
				return
			}
		} else {
			// Fallback if no history is configured
			msgs = []llm.Message{inputMessage}
		}

		if e.systemPrompt != "" {
			systemMsg := llm.Message{
				SessionID: inputMessage.SessionID,
				Identity:  llm.Identity{Role: "system"},
				Parts:     []llm.Part{llm.TextPart(e.systemPrompt)},
			}
			msgs = append([]llm.Message{systemMsg}, msgs...)
		}

		// 1a. Evaluate dynamic enrichers if configured
		if len(e.enrichers) > 0 {
			var enrichmentData string
			for _, en := range e.enrichers {
				res, err := en.Enrich(ctx)
				if err != nil {
					slog.Debug("failed to evaluate enricher", "type", en.Type(), "error", err)
					continue
				}
				if res != "" {
					enrichmentData += res + "\n"
				}
			}
			if enrichmentData != "" {
				enrichMsg := llm.Message{
					SessionID: inputMessage.SessionID,
					Identity:  llm.Identity{Role: "system"},
					Parts:     []llm.Part{llm.TextPart(enrichmentData)},
				}
				
				lastIdx := len(msgs) - 1
				if lastIdx >= 0 {
					// Insert immediately before the last user message
					msgs = append(msgs[:lastIdx], append([]llm.Message{enrichMsg}, msgs[lastIdx:]...)...)
				} else {
					msgs = append(msgs, enrichMsg)
				}
			}
		}

		// 2. Call Provider
		if e.provider == nil {
			// No provider configured, just exit without generating
			return
		}
		partStream, err := e.provider.GenerateStream(ctx, msgs)
		if err != nil {
			e.sendError(ctx, out, err, inputMessage.SessionID)
			return
		}

		// 3. Over the stream
		for part := range partStream {
			msg := llm.Message{
				SessionID: inputMessage.SessionID,
				Identity:  llm.Identity{Role: "assistant"},
				Parts:     []llm.Part{part},
			}
			if e.history != nil {
				if err := e.history.Append(ctx, msg.SessionID, msg); err != nil {
					e.sendError(ctx, out, err, msg.SessionID)
					return
				}
			}
			e.safeSend(ctx, out, msg)
		}
	}()

	return out, nil
}

func (e *Engine) sendError(ctx context.Context, out chan<- llm.Message, err error, sessionID string) {
	msg := llm.Message{
		SessionID: sessionID,
		Identity:  llm.Identity{Role: "system"},
		Parts:     []llm.Part{llm.TextPart("Error: " + err.Error())},
	}
	e.safeSend(ctx, out, msg)
}

func (e *Engine) safeSend(ctx context.Context, out chan<- llm.Message, msg llm.Message) {
	select {
	case <-ctx.Done():
	case out <- msg:
	}
}

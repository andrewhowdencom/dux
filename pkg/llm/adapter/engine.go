package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/llm/history"
	"github.com/andrewhowdencom/dux/pkg/llm/provider"
	"github.com/andrewhowdencom/dux/pkg/llm/tool"
)

// Engine orchestrates the convergence loop between the LLM provider,
// tools, and conversation history.
type Engine struct {
	history  history.History
	provider provider.Provider
	registry tool.Registry
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

// WithRegistry sets the tool registry available to the engine.
func WithRegistry(r tool.Registry) Option {
	return func(e *Engine) {
		e.registry = r
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

		for {
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

			// 2. Inject tool definitions
			if e.registry != nil {
				tools := e.registry.GetDefinitions()
				if len(tools) > 0 {
					msgs = append(msgs, llm.Message{
						SessionID: inputMessage.SessionID,
						Identity:  llm.Identity{Role: "system"},
						Parts:     tools,
					})
				}
			}

			// 3. Call Provider
			if e.provider == nil {
				// No provider configured, just exit without generating
				return
			}
			partStream, err := e.provider.GenerateStream(ctx, msgs)
			if err != nil {
				e.sendError(ctx, out, err, inputMessage.SessionID)
				return
			}

			var pendingTools []llm.ToolRequestPart

			// 4. Over the stream
			for part := range partStream {
				switch p := part.(type) {
				case llm.TextPart:
					msg := llm.Message{
						SessionID: inputMessage.SessionID,
						Identity:  llm.Identity{Role: "assistant"},
						Parts:     []llm.Part{p},
					}
					e.safeSend(ctx, out, msg)

				case llm.ToolRequestPart:
					pendingTools = append(pendingTools, p)
					msg := llm.Message{
						SessionID: inputMessage.SessionID,
						Identity:  llm.Identity{Role: "assistant"},
						Parts:     []llm.Part{p},
					}
					e.safeSend(ctx, out, msg)
				}
			}

			// 5. If no tools were requested, the loop ends (convergence reached).
			if len(pendingTools) == 0 {
				break
			}

			// 6. Execute tools and append results
			for _, tq := range pendingTools {
				var execResult interface{}
				var err error

				if e.registry != nil {
					execResult, err = e.registry.Execute(ctx, tq.Name, tq.Args)
				} else {
					err = fmt.Errorf("no tool registry configured")
				}

				var resultText string
				if err != nil {
					resultText = "error executing tool: " + err.Error()
				} else {
					if str, ok := execResult.(string); ok {
						resultText = str
					} else {
						if b, marshalErr := json.Marshal(execResult); marshalErr == nil {
							resultText = string(b)
						} else {
							resultText = "tool executed successfully"
						}
					}
				}

				if e.history != nil {
					toolResMsg := llm.Message{
						SessionID: inputMessage.SessionID,
						Identity:  llm.Identity{Role: "tool", Name: tq.Name},
						Parts:     []llm.Part{llm.TextPart(resultText)},
					}
					if err := e.history.Append(ctx, inputMessage.SessionID, toolResMsg); err != nil {
						e.sendError(ctx, out, err, inputMessage.SessionID)
						return
					}
				}
			}
			// After appending tool execution results, loop will repeat!
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

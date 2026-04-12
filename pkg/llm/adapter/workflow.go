package adapter

import (
	"context"
	"fmt"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// Factory is a callback that yields a brand new set of *Engine options
// to initialize the engine for the provided mode string natively. 
// It is the caller's responsibility to preserve shared Working Memory instances across calls.
type Factory func(modeName string) ([]Option, error)

// WorkflowEngine is an implementation of llm.Engine that securely wraps the core execution engine.
// It watches the stream for TransitionSignalPart markers and seamlessly re-configures the 
// underlying execution model without dropping the user's connection.
type WorkflowEngine struct {
	inner   *Engine
	factory Factory
}

// NewWorkflowEngine dynamically orchestrates multiple LLM contexts.
func NewWorkflowEngine(initialMode string, factory Factory) (*WorkflowEngine, error) {
	opts, err := factory(initialMode)
	if err != nil {
		return nil, fmt.Errorf("failed allocating mode %q: %w", initialMode, err)
	}

	inner := New(opts...)

	return &WorkflowEngine{
		inner:   inner,
		factory: factory,
	}, nil
}

// Stream securely intercepts Context Router transitions natively.
func (w *WorkflowEngine) Stream(ctx context.Context, msg llm.Message) (<-chan llm.Message, error) {
	out := make(chan llm.Message)

	go func() {
		defer close(out)
		
		// Create a recursive loop handler for hot-swaps
		var loop func(input llm.Message) error
		loop = func(input llm.Message) error {
			innerCh, err := w.inner.Stream(ctx, input)
			if err != nil {
				return err
			}

			for chunk := range innerCh {
				var transition *llm.TransitionSignalPart
				for _, p := range chunk.Parts {
					if ts, ok := p.(llm.TransitionSignalPart); ok {
						transition = &ts
						break
					}
				}

				if transition != nil {
					// The UI has already received the ToolRequestPart and visualized the intent.
					// We safely trap the discrete system signal.
					
					newOpts, err := w.factory(transition.TargetMode)
					if err != nil {
						errMsg := llm.Message{
							SessionID: msg.SessionID,
							Identity:  llm.Identity{Role: "system"},
							Parts:     []llm.Part{llm.TextPart(fmt.Sprintf("System Error: Failed to transition to mode '%s': %v", transition.TargetMode, err))},
						}
						out <- errMsg
						return err // Break the loop safely
					}

					// Hermetically hot-swap the engine instance entirely instead of attempting messy internal appends.
					w.inner = New(newOpts...)

					// Generate a hidden bridge message so the new context model immediately evaluates.
					resumeMsg := llm.Message{
						SessionID: msg.SessionID,
						Identity:  llm.Identity{Role: "system"},
						Parts:     []llm.Part{llm.TextPart(fmt.Sprintf("\n[System: State transitioned successfully to mode '%s'. Acknowledge dependencies and proceed.]", transition.TargetMode))},
					}

					return loop(resumeMsg)
				}

				// Otherwise, pass it along
				out <- chunk
			}

			return nil
		}

		_ = loop(msg)
	}()

	return out, nil
}

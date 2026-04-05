package ui

import (
	"context"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// View represents the absolute minimum capabilities required by a dynamic Dux interface.
type View interface {
	RenderTextChunk(chunk string)
	RenderError(err error)
	PromptHITL(req *llm.ToolRequestPart)
	Flush() // Signal the end of an incoming stream or a sync boundary
}

// ThinkingView an interface for UIs that support rendering the background reasoning / scratchpad natively.
type ThinkingView interface {
	RenderThinkingChunk(chunk string)
}

// ToolVisibilityView an interface for UIs that support visualizing what background API requests are being made on the user's behalf.
// Note: This visualizations is entirely disjoint from the HITL prompt (which is required by the core View).
type ToolVisibilityView interface {
	RenderToolIntent(toolName string, args any)
	RenderToolResult(toolName string, result any, isError bool)
}

// TelemetryView an interface for UIs that support tracking overall cost / tokens and performance statistics per message block.
type TelemetryView interface {
	RenderTelemetry(telemetry llm.TelemetryPart)
}

// ChatSession represents a unified conversation state that encapsulates context window streaming.
type ChatSession struct {
	ID      string
	Engine  llm.Engine
	View    View
	Cleanup func()
}

// StreamQuery sends the user message to the LLM backend via the Session's Engine, and pushes matching 
// rendering updates out to the View implementations interfaces if supported. 
func (s *ChatSession) StreamQuery(ctx context.Context, input string) error {
	ctxWithSession := llm.WithSessionID(ctx, s.ID)

	msg := llm.Message{
		SessionID: s.ID,
		Identity:  llm.Identity{Role: "user"},
		Parts: []llm.Part{
			llm.TextPart(input),
		},
	}

	streamChan, err := s.Engine.Stream(ctxWithSession, msg)
	if err != nil {
		s.View.RenderError(err)
		s.View.Flush()
		return err
	}

	for outMsg := range streamChan {
		for _, part := range outMsg.Parts {
			switch p := part.(type) {
			case llm.TextPart:
				s.View.RenderTextChunk(string(p))
			case llm.ReasoningPart:
				if tv, ok := s.View.(ThinkingView); ok {
					tv.RenderThinkingChunk(string(p))
				}
			case llm.ToolRequestPart:
				if tv, ok := s.View.(ToolVisibilityView); ok {
					tv.RenderToolIntent(p.Name, p.Args)
				}
				// Emit the HITL prompt interface
				// Note: It's up to the View to route the boolean approval back to the HITLHandler.
				// The Core View handles HITL differently. We just inform the view it reached this point.
				s.View.PromptHITL(&p)
			case llm.ToolResultPart:
				if tv, ok := s.View.(ToolVisibilityView); ok {
					tv.RenderToolResult(p.Name, p.Result, p.IsError)
				}
			case llm.TelemetryPart:
				if tv, ok := s.View.(TelemetryView); ok {
					tv.RenderTelemetry(p)
				}
			}
		}
		// Some views benefit from a per-message flush
		s.View.Flush()
	}

	// Final flush end of query
	s.View.Flush()

	return nil
}

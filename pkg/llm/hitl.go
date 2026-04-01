package llm

import (
	"context"
	"fmt"
)

// HITLHandler defines how the system explicitly requests human confirmation
// to execute a tool.
type HITLHandler interface {
	ApproveTool(ctx context.Context, req ToolRequestPart) (bool, error)
}

// NewHITLMiddleware wraps an execution loop with a conditional prompt.
// requiresSupervision flags each tool individually. If a tool is not mapped,
// or is mapped to true, the user must approve it.
// If unsafeAllTools is true, all interactive prompts are bypassed cleanly.
func NewHITLMiddleware(handler HITLHandler, requiresSupervision map[string]bool, unsafeAllTools bool) ToolMiddleware {
	return func(ctx context.Context, req ToolRequestPart, next func(ctx context.Context) (interface{}, error)) (interface{}, error) {
		if unsafeAllTools {
			return next(ctx)
		}

		needsSupervision, exists := requiresSupervision[req.Name]
		if !exists {
			// Unmapped tools (e.g. from dynamic MCP discovery) default to secure execution
			needsSupervision = true
		}

		if !needsSupervision {
			return next(ctx)
		}

		if handler == nil {
			return nil, fmt.Errorf("user denied tool execution: no interactive handler present to approve")
		}

		approved, err := handler.ApproveTool(ctx, req)
		if err != nil {
			return nil, err
		}

		if !approved {
			return nil, fmt.Errorf("user denied tool execution: please ask the user why they denied the tool and request follow-up instructions")
		}

		return next(ctx)
	}
}

package llm

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/cel-go/cel"
)

// HITLHandler defines how the system explicitly requests human confirmation
// to execute a tool.
type HITLHandler interface {
	ApproveTool(ctx context.Context, req ToolRequestPart) (bool, error)
}

// NewHITLMiddleware wraps an execution loop with a conditional prompt.
// requiresSupervision flags each tool namespace individually. If a namespace is not mapped,
// or is mapped to true/fails evaluation, the user must approve it.
// If unsafeAllTools is true, all interactive prompts are bypassed cleanly.
func NewHITLMiddleware(handler HITLHandler, requiresSupervision map[string]interface{}, unsafeAllTools bool) ToolMiddleware {
	env, err := cel.NewEnv(
		cel.Variable("tool_name", cel.StringType),
		cel.Variable("namespace", cel.StringType),
		cel.Variable("args", cel.DynType),
	)
	if err != nil {
		slog.Error("failed to create CEL env", "err", err)
	}

	programs := make(map[string]cel.Program)
	for ns, val := range requiresSupervision {
		if s, ok := val.(string); ok && env != nil {
			ast, issues := env.Compile(s)
			if issues != nil && issues.Err() != nil {
				slog.Error("failed to compile supervision CEL rule", "namespace", ns, "err", issues.Err())
				continue
			}
			prg, err := env.Program(ast)
			if err != nil {
				slog.Error("failed to instantiate CEL program", "namespace", ns, "err", err)
				continue
			}
			programs[ns] = prg
		}
	}

	return func(ctx context.Context, req ToolRequestPart, next func(ctx context.Context) (interface{}, error)) (interface{}, error) {
		if unsafeAllTools {
			return next(ctx)
		}

		namespace, _ := ctx.Value(ContextKeyNamespace).(string)

		var needsSupervision bool
		policy, exists := requiresSupervision[namespace]
		
		if !exists {
			// Unmapped tools fallback to secure execution
			needsSupervision = true
		} else {
			if b, ok := policy.(bool); ok {
				needsSupervision = b
			} else if p, ok := programs[namespace]; ok {
				argsMap := req.Args
				if argsMap == nil {
					argsMap = make(map[string]interface{})
				}
				
				out, _, err := p.Eval(map[string]interface{}{
					"tool_name": req.Name,
					"namespace": namespace,
					"args":      argsMap,
				})
				
				if err != nil {
					slog.Error("CEL evaluation failed, defaulting to supervision=true", "err", err)
					needsSupervision = true
				} else if val, ok := out.Value().(bool); ok {
					needsSupervision = val
				} else {
					slog.Warn("CEL expression didn't evaluate to a bool, defaulting to true", "namespace", namespace)
					needsSupervision = true
				}
			} else {
				needsSupervision = true
			}
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

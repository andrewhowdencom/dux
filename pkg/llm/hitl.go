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

// NewHITLHook returns a BeforeToolHook that conditionally requires human
// approval before a tool is executed.
func NewHITLHook(handler HITLHandler, requiresSupervision map[string]interface{}, unsafeAllTools bool) BeforeToolHook {
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

	return func(ctx context.Context, req BeforeToolRequest) error {
		if unsafeAllTools {
			return nil
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
				argsMap := req.ToolCall.Args
				if argsMap == nil {
					argsMap = make(map[string]interface{})
				}

				out, _, err := p.Eval(map[string]interface{}{
					"tool_name": req.ToolCall.Name,
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
			return nil
		}

		if handler == nil {
			return fmt.Errorf("user denied tool execution: no interactive handler present to approve")
		}

		approved, err := handler.ApproveTool(ctx, req.ToolCall)
		if err != nil {
			return err
		}

		if !approved {
			return fmt.Errorf("user denied tool execution: please ask the user why they denied the tool and request follow-up instructions")
		}

		return nil
	}
}

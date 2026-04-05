package stdlib

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Knetic/govaluate"
	"github.com/andrewhowdencom/dux/pkg/llm"
)

type MathTool struct{}

func NewMath() llm.Tool {
	return &MathTool{}
}

func (t *MathTool) Name() string { return "evaluate_math" }

func (t *MathTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        t.Name(),
		Description: "Safely evaluates mathematical expressions (e.g. 254 * 3.14 / 2). Avoid performing math manually, rely on this tool to get mathematically accurate results.",
		Parameters: json.RawMessage(`{"type":"object","properties":{"expression":{"type":"string","description":"The mathematical expression to evaluate string (e.g. 3 * 4)"}},"required":["expression"]}`),
	}
}

func (t *MathTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	exprStr, ok := args["expression"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'expression' parameter")
	}

	expression, err := govaluate.NewEvaluableExpression(exprStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse math expression: %w", err)
	}

	result, err := expression.Evaluate(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate expression: %w", err)
	}

	return map[string]interface{}{
		"result": result,
	}, nil
}

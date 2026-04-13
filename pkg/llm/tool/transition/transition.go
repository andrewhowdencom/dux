package transition

import (
	"context"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// Tool provides a dynamically instantiated tool that triggers a Context Router state change.
type Tool struct {
	targetMode  string
	description string
}

// New creates a new transition tool targeting a specific state in the Workflow.
func New(targetMode, description string) *Tool {
	return &Tool{
		targetMode:  targetMode,
		description: description,
	}
}

// Execute emits the TransitionSignalPart which safely trips the workflow engine interceptors.
func (t *Tool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	reason := ""
	if r, ok := args["reason"].(string); ok {
		reason = r
	}

	return llm.TransitionSignalPart{
		TargetMode: t.targetMode,
		Message:    reason,
	}, nil
}

func (t *Tool) Name() string {
	return "switch_to_" + t.targetMode
}

// Definition formats the tool accurately for the LLM to understand.
func (t *Tool) Definition() llm.ToolDefinitionPart {
	importJSON := []byte(`{
		"type": "object",
		"properties": {
			"reason": {
				"type": "string",
				"description": "Optional reasoning for why you are executing this state transition."
			}
		}
	}`)
	return llm.ToolDefinitionPart{
		Name:        t.Name(),
		Description: t.description,
		Parameters:  importJSON,
	}
}

var _ llm.Tool = (*Tool)(nil)

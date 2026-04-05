package time

import (
	"context"
	"encoding/json"
	"time"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// TimeTool provides the current system time to the agent.
type TimeTool struct{}

// New returns a fresh instance of the time tool.
func New() llm.Tool {
	return &TimeTool{}
}

func (t *TimeTool) Name() string { return "get_current_time" }

func (t *TimeTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        t.Name(),
		Description: "Returns the current system time in RFC3339 format. Useful when you need to know what time it is.\n\n### Examples\n\n**Example 1: basic usage**\n```json\n{}\n```",
		Parameters:  json.RawMessage(`{"type":"object","properties":{}}`),
	}
}

func (t *TimeTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return map[string]string{
		"time": time.Now().Format(time.RFC3339),
	}, nil
}

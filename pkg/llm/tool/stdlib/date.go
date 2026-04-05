package stdlib

import (
	"context"
	"encoding/json"
	"time"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// DateTool provides the current system date to the agent.
type DateTool struct{}

// NewDate returns a fresh instance of the date tool.
func NewDate() llm.Tool {
	return &DateTool{}
}

func (d *DateTool) Name() string { return "get_current_date" }

func (d *DateTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        d.Name(),
		Description: "Returns the current system date in YYYY-MM-DD format. Useful when you need to know what the current date is.\n\n### Examples\n\n**Example 1: basic usage**\n```json\n{}\n```",
		Parameters:  json.RawMessage(`{"type":"object","properties":{}}`),
	}
}

func (d *DateTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return map[string]string{
		"date": time.Now().Format("2006-01-02"),
	}, nil
}

package stdlib

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// TimerTool allows setting a timer that blocks for a given duration.
type TimerTool struct{}

// NewTimer returns a fresh instance of the timer tool.
func NewTimer() llm.Tool {
	return &TimerTool{}
}

func (t *TimerTool) Name() string { return "timer" }

func (t *TimerTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        t.Name(),
		Description: "Sets a timer for a given number of seconds. The tool will block and wait for the specified duration before returning.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"duration_seconds": {
					"type": "integer",
					"description": "The duration of the timer in seconds."
				}
			},
			"required": ["duration_seconds"]
		}`),
	}
}

func (t *TimerTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	durationInterface, ok := args["duration_seconds"]
	if !ok {
		return nil, fmt.Errorf("missing required argument 'duration_seconds'")
	}

	var durationSeconds int
	switch v := durationInterface.(type) {
	case int:
		durationSeconds = v
	case float64:
		durationSeconds = int(v)
	default:
		return nil, fmt.Errorf("argument 'duration_seconds' must be an integer")
	}

	if durationSeconds < 0 {
		return nil, fmt.Errorf("duration_seconds must be non-negative")
	}

	timer := time.NewTimer(time.Duration(durationSeconds) * time.Second)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		return map[string]string{
			"status":  "success",
			"message": fmt.Sprintf("Timer for %d seconds completed.", durationSeconds),
		}, nil
	}
}

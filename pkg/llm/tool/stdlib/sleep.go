package stdlib

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

type SleepTool struct{}

func NewSleep() llm.Tool {
	return &SleepTool{}
}

func (t *SleepTool) Name() string { return "sleep" }

func (t *SleepTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        t.Name(),
		Description: "Pauses your execution thread for a specific number of milliseconds. Useful for waiting for a known background process to finish before trying to check its status.",
		Parameters: json.RawMessage(`{"type":"object","properties":{"milliseconds":{"type":"integer","description":"The amount of time in milliseconds to halt execution. Do not exceed 10000 (10s) without good reason."}},"required":["milliseconds"]}`),
	}
}

func (t *SleepTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	ms, ok := args["milliseconds"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'milliseconds' parameter")
	}

	sleepTime := time.Duration(ms) * time.Millisecond
	
	select {
	case <-time.After(sleepTime):
		return map[string]interface{}{
			"status": "slept",
			"slept_ms": ms,
		}, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("sleep interrupted by context cancellation: %w", ctx.Err())
	}
}

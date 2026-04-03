package time

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

// StopwatchTool provides start, stop, and status capabilities for named stopwatches.
type StopwatchTool struct {
	mu      sync.Mutex
	watches map[string]time.Time
}

// NewStopwatch returns a fresh instance of the stopwatch tool.
func NewStopwatch() llm.Tool {
	return &StopwatchTool{
		watches: make(map[string]time.Time),
	}
}

func (t *StopwatchTool) Name() string { return "stopwatch" }

func (t *StopwatchTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        t.Name(),
		Description: "A stopwatch tool to track elapsed time. You can start, stop, or check the status of a stopwatch. A stopwatch is identified by a name.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"action": {
					"type": "string",
					"enum": ["start", "stop", "status"],
					"description": "The action to perform on the stopwatch."
				},
				"name": {
					"type": "string",
					"description": "The name of the stopwatch."
				}
			},
			"required": ["action", "name"]
		}`),
	}
}

func (t *StopwatchTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	actionInterface, ok := args["action"]
	if !ok {
		return nil, fmt.Errorf("missing required argument 'action'")
	}

	action, ok := actionInterface.(string)
	if !ok {
		return nil, fmt.Errorf("argument 'action' must be a string")
	}

	nameInterface, ok := args["name"]
	if !ok {
		return nil, fmt.Errorf("missing required argument 'name'")
	}

	name, ok := nameInterface.(string)
	if !ok {
		return nil, fmt.Errorf("argument 'name' must be a string")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	switch action {
	case "start":
		if _, exists := t.watches[name]; exists {
			return nil, fmt.Errorf("stopwatch '%s' is already running", name)
		}
		t.watches[name] = time.Now()
		return map[string]interface{}{
			"status":  "started",
			"message": fmt.Sprintf("Stopwatch '%s' started.", name),
		}, nil

	case "stop":
		startTime, exists := t.watches[name]
		if !exists {
			return nil, fmt.Errorf("stopwatch '%s' is not running", name)
		}
		elapsed := time.Since(startTime)
		delete(t.watches, name)
		return map[string]interface{}{
			"status":           "stopped",
			"message":          fmt.Sprintf("Stopwatch '%s' stopped.", name),
			"elapsed_seconds":  elapsed.Seconds(),
			"elapsed_duration": elapsed.String(),
		}, nil

	case "status":
		startTime, exists := t.watches[name]
		if !exists {
			return nil, fmt.Errorf("stopwatch '%s' is not running", name)
		}
		elapsed := time.Since(startTime)
		return map[string]interface{}{
			"status":           "running",
			"message":          fmt.Sprintf("Stopwatch '%s' is currently running.", name),
			"elapsed_seconds":  elapsed.Seconds(),
			"elapsed_duration": elapsed.String(),
		}, nil

	default:
		return nil, fmt.Errorf("invalid action '%s'. Must be 'start', 'stop', or 'status'", action)
	}
}

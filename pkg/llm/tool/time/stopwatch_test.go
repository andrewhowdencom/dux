package time

import (
	"context"
	"testing"
	"time"
)

func TestStopwatchTool_Execute(t *testing.T) {
	tool := NewStopwatch()
	ctx := context.Background()

	// Test starting a stopwatch
	res, err := tool.Execute(ctx, map[string]interface{}{
		"action": "start",
		"name":   "test1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resMap, ok := res.(map[string]interface{})
	if !ok || resMap["status"] != "started" {
		t.Fatalf("expected status 'started', got %v", res)
	}

	// Test starting same stopwatch again
	_, err = tool.Execute(ctx, map[string]interface{}{
		"action": "start",
		"name":   "test1",
	})
	if err == nil {
		t.Fatalf("expected error when starting an already running stopwatch")
	}

	// Test status
	time.Sleep(10 * time.Millisecond)
	res, err = tool.Execute(ctx, map[string]interface{}{
		"action": "status",
		"name":   "test1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resMap, _ = res.(map[string]interface{})
	if resMap["status"] != "running" {
		t.Fatalf("expected status 'running', got %v", res)
	}
	if elapsed, ok := resMap["elapsed_seconds"].(float64); !ok || elapsed <= 0 {
		t.Fatalf("expected positive elapsed_seconds, got %v", resMap["elapsed_seconds"])
	}

	// Test stopping
	res, err = tool.Execute(ctx, map[string]interface{}{
		"action": "stop",
		"name":   "test1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resMap, _ = res.(map[string]interface{})
	if resMap["status"] != "stopped" {
		t.Fatalf("expected status 'stopped', got %v", res)
	}

	// Test status on stopped
	_, err = tool.Execute(ctx, map[string]interface{}{
		"action": "status",
		"name":   "test1",
	})
	if err == nil {
		t.Fatalf("expected error when checking status of stopped stopwatch")
	}

	// Test stopping already stopped
	_, err = tool.Execute(ctx, map[string]interface{}{
		"action": "stop",
		"name":   "test1",
	})
	if err == nil {
		t.Fatalf("expected error when stopping an already stopped stopwatch")
	}
}

func TestStopwatchTool_Errors(t *testing.T) {
	tool := NewStopwatch()
	ctx := context.Background()

	tests := []struct {
		name        string
		args        map[string]interface{}
		expectError bool
	}{
		{
			name:        "missing action",
			args:        map[string]interface{}{"name": "t1"},
			expectError: true,
		},
		{
			name:        "missing name",
			args:        map[string]interface{}{"action": "start"},
			expectError: true,
		},
		{
			name:        "invalid action",
			args:        map[string]interface{}{"action": "invalid", "name": "t1"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tool.Execute(ctx, tt.args)
			if (err != nil) != tt.expectError {
				t.Fatalf("expected error: %v, got: %v", tt.expectError, err)
			}
		})
	}
}

func TestStopwatchTool_NameAndDefinition(t *testing.T) {
	tool := NewStopwatch()
	if tool.Name() != "stopwatch" {
		t.Errorf("expected name 'stopwatch', got '%s'", tool.Name())
	}

	def := tool.Definition()
	if def.Name != "stopwatch" {
		t.Errorf("expected definition name 'stopwatch', got '%s'", def.Name)
	}
}

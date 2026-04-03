package time

import (
	"context"
	"testing"
	"time"
)

func TestTimerTool_Execute(t *testing.T) {
	tool := NewTimer()

	tests := []struct {
		name        string
		args        map[string]interface{}
		expectError bool
		minDuration time.Duration
	}{
		{
			name: "valid duration",
			args: map[string]interface{}{
				"duration_seconds": 1,
			},
			expectError: false,
			minDuration: 1 * time.Second,
		},
		{
			name: "valid float duration",
			args: map[string]interface{}{
				"duration_seconds": 1.0,
			},
			expectError: false,
			minDuration: 1 * time.Second,
		},
		{
			name:        "missing duration",
			args:        map[string]interface{}{},
			expectError: true,
		},
		{
			name: "invalid duration type",
			args: map[string]interface{}{
				"duration_seconds": "1",
			},
			expectError: true,
		},
		{
			name: "negative duration",
			args: map[string]interface{}{
				"duration_seconds": -1,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			_, err := tool.Execute(context.Background(), tt.args)

			if (err != nil) != tt.expectError {
				t.Fatalf("expected error: %v, got: %v", tt.expectError, err)
			}

			if !tt.expectError {
				elapsed := time.Since(start)
				if elapsed < tt.minDuration {
					t.Fatalf("expected timer to wait at least %v, but it waited %v", tt.minDuration, elapsed)
				}
			}
		})
	}
}

func TestTimerTool_NameAndDefinition(t *testing.T) {
	tool := NewTimer()
	if tool.Name() != "timer" {
		t.Errorf("expected name 'timer', got '%s'", tool.Name())
	}

	def := tool.Definition()
	if def.Name != "timer" {
		t.Errorf("expected definition name 'timer', got '%s'", def.Name)
	}
}

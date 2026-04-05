package stdlib

import (
	"context"
	"testing"
)

func TestRandomTool_Execute(t *testing.T) {
	tool := NewRandom()
	ctx := context.Background()

	tests := []struct {
		name        string
		args        map[string]interface{}
		expectErr   bool
	}{
		{
			name:        "valid bounds",
			args:        map[string]interface{}{"min": float64(10), "max": float64(20)},
			expectErr:   false,
		},
		{
			name:        "invalid bounds",
			args:        map[string]interface{}{"min": float64(20), "max": float64(10)},
			expectErr:   true,
		},
		{
			name:        "missing params",
			args:        map[string]interface{}{"min": float64(20)},
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tool.Execute(ctx, tt.args)
			if (err != nil) != tt.expectErr {
				t.Fatalf("expected error: %v, got: %v", tt.expectErr, err)
			}
			if err == nil {
				resMap := res.(map[string]interface{})
				val := resMap["result"].(int)
				if val < 10 || val >= 20 {
					t.Fatalf("result out of expected range, got %v", val)
				}
			}
		})
	}
}

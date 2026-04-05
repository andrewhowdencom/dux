package stdlib

import (
	"context"
	"testing"
)

func TestMathTool_Execute(t *testing.T) {
	tool := NewMath()
	ctx := context.Background()

	tests := []struct {
		name        string
		args        map[string]interface{}
		expectErr   bool
		expectedVal interface{}
	}{
		{
			name:        "simple addition",
			args:        map[string]interface{}{"expression": "2 + 2"},
			expectErr:   false,
			expectedVal: float64(4),
		},
		{
			name:        "complex equation",
			args:        map[string]interface{}{"expression": "(20 / 4) * 3 + 2"},
			expectErr:   false,
			expectedVal: float64(17),
		},
		{
			name:      "missing expression",
			args:      map[string]interface{}{},
			expectErr: true,
		},
		{
			name:      "invalid expression",
			args:      map[string]interface{}{"expression": "2 + * 4"},
			expectErr: true,
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
				if resMap["result"] != tt.expectedVal {
					t.Fatalf("expected result %v, got %v", tt.expectedVal, resMap["result"])
				}
			}
		})
	}
}

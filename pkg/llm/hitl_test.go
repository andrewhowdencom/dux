package llm_test

import (
	"context"
	"testing"

	"github.com/andrewhowdencom/dux/pkg/llm"
)

type mockHITL struct {
	approved bool
}

func (m *mockHITL) ApproveTool(ctx context.Context, req llm.ToolRequestPart) (bool, error) {
	return m.approved, nil
}

func TestNewHITLMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		namespace      string
		req            llm.ToolRequestPart
		policies       map[string]interface{}
		unsafeAll      bool
		handlerApprove bool
		expectErr      bool
	}{
		{
			name:      "Boolean policy allows auto",
			namespace: "stdlib",
			req:       llm.ToolRequestPart{Name: "calculator"},
			policies: map[string]interface{}{
				"stdlib": false,
			},
			expectErr: false,
		},
		{
			name:      "Boolean policy enforces HITL - approved",
			namespace: "stdlib",
			req:       llm.ToolRequestPart{Name: "calculator"},
			policies: map[string]interface{}{
				"stdlib": true,
			},
			handlerApprove: true,
			expectErr:      false,
		},
		{
			name:      "Boolean policy enforces HITL - denied",
			namespace: "stdlib",
			req:       llm.ToolRequestPart{Name: "calculator"},
			policies: map[string]interface{}{
				"stdlib": true,
			},
			handlerApprove: false,
			expectErr:      true,
		},
		{
			name:      "CEL string policy allows auto",
			namespace: "bash",
			req:       llm.ToolRequestPart{Name: "bash", Args: map[string]interface{}{"command": "ls /tmp"}},
			policies: map[string]interface{}{
				"bash": `!(args.command.startsWith("ls "))`,
			},
			expectErr: false,
		},
		{
			name:      "CEL string policy enforces HITL - denied",
			namespace: "bash",
			req:       llm.ToolRequestPart{Name: "bash", Args: map[string]interface{}{"command": "rm -rf /"}},
			policies: map[string]interface{}{
				"bash": `!(args.command.startsWith("ls "))`,
			},
			handlerApprove: false,
			expectErr:      true,
		},
		{
			name:      "CEL string syntax error defaults to secure (HITL)",
			namespace: "mcp",
			req:       llm.ToolRequestPart{Name: "broken_tool"},
			policies: map[string]interface{}{
				"mcp": `---invalid cel syntax---`,
			},
			handlerApprove: false,
			expectErr:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := &mockHITL{approved: tc.handlerApprove}
			mw := llm.NewHITLMiddleware(handler, tc.policies, tc.unsafeAll)

			// Setup context with namespace
			ctx := context.WithValue(context.Background(), llm.ContextKeyNamespace, tc.namespace)

			nextFunc := func(ctx context.Context) (interface{}, error) {
				return "success", nil
			}

			_, err := mw(ctx, tc.req, nextFunc)

			if tc.expectErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.expectErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

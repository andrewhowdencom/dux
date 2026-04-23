package workspace

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"gopkg.in/yaml.v3"
)

func TestAfterToolHook_PlanCreate(t *testing.T) {
	hook := NewAfterToolHook()
	ctx := llm.WithSessionID(context.Background(), "test-session")

	req := llm.AfterToolRequest{
		ToolCall: llm.ToolRequestPart{
			Name: "plan_create",
			Args: map[string]interface{}{},
		},
		Result: llm.ToolResultPart{
			Result: map[string]string{
				"plan_id": "plan-123",
			},
		},
	}

	if err := hook(ctx, req); err != nil {
		t.Fatalf("hook returned error: %v", err)
	}

	// Verify index was written
	idxPath, err := planIndexPath("test-session")
	if err != nil {
		t.Fatalf("planIndexPath: %v", err)
	}

	b, err := os.ReadFile(idxPath)
	if err != nil {
		t.Fatalf("failed to read index file: %v", err)
	}

	var idx PlanIndex
	if err := yaml.Unmarshal(b, &idx); err != nil {
		t.Fatalf("failed to unmarshal index: %v", err)
	}

	if len(idx.Plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(idx.Plans))
	}

	if idx.Plans[0].PlanID != "plan-123" {
		t.Errorf("expected plan_id=plan-123, got %s", idx.Plans[0].PlanID)
	}
	if idx.Plans[0].Status != PlanStatusDraft {
		t.Errorf("expected status=draft, got %s", idx.Plans[0].Status)
	}
	if idx.Plans[0].ToolAction != "plan_create" {
		t.Errorf("expected tool_action=plan_create, got %s", idx.Plans[0].ToolAction)
	}

	// Cleanup
	os.RemoveAll(filepath.Dir(idxPath))
}

func TestAfterToolHook_PlanApprove(t *testing.T) {
	hook := NewAfterToolHook()
	ctx := llm.WithSessionID(context.Background(), "test-session-approve")

	// First create a plan
	createReq := llm.AfterToolRequest{
		ToolCall: llm.ToolRequestPart{
			Name: "plan_create",
			Args: map[string]interface{}{},
		},
		Result: llm.ToolResultPart{
			Result: map[string]string{"plan_id": "plan-456"},
		},
	}
	if err := hook(ctx, createReq); err != nil {
		t.Fatalf("create hook error: %v", err)
	}

	// Then approve it
	approveReq := llm.AfterToolRequest{
		ToolCall: llm.ToolRequestPart{
			Name: "plan_approve",
			Args: map[string]interface{}{
				"plan_id": "plan-456",
			},
		},
		Result: llm.ToolResultPart{
			Result: map[string]string{"success": "true"},
		},
	}
	if err := hook(ctx, approveReq); err != nil {
		t.Fatalf("approve hook error: %v", err)
	}

	idxPath, err := planIndexPath("test-session-approve")
	if err != nil {
		t.Fatalf("planIndexPath: %v", err)
	}
	b, err := os.ReadFile(idxPath)
	if err != nil {
		t.Fatalf("failed to read index: %v", err)
	}
	var idx PlanIndex
	if err := yaml.Unmarshal(b, &idx); err != nil {
		t.Fatalf("failed to unmarshal index: %v", err)
	}

	if len(idx.Plans) != 1 {
		t.Fatalf("expected 1 plan after upsert, got %d", len(idx.Plans))
	}
	if idx.Plans[0].Status != PlanStatusApproved {
		t.Errorf("expected status=approved after approve, got %s", idx.Plans[0].Status)
	}

	os.RemoveAll(filepath.Dir(idxPath))
}

func TestAfterToolHook_NonPlanToolIgnored(t *testing.T) {
	hook := NewAfterToolHook()
	ctx := llm.WithSessionID(context.Background(), "test-session-ignore")

	req := llm.AfterToolRequest{
		ToolCall: llm.ToolRequestPart{
			Name: "file_read",
		},
		Result: llm.ToolResultPart{
			Result: "some file content",
		},
	}

	if err := hook(ctx, req); err != nil {
		t.Fatalf("hook returned error: %v", err)
	}

	idxPath, _ := planIndexPath("test-session-ignore")
	if _, err := os.Stat(idxPath); !os.IsNotExist(err) {
		t.Errorf("expected no index file for non-plan tool, but file exists")
	}
}

func TestAfterToolHook_NoSessionID(t *testing.T) {
	hook := NewAfterToolHook()
	// No session ID in context
	ctx := context.Background()

	req := llm.AfterToolRequest{
		ToolCall: llm.ToolRequestPart{Name: "plan_create"},
	}

	if err := hook(ctx, req); err != nil {
		t.Fatalf("expected nil error when no sessionID, got %v", err)
	}
}

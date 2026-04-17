package workspace

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/adrg/xdg"
	"github.com/andrewhowdencom/dux/pkg/llm"
)

func TestPlanCreateTool(t *testing.T) {
	tmpDir := t.TempDir()
	originalDataHome := xdg.DataHome
	xdg.DataHome = tmpDir
	defer func() { xdg.DataHome = originalDataHome }()

	tool := &PlanCreateTool{}
	if tool.Name() != "plan_create" {
		t.Fatalf("expected name plan_create, got %s", tool.Name())
	}

	ctx := llm.WithSessionID(context.Background(), "test-session")

	args := map[string]interface{}{
		"title":   "Test Plan",
		"content": "This is a test plan.",
	}

	result, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultMap, ok := result.(map[string]string)
	if !ok {
		t.Fatalf("expected map[string]string result, got %T", result)
	}

	if resultMap["success"] != "true" {
		t.Fatalf("expected success=true, got %s", resultMap["success"])
	}

	planID := resultMap["plan_id"]
	if planID == "" {
		t.Fatal("expected non-empty plan_id")
	}

	planPath := filepath.Join(tmpDir, "dux", "sessions", "test-session", "workspace", "plans", planID+".md")
	data, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatalf("failed to read plan file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "status: draft") {
		t.Fatal("expected plan to have draft status in frontmatter")
	}
	if !strings.Contains(content, "# Test Plan") {
		t.Fatal("expected plan to have title")
	}
	if !strings.Contains(content, "This is a test plan.") {
		t.Fatal("expected plan to have content")
	}
}

func TestPlanApproveTool(t *testing.T) {
	tmpDir := t.TempDir()
	originalDataHome := xdg.DataHome
	xdg.DataHome = tmpDir
	defer func() { xdg.DataHome = originalDataHome }()

	createTool := &PlanCreateTool{}
	ctx := llm.WithSessionID(context.Background(), "test-session")

	_, err := createTool.Execute(ctx, map[string]interface{}{
		"title":   "Test Plan",
		"content": "Test content",
	})
	if err != nil {
		t.Fatalf("failed to create plan: %v", err)
	}

	planDir := filepath.Join(tmpDir, "dux", "sessions", "test-session", "workspace", "plans")
	files, err := os.ReadDir(planDir)
	if err != nil {
		t.Fatalf("failed to read plan dir: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 plan file, got %d", len(files))
	}
	planID := strings.TrimSuffix(files[0].Name(), ".md")

	approveTool := &PlanApproveTool{}
	result, err := approveTool.Execute(ctx, map[string]interface{}{
		"plan_id": planID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultMap, ok := result.(map[string]string)
	if !ok {
		t.Fatalf("expected map[string]string result, got %T", result)
	}

	if resultMap["success"] != "true" {
		t.Fatalf("expected success=true, got %s", resultMap["success"])
	}

	planPath := filepath.Join(planDir, planID+".md")
	data, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatalf("failed to read plan file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "status: approved") {
		t.Fatal("expected plan to have approved status")
	}
	if !strings.Contains(content, "approved_at:") {
		t.Fatal("expected plan to have approved_at timestamp")
	}
}

func TestPlanUpdateToolPreservesMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	originalDataHome := xdg.DataHome
	xdg.DataHome = tmpDir
	defer func() { xdg.DataHome = originalDataHome }()

	createTool := &PlanCreateTool{}
	ctx := llm.WithSessionID(context.Background(), "test-session")

	_, err := createTool.Execute(ctx, map[string]interface{}{
		"title":   "Test Plan",
		"content": "Original content",
	})
	if err != nil {
		t.Fatalf("failed to create plan: %v", err)
	}

	planDir := filepath.Join(tmpDir, "dux", "sessions", "test-session", "workspace", "plans")
	files, err := os.ReadDir(planDir)
	if err != nil {
		t.Fatalf("failed to read plan dir: %v", err)
	}
	planID := strings.TrimSuffix(files[0].Name(), ".md")

	updateTool := &PlanUpdateTool{}
	_, err = updateTool.Execute(ctx, map[string]interface{}{
		"plan_id": planID,
		"content": "Updated content",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	planPath := filepath.Join(planDir, planID+".md")
	data, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatalf("failed to read plan file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "status: draft") {
		t.Fatal("expected plan to preserve draft status after update")
	}
	if !strings.Contains(content, "Updated content") {
		t.Fatal("expected plan to have updated content")
	}
	if strings.Contains(content, "Original content") {
		t.Fatal("expected plan to not have original content")
	}
}

func TestReadPlanMetadata(t *testing.T) {
	tmpDir := t.TempDir()

	metadata := PlanMetadata{
		Status:    PlanStatusDraft,
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	content, err := writePlanWithMetadata("Test Plan", "Test content", metadata)
	if err != nil {
		t.Fatalf("failed to write plan: %v", err)
	}

	filePath := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	readMetadata, err := readPlanMetadata(filePath)
	if err != nil {
		t.Fatalf("failed to read metadata: %v", err)
	}

	if readMetadata.Status != PlanStatusDraft {
		t.Fatalf("expected status draft, got %s", readMetadata.Status)
	}
	if readMetadata.CreatedAt.Unix() != metadata.CreatedAt.Unix() {
		t.Fatalf("expected CreatedAt %v, got %v", metadata.CreatedAt, readMetadata.CreatedAt)
	}
}

func TestUpdatePlanStatus(t *testing.T) {
	tmpDir := t.TempDir()

	metadata := PlanMetadata{
		Status:    PlanStatusDraft,
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	content, err := writePlanWithMetadata("Test Plan", "Test content", metadata)
	if err != nil {
		t.Fatalf("failed to write plan: %v", err)
	}

	filePath := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	approvedAt := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	if err := updatePlanStatus(filePath, PlanStatusApproved, &approvedAt); err != nil {
		t.Fatalf("failed to update status: %v", err)
	}

	updatedMetadata, err := readPlanMetadata(filePath)
	if err != nil {
		t.Fatalf("failed to read updated metadata: %v", err)
	}

	if updatedMetadata.Status != PlanStatusApproved {
		t.Fatalf("expected status approved, got %s", updatedMetadata.Status)
	}
	if updatedMetadata.ApprovedAt == nil {
		t.Fatal("expected approved_at to be set")
	}
	if updatedMetadata.ApprovedAt.Unix() != approvedAt.Unix() {
		t.Fatalf("expected approved_at %v, got %v", approvedAt, updatedMetadata.ApprovedAt)
	}
}

func TestWritePlanWithMetadata(t *testing.T) {
	metadata := PlanMetadata{
		Status:    PlanStatusDraft,
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	content, err := writePlanWithMetadata("My Plan", "Plan body here", metadata)
	if err != nil {
		t.Fatalf("failed to write plan: %v", err)
	}

	if !strings.HasPrefix(content, "---\n") {
		t.Fatal("expected content to start with ---")
	}
	if !strings.Contains(content, "status: draft") {
		t.Fatal("expected content to contain status")
	}
	if !strings.Contains(content, "# My Plan") {
		t.Fatal("expected content to contain title")
	}
	if !strings.Contains(content, "Plan body here") {
		t.Fatal("expected content to contain body")
	}
}

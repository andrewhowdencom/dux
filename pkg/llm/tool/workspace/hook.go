package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/andrewhowdencom/dux/pkg/llm"
	"gopkg.in/yaml.v3"
)

// PlanIndexEntry tracks the metadata for a single plan in the session index.
type PlanIndexEntry struct {
	PlanID       string    `yaml:"plan_id"`
	Status       string    `yaml:"status"`
	LastModified time.Time `yaml:"last_modified"`
	ToolAction   string    `yaml:"tool_action"`
}

// PlanIndex is the top-level structure written to the session workspace.
type PlanIndex struct {
	UpdatedAt time.Time        `yaml:"updated_at"`
	Plans     []PlanIndexEntry `yaml:"plans"`
}

// NewAfterToolHook returns a hook that maintains a plans-index.yaml in the
// session workspace whenever a plan tool is executed.  This gives the LLM a
// quick overview of all plans without needing to list files.
func NewAfterToolHook() llm.AfterToolHook {
	return func(ctx context.Context, req llm.AfterToolRequest) error {
		if !strings.HasPrefix(req.ToolCall.Name, "plan_") {
			return nil
		}

		sessionID, err := llm.SessionIDFromContext(ctx)
		if err != nil || sessionID == "" {
			return nil // best-effort: no session, no index
		}

		idxPath, err := planIndexPath(sessionID)
		if err != nil {
			return nil // best-effort
		}

		index, err := readPlanIndex(idxPath)
		if err != nil {
			index = &PlanIndex{}
		}

		// Derive plan ID and status from the tool call / result.
		entry := PlanIndexEntry{
			LastModified: time.Now().UTC(),
			ToolAction:   req.ToolCall.Name,
		}

		// Try to extract plan_id from tool arguments.
		if id, ok := req.ToolCall.Args["plan_id"].(string); ok && id != "" {
			entry.PlanID = id
		}
		// For plan_create the result contains the new plan_id.
		if entry.PlanID == "" && !req.Result.IsError {
			entry.PlanID = extractPlanIDFromResult(req.Result.Result)
		}

		// Infer status from the tool name.
		switch req.ToolCall.Name {
		case "plan_create":
			entry.Status = PlanStatusDraft
		case "plan_approve":
			entry.Status = PlanStatusApproved
		default:
			// For update / read keep existing status if known.
			entry.Status = inferStatusFromResult(req.Result)
		}

		// Upsert into the index.
		found := false
		for i, p := range index.Plans {
			if p.PlanID == entry.PlanID && entry.PlanID != "" {
				index.Plans[i] = entry
				found = true
				break
			}
		}
		if !found {
			index.Plans = append(index.Plans, entry)
		}
		index.UpdatedAt = time.Now().UTC()

		return writePlanIndex(idxPath, index)
	}
}

func planIndexPath(sessionID string) (string, error) {
	dir := filepath.Join(xdg.DataHome, "dux", "sessions", sessionID, "workspace", "plans")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "plans-index.yaml"), nil
}

func readPlanIndex(path string) (*PlanIndex, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var idx PlanIndex
	if err := yaml.Unmarshal(b, &idx); err != nil {
		return nil, err
	}
	return &idx, nil
}

func writePlanIndex(path string, idx *PlanIndex) error {
	b, err := yaml.Marshal(idx)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}

func extractPlanIDFromResult(res interface{}) string {
	// The plan_create tool returns map[string]string{"plan_id": ...}
	if m, ok := res.(map[string]string); ok {
		return m["plan_id"]
	}
	if m, ok := res.(map[string]interface{}); ok {
		if id, ok := m["plan_id"].(string); ok {
			return id
		}
	}
	return ""
}

func inferStatusFromResult(result llm.ToolResultPart) string {
	if result.IsError {
		return "error"
	}
	// Best-effort: look for status words in the string representation.
	s := fmt.Sprintf("%v", result.Result)
	if strings.Contains(s, PlanStatusApproved) {
		return PlanStatusApproved
	}
	if strings.Contains(s, PlanStatusComplete) {
		return PlanStatusComplete
	}
	return PlanStatusDraft
}

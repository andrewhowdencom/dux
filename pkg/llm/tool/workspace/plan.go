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
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

const (
	PlanStatusDraft    = "draft"
	PlanStatusApproved = "approved"
	PlanStatusComplete = "complete"
)

type PlanMetadata struct {
	Status     string     `yaml:"status"`
	CreatedAt  time.Time  `yaml:"created_at"`
	ApprovedAt *time.Time `yaml:"approved_at,omitempty"`
}

// Provider implements the plan toolbox mapping down to the active Workspace Dir.
type Provider struct{}

func NewProvider() *Provider {
	return &Provider{}
}

func (p *Provider) Namespace() string {
	return "workspace_plans"
}

func (p *Provider) Inject(ctx context.Context, q llm.InjectQuery) ([]llm.Message, error) {
	return nil, nil
}

func (p *Provider) GetTool(name string) (llm.Tool, bool) {
	switch name {
	case "plan_create":
		return &PlanCreateTool{}, true
	case "plan_read":
		return &PlanReadTool{}, true
	case "plan_update":
		return &PlanUpdateTool{}, true
	case "plan_list":
		return &PlanListTool{}, true
	case "plan_approve":
		return &PlanApproveTool{}, true
	}
	return nil, false
}

func getWorkspacePath(ctx context.Context) (string, error) {
	sessionID, err := llm.SessionIDFromContext(ctx)
	if err != nil || sessionID == "" {
		return "", fmt.Errorf("agent operation requires an active session ID")
	}
	dir := filepath.Join(xdg.DataHome, "dux", "sessions", sessionID, "workspace", "plans")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

// === plan_create ===

type PlanCreateTool struct{}

func (t *PlanCreateTool) Name() string { return "plan_create" }

func (t *PlanCreateTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        t.Name(),
		Description: "Create a new persistent architectural plan document in the session workspace.",
		Parameters: []byte(`{
			"type": "object",
			"properties": {
				"title": { "type": "string" },
				"content": { "type": "string", "description": "The full markdown content of the plan." }
			},
			"required": ["title", "content"]
		}`),
	}
}

func (t *PlanCreateTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	title, _ := args["title"].(string)
	content, _ := args["content"].(string)

	if content == "" {
		return nil, fmt.Errorf("content is required")
	}

	basePath, err := getWorkspacePath(ctx)
	if err != nil {
		return nil, err
	}

	planID := uuid.New().String()
	filePath := filepath.Join(basePath, planID+".md")

	metadata := PlanMetadata{
		Status:    PlanStatusDraft,
		CreatedAt: time.Now().UTC(),
	}

	fullContent, err := writePlanWithMetadata(title, content, metadata)
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(filePath, []byte(fullContent), 0600); err != nil {
		return nil, err
	}

	return map[string]string{
		"success": "true",
		"plan_id": planID,
		"message": "Plan successfully written to session workspace memory with draft status.",
	}, nil
}

// === plan_read ===

type PlanReadTool struct{}

func (t *PlanReadTool) Name() string { return "plan_read" }

func (t *PlanReadTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        t.Name(),
		Description: "Read an existing architectural plan from the session workspace.",
		Parameters: []byte(`{
			"type": "object",
			"properties": {
				"plan_id": { "type": "string" }
			},
			"required": ["plan_id"]
		}`),
	}
}

func (t *PlanReadTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	planID, _ := args["plan_id"].(string)
	if planID == "" {
		return nil, fmt.Errorf("plan_id is required")
	}

	basePath, err := getWorkspacePath(ctx)
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(basePath, planID+".md")
	b, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return string(b), nil
}

// === plan_update ===

type PlanUpdateTool struct{}

func (t *PlanUpdateTool) Name() string { return "plan_update" }

func (t *PlanUpdateTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        t.Name(),
		Description: "Overwrite an existing architectural plan to mark off completed chunks or rectify architectural flaws.",
		Parameters: []byte(`{
			"type": "object",
			"properties": {
				"plan_id": { "type": "string" },
				"content": { "type": "string", "description": "The complete markdown content to save." }
			},
			"required": ["plan_id", "content"]
		}`),
	}
}

func (t *PlanUpdateTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	planID, _ := args["plan_id"].(string)
	content, _ := args["content"].(string)

	if planID == "" || content == "" {
		return nil, fmt.Errorf("plan_id and content are required")
	}

	basePath, err := getWorkspacePath(ctx)
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(basePath, planID+".md")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("plan %s does not exist", planID)
	}

	existingMetadata, err := readPlanMetadata(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read existing plan metadata: %w", err)
	}

	fullContent, err := writePlanWithMetadataFromExisting(content, existingMetadata)
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(filePath, []byte(fullContent), 0600); err != nil {
		return nil, err
	}

	return "Plan updated successfully.", nil
}

// === plan_list ===

type PlanListTool struct{}

func (t *PlanListTool) Name() string { return "plan_list" }

func (t *PlanListTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        t.Name(),
		Description: "List all existing plans stored in the session workspace.",
		Parameters: []byte(`{
			"type": "object",
			"properties": {}
		}`),
	}
}

func (t *PlanListTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	basePath, err := getWorkspacePath(ctx)
	if err != nil {
		return nil, err
	}

	files, err := os.ReadDir(basePath)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	var planIDs []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".md") {
			planIDs = append(planIDs, strings.TrimSuffix(f.Name(), ".md"))
		}
	}

	if len(planIDs) == 0 {
		return "No plans found.", nil
	}
	return map[string]interface{}{
		"plans": planIDs,
	}, nil
}

type PlanApproveTool struct{}

func (t *PlanApproveTool) Name() string { return "plan_approve" }

func (t *PlanApproveTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        t.Name(),
		Description: "Mark an existing plan as approved after user review.",
		Parameters: []byte(`{
			"type": "object",
			"properties": {
				"plan_id": { "type": "string", "description": "The ID of the plan to approve." }
			},
			"required": ["plan_id"]
		}`),
	}
}

func (t *PlanApproveTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	planID, _ := args["plan_id"].(string)
	if planID == "" {
		return nil, fmt.Errorf("plan_id is required")
	}

	basePath, err := getWorkspacePath(ctx)
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(basePath, planID+".md")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("plan %s does not exist", planID)
	}

	now := time.Now().UTC()
	if err := updatePlanStatus(filePath, PlanStatusApproved, &now); err != nil {
		return nil, err
	}

	return map[string]string{
		"success": "true",
		"message": fmt.Sprintf("Plan %s has been approved.", planID),
	}, nil
}

func writePlanWithMetadata(title, content string, metadata PlanMetadata) (string, error) {
	metaBytes, err := yaml.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("failed to marshal plan metadata: %w", err)
	}

	return fmt.Sprintf("---\n%s---\n# %s\n\n%s", string(metaBytes), title, content), nil
}

func writePlanWithMetadataFromExisting(content string, metadata PlanMetadata) (string, error) {
	metaBytes, err := yaml.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("failed to marshal plan metadata: %w", err)
	}

	if strings.HasPrefix(content, "---\n") {
		endIdx := strings.Index(content[4:], "---\n")
		if endIdx != -1 {
			content = strings.TrimSpace(content[4+endIdx+4:])
		}
	}

	lines := strings.SplitN(content, "\n", 2)
	var title string
	if len(lines) > 0 && strings.HasPrefix(lines[0], "# ") {
		title = strings.TrimPrefix(lines[0], "# ")
		content = strings.TrimPrefix(content, lines[0]+"\n")
	} else {
		title = "Untitled Plan"
	}

	return fmt.Sprintf("---\n%s---\n# %s\n\n%s", string(metaBytes), title, content), nil
}

func readPlanMetadata(filePath string) (PlanMetadata, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return PlanMetadata{}, err
	}

	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		return PlanMetadata{
			Status:    PlanStatusDraft,
			CreatedAt: time.Now().UTC(),
		}, nil
	}

	endIdx := strings.Index(content[4:], "---\n")
	if endIdx == -1 {
		return PlanMetadata{
			Status:    PlanStatusDraft,
			CreatedAt: time.Now().UTC(),
		}, nil
	}

	yamlContent := content[4 : 4+endIdx]
	var metadata PlanMetadata
	if err := yaml.Unmarshal([]byte(yamlContent), &metadata); err != nil {
		return PlanMetadata{}, fmt.Errorf("failed to parse plan metadata: %w", err)
	}

	return metadata, nil
}

func updatePlanStatus(filePath string, newStatus string, approvedAt *time.Time) error {
	metadata, err := readPlanMetadata(filePath)
	if err != nil {
		return err
	}

	metadata.Status = newStatus
	if approvedAt != nil {
		metadata.ApprovedAt = approvedAt
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	content := string(data)
	var bodyContent string
	if strings.HasPrefix(content, "---\n") {
		endIdx := strings.Index(content[4:], "---\n")
		if endIdx != -1 {
			bodyContent = strings.TrimSpace(content[4+endIdx+4:])
		} else {
			bodyContent = content
		}
	} else {
		bodyContent = content
	}

	lines := strings.SplitN(bodyContent, "\n", 2)
	var title string
	if len(lines) > 0 && strings.HasPrefix(lines[0], "# ") {
		title = strings.TrimPrefix(lines[0], "# ")
		bodyContent = strings.TrimPrefix(bodyContent, lines[0]+"\n")
	} else {
		title = "Untitled Plan"
	}

	fullContent, err := writePlanWithMetadata(title, bodyContent, metadata)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, []byte(fullContent), 0600)
}

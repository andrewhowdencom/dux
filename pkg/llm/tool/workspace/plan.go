package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/google/uuid"
)

// Provider implements the plan toolbox mapping down to the active Workspace Dir.
type Provider struct {}

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

type PlanCreateTool struct {}

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
	
	fullContent := fmt.Sprintf("# %s\n\n%s", title, content)
	if err := os.WriteFile(filePath, []byte(fullContent), 0600); err != nil {
		return nil, err
	}

	return map[string]string{
		"success": "true",
		"plan_id": planID,
		"message": "Plan successfully written to session workspace memory.",
	}, nil
}

// === plan_read ===

type PlanReadTool struct {}

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

type PlanUpdateTool struct {}

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

	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		return nil, err
	}

	return "Plan updated successfully.", nil
}

// === plan_list ===

type PlanListTool struct {}

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

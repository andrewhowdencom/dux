package semantic

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/memory/semantic"
)

type ValidateTool struct {
	service *semantic.Service
}

func NewValidateTool(service *semantic.Service) *ValidateTool {
	return &ValidateTool{service: service}
}

func (t *ValidateTool) Name() string {
	return "semantic_validate"
}

func (t *ValidateTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        "semantic_validate",
		Description: "Mark a fact as validated, updating its validated_at timestamp.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"required": ["id"],
			"properties": {
				"id": {"type": "string", "description": "The fact ID to validate"}
			}
		}`),
	}
}

func (t *ValidateTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)

	if id == "" {
		return nil, fmt.Errorf("missing required argument: id")
	}

	err := t.service.ValidateFact(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to validate fact: %w", err)
	}

	return map[string]string{"status": "validated", "id": id}, nil
}

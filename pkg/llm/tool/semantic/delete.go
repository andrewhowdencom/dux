package semantic

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/memory/semantic"
)

type DeleteTool struct {
	service *semantic.Service
}

func NewDeleteTool(service *semantic.Service) *DeleteTool {
	return &DeleteTool{service: service}
}

func (t *DeleteTool) Name() string {
	return "semantic_delete"
}

func (t *DeleteTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        "semantic_delete",
		Description: "Delete a fact from long-term memory by ID.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"required": ["id"],
			"properties": {
				"id": {"type": "string", "description": "The fact ID to delete"}
			}
		}`),
	}
}

func (t *DeleteTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)

	if id == "" {
		return nil, fmt.Errorf("missing required argument: id")
	}

	err := t.service.DeleteFact(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete fact: %w", err)
	}

	return map[string]string{"status": "deleted", "id": id}, nil
}

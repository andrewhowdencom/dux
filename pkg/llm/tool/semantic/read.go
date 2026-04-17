package semantic

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/memory/semantic"
)

type ReadTool struct {
	service *semantic.Service
}

func NewReadTool(service *semantic.Service) *ReadTool {
	return &ReadTool{service: service}
}

func (t *ReadTool) Name() string {
	return "semantic_read"
}

func (t *ReadTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        "semantic_read",
		Description: "Read a specific fact by ID from long-term memory.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"required": ["id"],
			"properties": {
				"id": {"type": "string", "description": "The fact ID to read"}
			}
		}`),
	}
}

func (t *ReadTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	id, _ := args["id"].(string)

	if id == "" {
		return nil, fmt.Errorf("missing required argument: id")
	}

	fact, err := t.service.ReadFact(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to read fact: %w", err)
	}

	return fact, nil
}

package stdlib

import (
	"context"
	"encoding/json"
	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/google/uuid"
)

type UUIDTool struct{}

func NewUUID() llm.Tool {
	return &UUIDTool{}
}

func (t *UUIDTool) Name() string { return "generate_uuid" }

func (t *UUIDTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        t.Name(),
		Description: "Generates a globally unique identifier (UUID v4) for mock data, database insertion, tracing, etc.",
		Parameters:  json.RawMessage(`{"type":"object","properties":{}}`),
	}
}

func (t *UUIDTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{
		"uuid": uuid.NewString(),
	}, nil
}

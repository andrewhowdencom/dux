package semantic

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/memory/semantic"
)

type CreateRelationshipTool struct {
	service *semantic.Service
}

func NewCreateRelationshipTool(service *semantic.Service) *CreateRelationshipTool {
	return &CreateRelationshipTool{service: service}
}

func (t *CreateRelationshipTool) Name() string {
	return "semantic_create_relationship"
}

func (t *CreateRelationshipTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        "semantic_create_relationship",
		Description: "Create a relationship edge between two entities in the knowledge graph.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"required": ["subject", "predicate", "object"],
			"properties": {
				"subject": {"type": "string", "description": "The source entity (e.g., 'person:john-doe')"},
				"predicate": {"type": "string", "description": "The relationship type (e.g., 'has_condition', 'works_at')"},
				"object": {"type": "string", "description": "The target entity (e.g., 'condition:pvcs')"}
			}
		}`),
	}
}

func (t *CreateRelationshipTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	subject, _ := args["subject"].(string)
	predicate, _ := args["predicate"].(string)
	object, _ := args["object"].(string)

	if subject == "" || predicate == "" || object == "" {
		return nil, fmt.Errorf("missing required arguments: subject, predicate, or object")
	}

	err := t.service.CreateRelationship(ctx, subject, predicate, object)
	if err != nil {
		return nil, fmt.Errorf("failed to create relationship: %w", err)
	}

	return map[string]string{
		"subject":   subject,
		"predicate": predicate,
		"object":    object,
		"status":    "created",
	}, nil
}

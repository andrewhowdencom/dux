package semantic

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/memory/semantic"
	"github.com/google/uuid"
)

type WriteTripleTool struct {
	service *semantic.Service
}

func NewWriteTripleTool(service *semantic.Service) *WriteTripleTool {
	return &WriteTripleTool{service: service}
}

func (t *WriteTripleTool) Name() string {
	return "semantic_write_triple"
}

func (t *WriteTripleTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        "semantic_write_triple",
		Description: "Save a structured fact (entity-attribute-value triple) to long-term memory.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"required": ["entity", "attribute", "value"],
			"properties": {
				"entity": {"type": "string", "description": "The subject of the fact"},
				"attribute": {"type": "string", "description": "The property being set"},
				"value": {"type": "string", "description": "The value to remember"},
				"tags": {"type": "array", "items": {"type": "string"}, "description": "Optional tags"},
				"source_uri": {"type": "string", "description": "URI of the source"},
				"constraints": {"type": "object", "additionalProperties": {"type": "string"}, "description": "Optional key-value constraints for fact relevance"}
			}
		}`),
	}
}

func (t *WriteTripleTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	entity, _ := args["entity"].(string)
	attribute, _ := args["attribute"].(string)
	value, _ := args["value"].(string)

	if entity == "" || attribute == "" || value == "" {
		return nil, fmt.Errorf("missing required arguments: entity, attribute, or value")
	}

	var tags []string
	if t, ok := args["tags"].([]interface{}); ok {
		for _, v := range t {
			if s, ok := v.(string); ok {
				tags = append(tags, s)
			}
		}
	}

	var constraintsMap map[string]string
	if c, ok := args["constraints"].(map[string]interface{}); ok {
		constraintsMap = make(map[string]string)
		for k, v := range c {
			if s, ok := v.(string); ok {
				constraintsMap[k] = s
			}
		}
	}

	sourceURI, _ := args["source_uri"].(string)
	sources := []semantic.Source{}
	if sourceURI != "" {
		sources = append(sources, semantic.Source{URI: sourceURI, RetrievedAt: time.Now()})
	}

	id := uuid.New().String()
	fact := semantic.TripleFact{
		ID:        id,
		Entity:    entity,
		Attribute: attribute,
		Value:     value,
		Sources:   sources,
		Tags:      tags,
		Metadata: semantic.FactMetadata{
			CreatedAt:    time.Now(),
			ValidatedAt:  time.Now(),
			LastAccessed: time.Now(),
			Constraints:  constraintsMap,
		},
	}

	err := t.service.WriteTriple(ctx, fact)
	if err != nil {
		return nil, fmt.Errorf("failed to write triple: %w", err)
	}

	return map[string]string{"id": id, "status": "saved"}, nil
}

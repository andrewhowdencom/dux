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

type WriteStatementTool struct {
	service *semantic.Service
}

func NewWriteStatementTool(service *semantic.Service) *WriteStatementTool {
	return &WriteStatementTool{service: service}
}

func (t *WriteStatementTool) Name() string {
	return "semantic_write_statement"
}

func (t *WriteStatementTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        "semantic_write_statement",
		Description: "Save a freeform statement to long-term memory.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"required": ["statement"],
			"properties": {
				"statement": {"type": "string", "description": "The statement to remember"},
				"tags": {"type": "array", "items": {"type": "string"}, "description": "Optional tags"},
				"source_uri": {"type": "string", "description": "URI of the source"},
				"constraints": {"type": "object", "additionalProperties": {"type": "string"}, "description": "Optional key-value constraints for fact relevance"}
			}
		}`),
	}
}

func (t *WriteStatementTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	statement, _ := args["statement"].(string)

	if statement == "" {
		return nil, fmt.Errorf("missing required argument: statement")
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
	fact := semantic.StatementFact{
		ID:        id,
		Statement: statement,
		Sources:   sources,
		Tags:      tags,
		Metadata: semantic.FactMetadata{
			CreatedAt:    time.Now(),
			ValidatedAt:  time.Now(),
			LastAccessed: time.Now(),
			Constraints:  constraintsMap,
		},
	}

	err := t.service.WriteStatement(ctx, fact)
	if err != nil {
		return nil, fmt.Errorf("failed to write statement: %w", err)
	}

	return map[string]string{"id": id, "status": "saved"}, nil
}

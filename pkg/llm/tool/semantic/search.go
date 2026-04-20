package semantic

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/memory/semantic"
)

type SearchTool struct {
	service *semantic.Service
}

func NewSearchTool(service *semantic.Service) *SearchTool {
	return &SearchTool{service: service}
}

func (t *SearchTool) Name() string {
	return "semantic_search"
}

func (t *SearchTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        "semantic_search",
		Description: "Search long-term memory for facts matching criteria.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"type": {"type": "string", "description": "Filter by fact type (triple or statement)"},
				"entity": {"type": "string", "description": "Filter by entity"},
				"attribute": {"type": "string", "description": "Filter by attribute"},
				"value": {"type": "string", "description": "Filter by value"},
				"tag": {"type": "string", "description": "Filter by tag"},
				"statement": {"type": "string", "description": "Search in statements"},
				"constraints": {"type": "object", "description": "Filter by fact constraints"},
				"limit": {"type": "integer", "description": "Max results"},
				"sort_by": {"type": "string", "description": "Sort field"},
				"sort_order": {"type": "string", "description": "Sort order (asc or desc)"}
			}
		}`),
	}
}

func (t *SearchTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	query := semantic.SearchQuery{}

	if t, ok := args["type"].(string); ok {
		ft := semantic.FactType(t)
		query.Type = &ft
	}
	if e, ok := args["entity"].(string); ok {
		query.Entity = &e
	}
	if a, ok := args["attribute"].(string); ok {
		query.Attribute = &a
	}
	if v, ok := args["value"].(string); ok {
		query.Value = &v
	}
	if tag, ok := args["tag"].(string); ok {
		query.Tag = &tag
	}
	if s, ok := args["statement"].(string); ok {
		query.Statement = &s
	}
	if c, ok := args["constraints"].(map[string]interface{}); ok {
		query.Constraints = make(map[string]string)
		for k, v := range c {
			if vs, ok := v.(string); ok {
				query.Constraints[k] = vs
			}
		}
	}
	if l, ok := args["limit"].(float64); ok {
		query.Limit = int(l)
	}
	if sb, ok := args["sort_by"].(string); ok {
		query.SortBy = semantic.SortField(sb)
	}
	if so, ok := args["sort_order"].(string); ok {
		query.SortOrder = semantic.SortOrder(so)
	}

	facts, err := t.service.Search(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search facts: %w", err)
	}

	return map[string]interface{}{"facts": facts, "count": len(facts)}, nil
}

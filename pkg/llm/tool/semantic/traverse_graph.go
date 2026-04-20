package semantic

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/memory/semantic"
)

type TraverseGraphTool struct {
	service *semantic.Service
}

func NewTraverseGraphTool(service *semantic.Service) *TraverseGraphTool {
	return &TraverseGraphTool{service: service}
}

func (t *TraverseGraphTool) Name() string {
	return "semantic_traverse_graph"
}

func (t *TraverseGraphTool) Definition() llm.ToolDefinitionPart {
	return llm.ToolDefinitionPart{
		Name:        "semantic_traverse_graph",
		Description: "Traverse the knowledge graph from a starting entity to find related facts and entities.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"required": ["start_entity"],
			"properties": {
				"start_entity": {"type": "string", "description": "The entity to start traversal from (e.g., 'person:john-doe')"},
				"predicates": {"type": "array", "items": {"type": "string"}, "description": "Optional filter for relationship types to follow"},
				"max_depth": {"type": "integer", "description": "Maximum traversal depth (default: 3)"},
				"max_results": {"type": "integer", "description": "Maximum number of nodes to return (default: 50)"}
			}
		}`),
	}
}

func (t *TraverseGraphTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	startEntity, _ := args["start_entity"].(string)
	if startEntity == "" {
		return nil, fmt.Errorf("missing required argument: start_entity")
	}

	maxDepth := 3
	if d, ok := args["max_depth"].(float64); ok {
		maxDepth = int(d)
	}

	maxResults := 50
	if r, ok := args["max_results"].(float64); ok {
		maxResults = int(r)
	}

	var predicates []string
	if p, ok := args["predicates"].([]interface{}); ok {
		for _, v := range p {
			if s, ok := v.(string); ok {
				predicates = append(predicates, s)
			}
		}
	}

	result, err := t.service.FindRelatedFactsWithPredicates(ctx, startEntity, maxDepth, predicates)
	if err != nil {
		return nil, fmt.Errorf("failed to traverse graph: %w", err)
	}

	if maxResults > 0 && len(result.Nodes) > maxResults {
		result.Nodes = result.Nodes[:maxResults]
	}

	return map[string]interface{}{
		"start_entity": startEntity,
		"nodes":        result.Nodes,
		"edges":        result.Edges,
		"node_count":   len(result.Nodes),
		"edge_count":   len(result.Edges),
	}, nil
}

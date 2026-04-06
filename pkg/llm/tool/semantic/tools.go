package semantic

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/memory/semantic"
)

// Provider implements llm.ToolProvider for semantic memory tools.
type Provider struct {
	store semantic.Store
	tools map[string]llm.Tool
}

// NewProvider creates a new semantic tools provider.
func NewProvider(store semantic.Store) *Provider {
	p := &Provider{
		store: store,
		tools: make(map[string]llm.Tool),
	}

	writeDef := llm.ToolDefinitionPart{
		Name:        "semantic_write",
		Description: "Save a persistent fact about the user or current project to long-term memory for later recall.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"required": ["entity", "attribute", "value"],
			"properties": {
				"entity": {
					"type": "string",
					"description": "The target subject of this fact (e.g., 'user', 'project', 'system')."
				},
				"attribute": {
					"type": "string",
					"description": "The specific feature or preference being saved (e.g., 'theme', 'language')."
				},
				"value": {
					"type": "string",
					"description": "The value to remember for this entity and attribute."
				}
			}
		}`),
	}

	readDef := llm.ToolDefinitionPart{
		Name:        "semantic_read",
		Description: "Read a specific persistent fact from long-term memory by entity and attribute.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"required": ["entity", "attribute"],
			"properties": {
				"entity": {
					"type": "string",
					"description": "The target subject to recall."
				},
				"attribute": {
					"type": "string",
					"description": "The specific feature or preference to read."
				}
			}
		}`),
	}

	searchDef := llm.ToolDefinitionPart{
		Name:        "semantic_search",
		Description: "Search persistent long-term memory for all entities matching a specific attribute and value.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"required": ["attribute", "value"],
			"properties": {
				"attribute": {
					"type": "string",
					"description": "The specific feature or preference to query."
				},
				"value": {
					"type": "string",
					"description": "The exact value to match against."
				}
			}
		}`),
	}

	deleteDef := llm.ToolDefinitionPart{
		Name:        "semantic_delete",
		Description: "Delete a specific persistent fact from long-term memory.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"required": ["entity", "attribute"],
			"properties": {
				"entity": {
					"type": "string",
					"description": "The target subject of the fact to remove."
				},
				"attribute": {
					"type": "string",
					"description": "The specific feature or preference to remove."
				}
			}
		}`),
	}

	p.tools["semantic_write"] = &genericTool{
		def: writeDef,
		execute: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			entity, _ := args["entity"].(string)
			attribute, _ := args["attribute"].(string)
			value, _ := args["value"].(string)

			if entity == "" || attribute == "" || value == "" {
				return nil, fmt.Errorf("missing required arguments: entity, attribute, or value")
			}

			err := p.store.Write(ctx, semantic.Fact{
				Entity:    entity,
				Attribute: attribute,
				Value:     value,
			})
			if err != nil {
				return nil, err
			}
			return map[string]string{"result": "success", "message": "Fact saved to semantic memory."}, nil
		},
	}

	p.tools["semantic_read"] = &genericTool{
		def: readDef,
		execute: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			entity, _ := args["entity"].(string)
			attribute, _ := args["attribute"].(string)

			if entity == "" || attribute == "" {
				return nil, fmt.Errorf("missing required arguments: entity or attribute")
			}

			fact, err := p.store.Read(ctx, entity, attribute)
			if err != nil {
				return nil, err
			}
			return fact, nil
		},
	}

	p.tools["semantic_search"] = &genericTool{
		def: searchDef,
		execute: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			attribute, _ := args["attribute"].(string)
			value, _ := args["value"].(string)

			if attribute == "" || value == "" {
				return nil, fmt.Errorf("missing required arguments: attribute or value")
			}

			facts, err := p.store.Search(ctx, attribute, value)
			if err != nil {
				return nil, err
			}
			return map[string]interface{}{"matches": facts}, nil
		},
	}

	p.tools["semantic_delete"] = &genericTool{
		def: deleteDef,
		execute: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			entity, _ := args["entity"].(string)
			attribute, _ := args["attribute"].(string)

			if entity == "" || attribute == "" {
				return nil, fmt.Errorf("missing required arguments: entity or attribute")
			}

			err := p.store.Delete(ctx, entity, attribute)
			if err != nil {
				return nil, err
			}
			return map[string]string{"result": "success", "message": "Fact removed from semantic memory."}, nil
		},
	}

	return p
}

// Inject implements llm.Injector
func (p *Provider) Inject(ctx context.Context, query llm.InjectQuery) ([]llm.Message, error) {
	return nil, nil // we do not inject passive messages
}

// Namespace implements llm.ToolProvider
func (p *Provider) Namespace() string {
	return "semantic"
}

// GetTool implements llm.ToolProvider
func (p *Provider) GetTool(name string) (llm.Tool, bool) {
	t, ok := p.tools[name]
	return t, ok
}

type genericTool struct {
	def     llm.ToolDefinitionPart
	execute func(context.Context, map[string]interface{}) (interface{}, error)
}

func (g *genericTool) Name() string {
	return g.def.Name
}

func (g *genericTool) Definition() llm.ToolDefinitionPart {
	return g.def
}

func (g *genericTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return g.execute(ctx, args)
}

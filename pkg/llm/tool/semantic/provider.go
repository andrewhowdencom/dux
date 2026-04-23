package semantic

import (
	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/memory/semantic"
)

type Provider struct {
	service *semantic.Service
	tools   map[string]llm.Tool
}

func NewProvider(service *semantic.Service) *Provider {
	p := &Provider{
		service: service,
		tools:   make(map[string]llm.Tool),
	}

	p.tools["semantic_write_triple"] = NewWriteTripleTool(service)
	p.tools["semantic_write_statement"] = NewWriteStatementTool(service)
	p.tools["semantic_read"] = NewReadTool(service)
	p.tools["semantic_search"] = NewSearchTool(service)
	p.tools["semantic_delete"] = NewDeleteTool(service)
	p.tools["semantic_validate"] = NewValidateTool(service)
	p.tools["semantic_create_relationship"] = NewCreateRelationshipTool(service)
	p.tools["semantic_traverse_graph"] = NewTraverseGraphTool(service)

	return p
}

func (p *Provider) Namespace() string {
	return "semantic"
}

func (p *Provider) Tools() []llm.Tool {
	var tools []llm.Tool
	for _, t := range p.tools {
		tools = append(tools, t)
	}
	return tools
}

func (p *Provider) GetTool(name string) (llm.Tool, bool) {
	t, ok := p.tools[name]
	return t, ok
}

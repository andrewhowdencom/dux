package semantic

import (
	"context"

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

	return p
}

func (p *Provider) Inject(ctx context.Context, query llm.InjectQuery) ([]llm.Message, error) {
	return nil, nil
}

func (p *Provider) Namespace() string {
	return "semantic"
}

func (p *Provider) GetTool(name string) (llm.Tool, bool) {
	t, ok := p.tools[name]
	return t, ok
}

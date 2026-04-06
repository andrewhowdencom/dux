package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/mark3labs/mcp-go/client"
	mcp "github.com/mark3labs/mcp-go/mcp"
)

// MCPResolver adapts an MCP client to the llm.ToolResolver interface.
type MCPResolver struct {
	namespace string
	client    client.MCPClient
	mu        sync.RWMutex
	tools     []llm.Tool
}

// NewMCPResolver creates a new MCPResolver bound to the provided MCP client.
func NewMCPResolver(ctx context.Context, namespace string, c client.MCPClient) (*MCPResolver, error) {
	r := &MCPResolver{
		namespace: namespace,
		client:    c,
	}

	if err := r.refreshCache(ctx); err != nil {
		return nil, fmt.Errorf("failed to fetch initial tools: %w", err)
	}

	c.OnNotification(func(notification mcp.JSONRPCNotification) {
		if notification.Method == "notifications/tools/list_changed" {
			slog.Info("received notifications/tools/list_changed from MCP server")
			go func() {
				if err := r.refreshCache(context.Background()); err != nil {
					slog.Error("failed to refresh MCP tool cache", "error", err)
				}
			}()
		}
	})

	return r, nil
}

func (r *MCPResolver) refreshCache(ctx context.Context) error {
	req := mcp.ListToolsRequest{}
	res, err := r.client.ListTools(ctx, req)
	if err != nil {
		return err
	}

	var newTools []llm.Tool
	for _, t := range res.Tools {
		var params json.RawMessage
		if len(t.RawInputSchema) > 0 {
			params = t.RawInputSchema
		} else {
			b, err := json.Marshal(t.InputSchema)
			if err != nil {
				slog.Warn("failed to marshal MCP tool input schema", "tool", t.Name, "error", err)
				continue
			}
			params = b
		}

		newTools = append(newTools, &mcpTool{
			name:   t.Name,
			client: r.client,
			def: llm.ToolDefinitionPart{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		})
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools = newTools
	return nil
}

// Inject returns the currently cached definitions wrapped in an llm.Message.
func (r *MCPResolver) Inject(ctx context.Context, q llm.InjectQuery) ([]llm.Message, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var parts []llm.Part
	for _, t := range r.tools {
		parts = append(parts, t.Definition())
	}

	if len(parts) == 0 {
		return nil, nil
	}

	return []llm.Message{{
		Identity:   llm.Identity{Role: "system"},
		Parts:      parts,
		Volatility: llm.VolatilityHigh,
	}}, nil
}

func (r *MCPResolver) Namespace() string {
	return r.namespace
}

func (r *MCPResolver) GetTool(name string) (llm.Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, t := range r.tools {
		if t.Name() == name {
			return t, true
		}
	}
	return nil, false
}

type mcpTool struct {
	name   string
	client client.MCPClient
	def    llm.ToolDefinitionPart
}

func (t *mcpTool) Name() string {
	return t.name
}

func (t *mcpTool) Definition() llm.ToolDefinitionPart {
	return t.def
}

func (t *mcpTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	req := mcp.CallToolRequest{}
	req.Params.Name = t.name
	req.Params.Arguments = args

	res, err := t.client.CallTool(ctx, req)
	if err != nil {
		return nil, err
	}

	if res.IsError {
		slog.Error("MCP tool execution returned an error state", "tool", t.name)
	}

	var finalOutput string
	for _, content := range res.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			finalOutput += textContent.Text + "\n"
		} else {
			b, _ := json.Marshal(content)
			finalOutput += string(b) + "\n"
		}
	}

	return finalOutput, nil
}

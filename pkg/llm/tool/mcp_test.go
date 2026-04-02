package tool_test

import (
	"context"
	"strings"
	"testing"

	"github.com/andrewhowdencom/dux/pkg/llm/tool"
	"github.com/mark3labs/mcp-go/client"
	mcp "github.com/mark3labs/mcp-go/mcp"
)

type mockMCPClient struct {
	client.MCPClient 
	listToolsFunc       func(ctx context.Context, request mcp.ListToolsRequest) (*mcp.ListToolsResult, error)
	callToolFunc        func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)
	notificationHandler func(notification mcp.JSONRPCNotification)
}

func (m *mockMCPClient) ListTools(ctx context.Context, request mcp.ListToolsRequest) (*mcp.ListToolsResult, error) {
	if m.listToolsFunc != nil {
		return m.listToolsFunc(ctx, request)
	}
	return &mcp.ListToolsResult{}, nil
}

func (m *mockMCPClient) CallTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if m.callToolFunc != nil {
		return m.callToolFunc(ctx, request)
	}
	return &mcp.CallToolResult{}, nil
}

func (m *mockMCPClient) OnNotification(handler func(notification mcp.JSONRPCNotification)) {
	m.notificationHandler = handler
}

func TestMCPResolver_Initialization(t *testing.T) {
	mockClient := &mockMCPClient{
		listToolsFunc: func(ctx context.Context, request mcp.ListToolsRequest) (*mcp.ListToolsResult, error) {
			return &mcp.ListToolsResult{
				Tools: []mcp.Tool{
					{
						Name:        "calculator",
						Description: "Adds numbers",
					},
				},
			}, nil
		},
	}

	reg, err := tool.NewMCPResolver(context.Background(), mockClient)
	if err != nil {
		t.Fatalf("failed to create MCP resolver: %v", err)
	}

	tools, err := reg.Resolve(context.Background())
	if err != nil {
		t.Fatalf("failed to resolve tools: %v", err)
	}
	
	if len(tools) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(tools))
	}

	defPart := tools[0].Definition()
	if defPart.Name != "calculator" {
		t.Errorf("expected tool name 'calculator', got %q", defPart.Name)
	}
}

func TestMCPResolver_Execute(t *testing.T) {
	mockClient := &mockMCPClient{
		listToolsFunc: func(ctx context.Context, request mcp.ListToolsRequest) (*mcp.ListToolsResult, error) {
			return &mcp.ListToolsResult{
				Tools: []mcp.Tool{
					{Name: "echo"},
				},
			}, nil
		},
		callToolFunc: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if request.Params.Name != "echo" {
				t.Fatalf("expected tool name echo, got %s", request.Params.Name)
			}
			return mcp.NewToolResultText("hello world"), nil
		},
	}

	reg, err := tool.NewMCPResolver(context.Background(), mockClient)
	if err != nil {
		t.Fatalf("failed to create MCP resolver: %v", err)
	}

	tools, err := reg.Resolve(context.Background())
	if err != nil {
		t.Fatalf("failed to resolve tools: %v", err)
	}
	
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	
	res, err := tools[0].Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error on execute: %v", err)
	}

	resStr, ok := res.(string)
	if !ok {
		t.Fatalf("expected string result")
	}

	if !strings.Contains(resStr, "hello world") {
		t.Fatalf("expected 'hello world' in result, got: %s", resStr)
	}
}

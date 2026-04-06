package ui

import (
	"context"
	"fmt"

	"github.com/andrewhowdencom/dux/internal/config"
	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/llm/adapter"
	"github.com/andrewhowdencom/dux/pkg/llm/history"
	"github.com/andrewhowdencom/dux/pkg/llm/tool"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"time"
)

// NewEngine creates an adapter.Engine using the given parameters and global configuration configurations.
func NewEngine(
	ctx context.Context,
	agentName string,
	providerID string,
	agentsFilePath string,
	hitl llm.HITLHandler,
	unsafeAllTools bool,
) (*adapter.Engine, *config.InstanceConfig, func(), error) {

	var finalProvider = providerID
	var sysPrompt string
	var enrichers []llm.Injector
	var resolvers []llm.ToolProvider

	globalTools := config.LoadGlobalTools()

	toolMap := make(map[string]config.ToolConfig)
	requiresSupervision := make(map[string]bool)

	for _, t := range globalTools {
		toolMap[t.Name] = t
	}

	if agentName != "" {
		agents, err := config.LoadAgents(agentsFilePath)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to load agents file: %w", err)
		}
		agt, err := config.GetAgent(agents, agentName)
		if err != nil {
			return nil, nil, nil, err
		}
		finalProvider = agt.Provider
		if agt.Context != nil {
			sysPrompt = agt.Context.System
			en, err := NewEnrichersFromConfig(agt.Context.Enrichers)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed to initialize enrichers for agent %q: %w", agentName, err)
			}
			enrichers = en

			for _, t := range agt.Context.Tools {
				toolMap[t.Name] = t
			}
		}
	}

	var nativeToolNames []string
	var mcpConfigs []config.ToolConfig

	timeouts := make(map[string]time.Duration)

	for name, t := range toolMap {
		if !t.Enabled {
			continue
		}

		if t.TimeoutSeconds != nil {
			timeouts[name] = time.Duration(*t.TimeoutSeconds) * time.Second
		} else if t.MCP != nil {
			timeouts[name] = 300 * time.Second
		} else {
			timeouts[name] = 5 * time.Second
		}

		if t.Requirements.Supervision != nil {
			requiresSupervision[name] = *t.Requirements.Supervision
		} else {
			requiresSupervision[name] = true
		}

		if t.MCP != nil {
			mcpConfigs = append(mcpConfigs, t)
		} else {
			nativeToolNames = append(nativeToolNames, name)
		}
	}

	res, err := NewResolversFromConfig(nativeToolNames)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to initialize native tools: %w", err)
	}
	resolvers = res

	selectedCfg, err := config.LoadLLMProvider(finalProvider)
	if err != nil {
		return nil, nil, nil, err
	}

	prv, err := NewProviderFromConfig(selectedCfg)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to initialize provider %q: %w", selectedCfg.ID, err)
	}

	// Initialize MCP Tool Resolvers
	var mcpClients []*client.Client
	cleanup := func() {
		for _, c := range mcpClients {
			_ = c.Close()
		}
	}

	for _, t := range mcpConfigs {
		var mcpClient *client.Client
		var clientErr error
		s := t.MCP

		transportType := s.Transport
		if transportType == "" {
			if s.Command != "" {
				transportType = "stdio"
			} else if s.URL != "" {
				transportType = "streamable_http"
			}
		}

		switch transportType {
		case "stdio":
			env := make([]string, 0, len(s.Env))
			for k, v := range s.Env {
				env = append(env, fmt.Sprintf("%s=%s", k, v))
			}
			mcpClient, clientErr = client.NewStdioMCPClient(s.Command, env, s.Args...)
		case "sse":
			var opts []transport.ClientOption
			if s.Headers != nil {
				opts = append(opts, transport.WithHeaders(s.Headers))
			}
			mcpClient, clientErr = client.NewSSEMCPClient(s.URL, opts...)
			if clientErr == nil {
				clientErr = mcpClient.Start(ctx)
			}
		case "streamable_http":
			var opts []transport.StreamableHTTPCOption
			if s.Headers != nil {
				opts = append(opts, transport.WithHTTPHeaders(s.Headers))
			}
			var tport *transport.StreamableHTTP
			tport, clientErr = transport.NewStreamableHTTP(s.URL, opts...)
			if clientErr == nil {
				mcpClient = client.NewClient(tport)
				clientErr = mcpClient.Start(ctx)
			}
		default:
			clientErr = fmt.Errorf("unsupported or missing MCP transport type: %q", transportType)
		}

		if clientErr != nil {
			cleanup()
			return nil, nil, nil, fmt.Errorf("failed to create MCP client for %q: %w", t.Name, clientErr)
		}

		// Initialize
		initReq := mcp.InitializeRequest{}
		initReq.Params.ProtocolVersion = "2024-11-05"

		mcpName := agentName
		if mcpName == "" {
			mcpName = "dux"
		}
		initReq.Params.ClientInfo = mcp.Implementation{Name: mcpName, Version: "1.0.0"}
		if _, err := mcpClient.Initialize(ctx, initReq); err != nil {
			cleanup()
			return nil, nil, nil, fmt.Errorf("failed to initialize MCP client %q: %w", t.Name, err)
		}
		mcpClients = append(mcpClients, mcpClient)

		r, err := tool.NewMCPResolver(ctx, mcpClient)
		if err != nil {
			cleanup()
			return nil, nil, nil, fmt.Errorf("failed to bind MCP resolver %q: %w", t.Name, err)
		}
		resolvers = append(resolvers, r)
	}

	mem := history.NewInMemory()

	// Ensure hitl is provided; we can still wrap with unsafeAllTools flag
	hitlMiddleware := llm.NewHITLMiddleware(hitl, requiresSupervision, unsafeAllTools)
	timeoutMiddleware := llm.NewTimeoutMiddleware(timeouts, 5*time.Second)

	opts := []adapter.Option{
		adapter.WithProvider(prv),
		adapter.WithHistory(mem),
		adapter.WithSystemPrompt(sysPrompt),
		adapter.WithEnrichers(enrichers),
		adapter.WithToolMiddleware(hitlMiddleware),
		adapter.WithToolMiddleware(timeoutMiddleware),
	}
	for _, r := range resolvers {
		opts = append(opts, adapter.WithResolver(r))
	}

	engine := adapter.New(opts...)
	return engine, &selectedCfg, cleanup, nil
}

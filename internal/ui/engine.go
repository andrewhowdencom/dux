package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/andrewhowdencom/dux/internal/config"
	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/llm/adapter"
	"github.com/andrewhowdencom/dux/pkg/llm/tool/semantic"
	"github.com/andrewhowdencom/dux/pkg/memory/semantic/sqlite"
	"github.com/andrewhowdencom/dux/pkg/memory/working"
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
	requiresSupervision := make(map[string]interface{})

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

	var semanticToolNames []string

	for name, t := range toolMap {
		if !t.Enabled {
			continue
		}

		if t.Requirements.Supervision != nil {
			requiresSupervision[name] = t.Requirements.Supervision
		} else {
			if len(name) >= 9 && name[:9] == "semantic_" {
				requiresSupervision[name] = false
			} else {
				requiresSupervision[name] = true
			}
		}

		if t.TimeoutSeconds != nil {
			timeouts[name] = time.Duration(*t.TimeoutSeconds) * time.Second
		} else if t.MCP != nil {
			timeouts[name] = 300 * time.Second
		} else {
			timeouts[name] = 5 * time.Second
		}

		if t.MCP != nil {
			mcpConfigs = append(mcpConfigs, t)
		} else if name == "semantic" || (len(name) >= 9 && name[:9] == "semantic_") {
			semanticToolNames = append(semanticToolNames, name)
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
	mcpResolvers, cleanup, err := NewMCPResolversFromConfig(ctx, agentName, mcpConfigs)
	if err != nil {
		return nil, nil, nil, err
	}
	resolvers = append(resolvers, mcpResolvers...)

	if len(semanticToolNames) > 0 {
		var dbPath = ":memory:" // Default to in-memory for now
		store, err := sqlite.NewStore(dbPath)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to initialize semantic memory store: %w", err)
		}
		semProvider := semantic.NewProvider(store)
		resolvers = append(resolvers, semProvider)
	}

	mem := working.NewInMemory()

	// Ensure hitl is provided; we can still wrap with unsafeAllTools flag
	hitlMiddleware := llm.NewHITLMiddleware(hitl, requiresSupervision, unsafeAllTools)
	timeoutMiddleware := llm.NewTimeoutMiddleware(timeouts, 5*time.Second)

	opts := []adapter.Option{
		adapter.WithProvider(prv),
		adapter.WithWorkingMemory(mem),
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

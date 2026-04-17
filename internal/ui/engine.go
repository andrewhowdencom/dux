package ui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/spf13/viper"

	"github.com/andrewhowdencom/dux/internal/config"
	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/llm/adapter"
	"github.com/andrewhowdencom/dux/pkg/llm/tool/binary"
	"github.com/andrewhowdencom/dux/pkg/llm/tool/static"
	"github.com/andrewhowdencom/dux/pkg/llm/tool/transition"
	"github.com/andrewhowdencom/dux/pkg/memory/working"
)

// NewEngine creates an llm.Engine using the given parameters and configurations.
// If the agent specifies a Workflow, it natively wraps execution in a WorkflowEngine.
func NewEngine(
	ctx context.Context,
	agentName string,
	providerID string,
	agentsFilePath string,
	hitl llm.HITLHandler,
	unsafeAllTools bool,
) (llm.Engine, *config.InstanceConfig, working.WorkingMemory, func(), error) {
	var allCleanups []func()
	globalCleanup := func() {
		for _, c := range allCleanups {
			c()
		}
	}

	var agt *config.Agent
	if agentName != "" {
		agents, err := config.LoadAgents(agentsFilePath)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("failed to load agents file: %w", err)
		}
		a, err := config.GetAgent(agents, agentName)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		agt = &a
	}

	if agt != nil && agt.Workflow != nil {
		var selectedCfg *config.InstanceConfig
		memories := make(map[string]llm.Injector)

		var effectiveHistoryPath string
		if viper.GetString("memory.type") == "disk" {
			effectiveHistoryPath = filepath.Join(xdg.DataHome, "dux", "sessions")
		}

		var primaryMem working.WorkingMemory
		if effectiveHistoryPath != "" {
			dbMem, err := working.NewDiskBacked(effectiveHistoryPath)
			if err != nil {
				return nil, nil, nil, nil, fmt.Errorf("failed to load disk-backed memory: %w", err)
			}
			memories["orchestrator"] = dbMem
			primaryMem = dbMem
		} else {
			im := working.NewInMemory()
			memories["orchestrator"] = im
			primaryMem = im
		}

		factory := func(modeName string) ([]adapter.Option, error) {
			var targetMode *config.Mode
			for _, m := range agt.Workflow.Modes {
				if m.Name == modeName {
					targetMode = &m
					break
				}
			}
			if targetMode == nil {
				return nil, fmt.Errorf("workflow mode not found: %s", modeName)
			}

			var transitionTools []llm.Tool
			for _, t := range targetMode.Transitions {
				transitionTools = append(transitionTools, transition.New(t.To, t.Description))
			}

			modeMem, ok := memories[modeName]
			if !ok {
				modeMem = working.NewInMemory()
				memories[modeName] = modeMem
			}

			globalMem := memories["orchestrator"]

			effectiveGlobalProvider := providerID
			if effectiveGlobalProvider == "" && agt != nil {
				effectiveGlobalProvider = agt.Provider
			}

			opts, cfg, cleanup, err := compileOptions(ctx, agentName, effectiveGlobalProvider, targetMode.Provider, targetMode.Context, hitl, unsafeAllTools, modeMem, globalMem, transitionTools)
			if err != nil {
				return nil, err
			}
			if cleanup != nil {
				allCleanups = append(allCleanups, cleanup)
			}

			// We always return the configuration of the latest mode evaluated requested by the caller
			selectedCfg = cfg

			return opts, nil
		}

		engine, err := adapter.NewWorkflowEngine(agt.Workflow.DefaultMode, factory)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		return engine, selectedCfg, primaryMem, globalCleanup, nil
	}

	// Fallback to single-context core engine
	var effectiveHistoryPath string
	if viper.GetString("memory.type") == "disk" {
		effectiveHistoryPath = filepath.Join(xdg.DataHome, "dux", "sessions")
	}

	var mem working.WorkingMemory
	if effectiveHistoryPath != "" {
		dbMem, err := working.NewDiskBacked(effectiveHistoryPath)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("failed to load disk-backed memory: %w", err)
		}
		mem = dbMem
	} else {
		mem = working.NewInMemory()
	}

	var contextCfg *config.AgentContext
	var fallbackProvider string
	if agt != nil {
		contextCfg = agt.Context
		fallbackProvider = agt.Provider
	}

	opts, cfg, cleanup, err := compileOptions(ctx, agentName, providerID, fallbackProvider, contextCfg, hitl, unsafeAllTools, mem, mem, nil)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if cleanup != nil {
		allCleanups = append(allCleanups, cleanup)
	}

	return adapter.New(opts...), cfg, mem, globalCleanup, nil
}

func compileOptions(
	ctx context.Context,
	agentName string,
	globalProviderID string,
	localProviderID string,
	contextCfg *config.AgentContext,
	hitl llm.HITLHandler,
	unsafeAllTools bool,
	mem llm.Injector,
	globalMem llm.Injector,
	transitionTools []llm.Tool,
) ([]adapter.Option, *config.InstanceConfig, func(), error) {
	var finalProvider = localProviderID
	if finalProvider == "" {
		finalProvider = globalProviderID
	}

	var sysPrompt string
	var enrichers []llm.Injector
	var resolvers []llm.ToolProvider

	globalTools := config.LoadGlobalTools()
	toolMap := make(map[string]config.ToolConfig)
	requiresSupervision := make(map[string]interface{})

	var flattenTools func(tools []config.ToolConfig)
	flattenTools = func(tools []config.ToolConfig) {
		for _, t := range tools {
			toolMap[t.Name] = t
			if len(t.Tools) > 0 {
				flattenTools(t.Tools)
			}
		}
	}

	flattenTools(globalTools)

	if contextCfg != nil {
		sysPrompt = contextCfg.System
		en, err := NewEnrichersFromConfig(contextCfg.Enrichers)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to initialize enrichers: %w", err)
		}
		enrichers = en
		flattenTools(contextCfg.Tools)
	}

	var nativeToolNames []string
	var mcpConfigs []config.ToolConfig
	var binaryConfigs []config.ToolConfig
	timeouts := make(map[string]time.Duration)

	for name, t := range toolMap {
		if !t.Enabled {
			continue
		}

		if t.Requirements.Supervision != nil {
			requiresSupervision[name] = t.Requirements.Supervision
		} else {
			if strings.HasPrefix(name, "semantic_") || strings.HasPrefix(name, "switch_to_") || strings.HasPrefix(name, "delegate_") || strings.HasPrefix(name, "handover_") || strings.HasPrefix(name, "transition_") || strings.HasPrefix(name, "plan_") || name == "read_working_memory" || name == "librarian" {
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
		} else if t.Binary != nil {
			binaryConfigs = append(binaryConfigs, t)
		} else {
			if len(t.Tools) == 0 {
				nativeToolNames = append(nativeToolNames, name)
			}
		}
	}

	res, err := NewResolversFromConfig(nativeToolNames, ResolverDependencies{GlobalMemory: globalMem})
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

	mcpResolvers, cleanup, err := NewMCPResolversFromConfig(ctx, agentName, mcpConfigs)
	if err != nil {
		return nil, nil, nil, err
	}
	resolvers = append(resolvers, mcpResolvers...)

	for _, bCfg := range binaryConfigs {
		resolvers = append(resolvers, binary.NewProvider(bCfg.Name, bCfg.Binary))
	}

	// Inject standard transition tools statically
	if len(transitionTools) > 0 {
		resolvers = append(resolvers, static.New("transitions", transitionTools...))
		// Transitions bypass HITL by default because:
		// 1. Transition tools only emit signals (no side effects)
		// 2. Modes are pre-configured in agent.yaml (no arbitrary mode switching)
		// 3. Mode configurations define their own tool supervision rules
		// 4. Users can override by setting supervision: true for "transitions" namespace
		//
		// Example override in agent.yaml:
		//   context:
		//     tools:
		//       - name: "transitions"
		//         enabled: true
		//         requirements:
		//           supervision: true
		if _, exists := requiresSupervision["transitions"]; !exists {
			requiresSupervision["transitions"] = false
		}
	}

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

	return opts, &selectedCfg, cleanup, nil
}

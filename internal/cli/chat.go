package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/adrg/xdg"
	"github.com/andrewhowdencom/dux/internal/config"
	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/llm/adapter"
	"github.com/andrewhowdencom/dux/pkg/llm/enrich"
	"github.com/andrewhowdencom/dux/pkg/llm/history"
	"github.com/andrewhowdencom/dux/pkg/llm/tool"
	"github.com/andrewhowdencom/dux/pkg/terminal"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var providerID string
var chatTheme string
var agentName string
var unsafeAllTools bool

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start an interactive chat session with Dux",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigCh
			fmt.Println("\nExiting chat...")
			cancel()
			os.Exit(0)
		}()

		var finalProvider = providerID
		var sysPrompt string
		var enrichers []enrich.Enricher
		var resolvers []llm.ToolResolver

		// Resolve agents file default if not specified explicitly
		var agentsFilePath = agentsFile
		if agentsFilePath == "" {
			path, err := xdg.ConfigFile("dux/agents.yaml")
			if err == nil {
				agentsFilePath = path
			}
		}

		globalTools := config.LoadGlobalTools()

		toolMap := make(map[string]config.ToolConfig)
		requiresSupervision := make(map[string]bool)

		for _, t := range globalTools {
			toolMap[t.Name] = t
		}

		if agentName != "" {
			agents, err := config.LoadAgents(agentsFilePath)
			if err != nil {
				return fmt.Errorf("failed to load agents file: %w", err)
			}
			agt, err := config.GetAgent(agents, agentName)
			if err != nil {
				return err
			}
			finalProvider = agt.Provider
			if agt.Context != nil {
				sysPrompt = agt.Context.System
				en, err := newEnrichersFromConfig(agt.Context.Enrichers)
				if err != nil {
					return fmt.Errorf("failed to initialize enrichers for agent %q: %w", agentName, err)
				}
				enrichers = en

				for _, t := range agt.Context.Tools {
					toolMap[t.Name] = t
				}
			}
		}

		var nativeToolNames []string
		var mcpConfigs []config.ToolConfig

		for name, t := range toolMap {
			if !t.Enabled {
				continue
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

		res, err := newResolversFromConfig(nativeToolNames)
		if err != nil {
			return fmt.Errorf("failed to initialize native tools: %w", err)
		}
		resolvers = res

		selectedCfg, err := config.LoadLLMProvider(finalProvider)
		if err != nil {
			return err
		}

		prv, err := newProviderFromConfig(selectedCfg)
		if err != nil {
			return fmt.Errorf("failed to initialize provider %q: %w", selectedCfg.ID, err)
		}

		// Initialize MCP Tool Resolvers
		var mcpClients []*client.Client
		defer func() {
			for _, c := range mcpClients {
				_ = c.Close()
			}
		}()

		for _, t := range mcpConfigs {
			var mcpClient *client.Client
			var clientErr error
			s := t.MCP

			if s.Command != "" {
				env := make([]string, 0, len(s.Env))
				for k, v := range s.Env {
					env = append(env, fmt.Sprintf("%s=%s", k, v))
				}
				mcpClient, clientErr = client.NewStdioMCPClient(s.Command, env, s.Args...)
			} else if s.URL != "" {
				var opts []transport.ClientOption
				if s.Headers != nil {
					opts = append(opts, transport.WithHeaders(s.Headers))
				}
				mcpClient, clientErr = client.NewSSEMCPClient(s.URL, opts...)
				if clientErr == nil {
					clientErr = mcpClient.Start(ctx)
				}
			}

			if clientErr != nil {
				return fmt.Errorf("failed to create MCP client for %q: %w", t.Name, clientErr)
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
				return fmt.Errorf("failed to initialize MCP client %q: %w", t.Name, err)
			}
			mcpClients = append(mcpClients, mcpClient)

			r, err := tool.NewMCPResolver(ctx, mcpClient)
			if err != nil {
				return fmt.Errorf("failed to bind MCP resolver %q: %w", t.Name, err)
			}
			resolvers = append(resolvers, r)
		}

		mem := history.NewInMemory()

		hitl := terminal.NewBubbleTeaHITL()
		hitlMiddleware := llm.NewHITLMiddleware(hitl, requiresSupervision, unsafeAllTools)

		opts := []adapter.Option{
			adapter.WithProvider(prv),
			adapter.WithHistory(mem),
			adapter.WithSystemPrompt(sysPrompt),
			adapter.WithEnrichers(enrichers),
			adapter.WithToolMiddleware(hitlMiddleware),
		}
		for _, r := range resolvers {
			opts = append(opts, adapter.WithResolver(r))
		}

		engine := adapter.New(opts...)

		var modelName string
		if m, ok := selectedCfg.Config["model"].(string); ok {
			modelName = m
		} else {
			modelName = selectedCfg.Type
		}

		var theme string
		if t := viper.GetString("chat.theme"); t != "" {
			theme = t
		} else {
			theme = "dark"
		}

		_ = terminal.StartREPL(ctx, engine, modelName, theme, agentName, hitl, os.Stdin, os.Stdout)

		fmt.Println("\nChat session ended.")
		return nil
	},
}

func init() {
	chatCmd.Flags().StringVar(&providerID, "provider", "", "LLM provider ID from config to use (e.g. 'ollama', 'static')")
	chatCmd.Flags().StringVar(&agentName, "agent", "", "Agent spec name to use (mutually exclusive with --provider)")
	chatCmd.MarkFlagsMutuallyExclusive("provider", "agent")
	chatCmd.Flags().StringVar(&chatTheme, "theme", "dark", "Theme for chat rendering. Supported: ascii, dark, dracula, light, notty, pink, tokyo-night, or path/to/style.json")
	_ = viper.BindPFlag("chat.theme", chatCmd.Flags().Lookup("theme"))
	chatCmd.Flags().BoolVar(&unsafeAllTools, "unsafe-all-tools", false, "Disable hitl prompts unconditionally for all tools")
	RootCmd.AddCommand(chatCmd)
}

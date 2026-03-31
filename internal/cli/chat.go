package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/adrg/xdg"
	"github.com/andrewhowdencom/dux/internal/config"
	"github.com/andrewhowdencom/dux/pkg/llm/adapter"
	"github.com/andrewhowdencom/dux/pkg/llm/enrich"
	"github.com/andrewhowdencom/dux/pkg/llm/history"
	"github.com/andrewhowdencom/dux/pkg/llm/provider/factory"
	"github.com/andrewhowdencom/dux/pkg/terminal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var providerID string
var chatTheme string
var agentName string

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

		// Resolve agents file default if not specified explicitly
		var agentsFilePath = agentsFile
		if agentsFilePath == "" {
			path, err := xdg.ConfigFile("dux/agents.yaml")
			if err == nil {
				agentsFilePath = path
			}
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
				en, err := enrich.NewFromConfig(agt.Context.Enrichers)
				if err != nil {
					return fmt.Errorf("failed to initialize enrichers for agent %q: %w", agentName, err)
				}
				enrichers = en
			}
		}

		selectedCfg, err := config.LoadLLMProvider(finalProvider)
		if err != nil {
			return err
		}

		prv, err := factory.New(selectedCfg)
		if err != nil {
			return fmt.Errorf("failed to initialize provider %q: %w", selectedCfg.ID, err)
		}

		mem := history.NewInMemory()

		engine := adapter.New(
			adapter.WithProvider(prv),
			adapter.WithHistory(mem),
			adapter.WithSystemPrompt(sysPrompt),
			adapter.WithEnrichers(enrichers),
		)

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

		_ = terminal.StartREPL(ctx, engine, modelName, theme, os.Stdin, os.Stdout)

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
	RootCmd.AddCommand(chatCmd)
}

package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/andrewhowdencom/dux/internal/ui"
	"github.com/andrewhowdencom/dux/pkg/terminal"
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

		hitl := terminal.NewBubbleTeaHITL()
		engine, selectedCfg, cleanup, err := ui.NewEngine(ctx, agentName, providerID, agentsDir, hitl, unsafeAllTools)
		if err != nil {
			return err
		}
		defer cleanup()

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

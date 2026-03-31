package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/andrewhowdencom/dux/internal/config"
	"github.com/andrewhowdencom/dux/pkg/llm/adapter"
	"github.com/andrewhowdencom/dux/pkg/llm/provider/factory"
	"github.com/andrewhowdencom/dux/pkg/terminal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var providerID string
var chatTheme string

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

		selectedCfg, err := config.LoadLLMProvider(providerID)
		if err != nil {
			return err
		}

		prv, err := factory.New(selectedCfg)
		if err != nil {
			return fmt.Errorf("failed to initialize provider %q: %w", selectedCfg.ID, err)
		}

		engine := adapter.New(adapter.WithProvider(prv))

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
	chatCmd.Flags().StringVar(&chatTheme, "theme", "dark", "Theme for chat rendering. Supported: ascii, dark, dracula, light, notty, pink, tokyo-night, or path/to/style.json")
	viper.BindPFlag("chat.theme", chatCmd.Flags().Lookup("theme"))
	RootCmd.AddCommand(chatCmd)
}

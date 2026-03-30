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
)

var providerID string

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

		fmt.Printf("Starting interactive chat using provider: %s (%s). Press Ctrl+D/Ctrl+C to exit.\n", selectedCfg.ID, selectedCfg.Type)

		prv, err := factory.New(selectedCfg)
		if err != nil {
			return fmt.Errorf("failed to initialize provider %q: %w", selectedCfg.ID, err)
		}

		engine := adapter.New(adapter.WithProvider(prv))

		_ = terminal.StartREPL(ctx, engine, os.Stdin, os.Stdout)

		fmt.Println("\nChat session ended.")
		return nil
	},
}

func init() {
	chatCmd.Flags().StringVar(&providerID, "provider", "", "LLM provider ID from config to use (e.g. 'ollama', 'static')")
	RootCmd.AddCommand(chatCmd)
}

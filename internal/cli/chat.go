package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/terminal"
	"github.com/spf13/cobra"
)

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

		fmt.Println("Starting interactive chat. Press Ctrl+D or Ctrl+C to exit.")

		// Initialize StaticEngine with canned response
		staticEngine := llm.NewStaticEngine(llm.Message{
			SessionID: "cli-session",
			Identity:  llm.Identity{Role: "assistant"},
			Parts:     []llm.Part{llm.TextPart("I am the static agent. I always return this canned response!")},
		})

		_ = terminal.StartREPL(ctx, staticEngine, os.Stdin, os.Stdout)

		fmt.Println("\nChat session ended.")
		return nil
	},
}

func init() {
	RootCmd.AddCommand(chatCmd)
}

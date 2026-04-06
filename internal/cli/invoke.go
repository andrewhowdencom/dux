package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/andrewhowdencom/dux/internal/ui"
	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/terminal"
	"github.com/andrewhowdencom/dux/pkg/trigger"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var invokeCmd = &cobra.Command{
	Use:   "invoke [agent]",
	Short: "Execute a single one-shot background query using stdin",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// Read stdin
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read stdin: %w", err)
		}
		promptLine := string(b)
		if promptLine == "" {
			return fmt.Errorf("stdin is empty")
		}

		agentName := args[0]
		
		hitl := terminal.NewBubbleTeaHITL() // Could be a dummy HITL that always approves or rejects
		engine, _, cleanup, err := ui.NewEngine(ctx, agentName, "", agentsDir, hitl, unsafeAllTools)
		if err != nil {
			return err
		}
		defer cleanup()

		handler := func(ctx context.Context, prompt string) error {
			msg := llm.Message{
				SessionID: uuid.New().String(),
				Identity:  llm.Identity{Role: "user"},
				Parts:     []llm.Part{llm.TextPart(prompt)},
			}
			ctxWithSession := llm.WithSessionID(ctx, msg.SessionID)
			ch, err := engine.Stream(ctxWithSession, msg)
			if err != nil {
				return err
			}
			
			for out := range ch {
				fmt.Print(out.Text())
			}
			fmt.Println()
			return nil
		}

		immediate := trigger.NewImmediate(promptLine, handler)
		return immediate.Start(ctx)
	},
}

func init() {
	invokeCmd.Flags().BoolVar(&unsafeAllTools, "unsafe-all-tools", false, "Disable hitl prompts unconditionally for all tools")
	RootCmd.AddCommand(invokeCmd)
}

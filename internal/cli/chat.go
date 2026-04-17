package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/andrewhowdencom/dux/internal/config"
	internalui "github.com/andrewhowdencom/dux/internal/ui"
	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/memory/working"
	"github.com/andrewhowdencom/dux/pkg/terminal"
	"github.com/andrewhowdencom/dux/pkg/ui"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var providerID string
var chatTheme string
var agentName string
var unsafeAllTools bool
var historyPath string

func makeToolConfigs() (map[string]ui.ToolDisplayConfig, string) {
	toolDisplayCfg := config.LoadToolDisplayConfig()
	toolConfigs := make(map[string]ui.ToolDisplayConfig)
	for name, cfg := range toolDisplayCfg.Tools {
		toolConfigs[name] = ui.ToolDisplayConfig{
			Icon:         cfg.Icon,
			HideArgs:     cfg.HideArgs,
			HideResult:   cfg.HideResult,
			MaxResultLen: cfg.MaxResultLen,
		}
	}
	return toolConfigs, toolDisplayCfg.DefaultIcon
}

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
		engine, selectedCfg, mem, cleanup, err := internalui.NewEngine(ctx, agentName, providerID, agentsDir, hitl, unsafeAllTools)
		if err != nil {
			return err
		}
		defer cleanup()

		initialSessionID := uuid.New().String()
		var initialMessages []llm.Message

		if historyPath != "" {
			data, err := os.ReadFile(historyPath)
			if err == nil {
				// We expect an array of llm.Message if created by DiskBacked Dir store
				var msgs []llm.Message
				if err := json.Unmarshal(data, &msgs); err == nil {
					initialMessages = msgs
				}

				for _, msg := range initialMessages {
					_ = mem.Append(ctx, initialSessionID, msg)
				}
			}
		} else if db, ok := mem.(*working.DiskBacked); ok {
			sessions := db.Sessions()
			for sid, msgs := range sessions {
				initialSessionID = sid
				initialMessages = append(initialMessages, msgs...)
				break // Naively pick the first one since it's a 1:1 file usually
			}
		}

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

		toolConfigs, defaultIcon := makeToolConfigs()

		_ = terminal.StartREPL(
			ctx,
			initialSessionID,
			initialMessages,
			engine,
			modelName,
			theme,
			agentName,
			hitl,
			os.Stdin,
			os.Stdout,
			terminal.WithToolDisplayConfig(toolConfigs, defaultIcon),
		)

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
	chatCmd.Flags().StringVar(&historyPath, "history", "", "Path to a JSON file to load and save session memory")
	RootCmd.AddCommand(chatCmd)
}

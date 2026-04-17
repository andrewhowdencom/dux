package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/andrewhowdencom/dux/internal/config"
	internalui "github.com/andrewhowdencom/dux/internal/ui"
	"github.com/andrewhowdencom/dux/pkg/llm"
	"github.com/andrewhowdencom/dux/pkg/terminal"
	"github.com/andrewhowdencom/dux/pkg/trigger"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

var runCmd = &cobra.Command{
	Use:   "run [agent]",
	Short: "Run an agent and all its associated triggers",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigCh
			fmt.Println("\nExiting run...")
			cancel()
			os.Exit(0)
		}()

		agentName := args[0]
		agents, err := config.LoadAgents(agentsDir)
		if err != nil {
			return err
		}

		agt, err := config.GetAgent(agents, agentName)
		if err != nil {
			return err
		}

		bus := trigger.NewInMemoryEventBus()

		hitl := terminal.NewBubbleTeaHITL()
		engine, selectedCfg, _, cleanup, err := internalui.NewEngine(ctx, agentName, agt.Provider, agentsDir, hitl, unsafeAllTools)
		if err != nil {
			return err
		}
		defer cleanup()

		// Construct the handler for standard event / schedule consumption
		handler := func(hCtx context.Context, prompt string) error {
			msg := llm.Message{
				SessionID: uuid.New().String(),
				Identity:  llm.Identity{Role: "user"},
				Parts:     []llm.Part{llm.TextPart(prompt)},
			}
			ctxWithSession := llm.WithSessionID(hCtx, msg.SessionID)
			ch, streamErr := engine.Stream(ctxWithSession, msg)
			if streamErr != nil {
				return streamErr
			}
			for out := range ch {
				// Easiest headless printing
				// For real use, maybe pipe logger / stderr?
				fmt.Print(out.Text())
			}
			fmt.Println()
			return nil
		}

		var triggers []trigger.Trigger

		for _, tConfig := range agt.Triggers {
			switch tConfig.Type {
			case "schedule":
				expr := tConfig.Config["cron"]
				if expr == "" {
					return fmt.Errorf("cron expression missing for schedule trigger")
				}
				eventType := tConfig.Config["event_type"]
				topic := tConfig.Config["topic"]
				prompt := tConfig.Config["prompt"]
				triggers = append(triggers, trigger.NewSchedule(expr, eventType, topic, prompt, bus, nil))
			case "event":
				topic := tConfig.Config["topic"]
				if topic == "" {
					return fmt.Errorf("topic missing for event trigger")
				}
				triggers = append(triggers, trigger.NewEvent(topic, bus, handler))
			case "chat":
				var modelName string
				if m, ok := selectedCfg.Config["model"].(string); ok {
					modelName = m
				} else {
					modelName = selectedCfg.Type
				}
				theme := viper.GetString("chat.theme")
				if theme == "" {
					theme = "dark"
				}

				startFn := func(iCtx context.Context) error {
					toolConfigs, defaultIcon := makeToolConfigs()

					return terminal.StartREPL(
						iCtx,
						"",
						nil,
						engine,
						modelName,
						theme,
						agentName,
						hitl,
						os.Stdin,
						os.Stdout,
						terminal.WithToolDisplayConfig(toolConfigs, defaultIcon),
					)
				}
				triggers = append(triggers, trigger.NewInteractive(startFn))
			default:
				return fmt.Errorf("unknown trigger type: %q", tConfig.Type)
			}
		}

		if len(triggers) == 0 {
			return fmt.Errorf("no triggers configured for agent %q", agentName)
		}

		g, gCtx := errgroup.WithContext(ctx)
		for _, tr := range triggers {
			tr := tr
			g.Go(func() error {
				return tr.Start(gCtx)
			})
		}

		return g.Wait()
	},
}

func init() {
	runCmd.Flags().BoolVar(&unsafeAllTools, "unsafe-all-tools", false, "Disable hitl prompts unconditionally for all tools")
	RootCmd.AddCommand(runCmd)
}

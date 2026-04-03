package cli

import (
	"context"
	"fmt"
	"sort"

	"github.com/andrewhowdencom/dux/internal/config"
	"github.com/andrewhowdencom/dux/internal/ui"
	"github.com/spf13/cobra"
)

var llmProviderID string

var llmCmd = &cobra.Command{
	Use:   "llm",
	Short: "Manage LLM configurations and models",
}

var llmModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Interact with LLM models",
}

var llmModelsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available models for the configured or specified provider",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		selectedCfg, err := config.LoadLLMProvider(llmProviderID)
		if err != nil {
			return err
		}

		prv, err := ui.NewProviderFromConfig(selectedCfg)
		if err != nil {
			return fmt.Errorf("failed to initialize provider %q: %w", selectedCfg.ID, err)
		}

		models, err := prv.ListModels(ctx)
		if err != nil {
			return fmt.Errorf("failed to list models for provider %q: %w", selectedCfg.ID, err)
		}

		sort.Strings(models)

		for _, m := range models {
			fmt.Println(m)
		}

		return nil
	},
}

func init() {
	llmModelsListCmd.Flags().StringVar(&llmProviderID, "provider", "", "LLM provider ID from config to use (e.g. 'ollama', 'litellm')")

	llmModelsCmd.AddCommand(llmModelsListCmd)
	llmCmd.AddCommand(llmModelsCmd)

	RootCmd.AddCommand(llmCmd)
}

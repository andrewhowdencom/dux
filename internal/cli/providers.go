package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/andrewhowdencom/dux/internal/config"
	"github.com/spf13/cobra"
)

var providersCmd = &cobra.Command{
	Use:   "providers",
	Short: "Manage LLM provider configurations",
	Long:  "Commands for managing and viewing LLM provider configurations.",
}

var providersListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List configured LLM providers",
	Long:    "Display a table of all configured LLM providers with their type, connection details, and default status.",
	Example: "  dux providers list",
	RunE: func(cmd *cobra.Command, args []string) error {
		infos, err := config.ListProviders()
		if err != nil {
			return err
		}

		if len(infos) == 0 {
			fmt.Println("No LLM providers configured.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tTYPE\tDETAILS\tDEFAULT")
		for _, info := range infos {
			defaultMark := ""
			if info.IsDefault {
				defaultMark = "yes"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", info.ID, info.Type, info.Details, defaultMark)
		}
		return w.Flush()
	},
}

func init() {
	providersCmd.AddCommand(providersListCmd)
	RootCmd.AddCommand(providersCmd)
}

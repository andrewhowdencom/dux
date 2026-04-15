package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/andrewhowdencom/dux/internal/config"
	"github.com/spf13/cobra"
)

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "Manage agent configurations",
	Long:  "Commands for managing and viewing agent configurations.",
}

var agentsListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List configured agents",
	Long:    "Display a table of all configured agents with their provider, workflow modes, and triggers.",
	Example: "  dux agents list",
	RunE: func(cmd *cobra.Command, args []string) error {
		infos, err := config.ListAgents(agentsDir)
		if err != nil {
			return err
		}

		if len(infos) == 0 {
			fmt.Println("No agents configured. Add agents to:", config.ResolveAgentsDir(agentsDir))
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tPROVIDER\tMODES\tTRIGGERS")
		for _, info := range infos {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", info.Name, info.Provider, info.Modes, info.Triggers)
		}
		return w.Flush()
	},
}

var agentsModesCmd = &cobra.Command{
	Use:   "modes",
	Short: "Manage agent workflow modes",
	Long:  "Commands for managing and viewing agent workflow modes.",
}

var agentsModesListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List workflow modes for all agents",
	Long:    "Display a table of all workflow modes across all agents, including the agent name, mode name, provider, and transitions.",
	Example: "  dux agents modes list",
	RunE: func(cmd *cobra.Command, args []string) error {
		infos, err := config.ListModes(agentsDir)
		if err != nil {
			return err
		}

		if len(infos) == 0 {
			fmt.Println("No workflow modes configured.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "AGENT\tMODE\tPROVIDER\tTRANSITIONS")
		for _, info := range infos {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", info.AgentName, info.Name, info.Provider, info.Transitions)
		}
		return w.Flush()
	},
}

func init() {
	agentsModesCmd.AddCommand(agentsModesListCmd)
	agentsCmd.AddCommand(agentsListCmd)
	agentsCmd.AddCommand(agentsModesCmd)
	RootCmd.AddCommand(agentsCmd)
}

package cli

import (
	"github.com/spf13/cobra"
)

var httpCmd = &cobra.Command{
	Use:   "http",
	Short: "HTTP operations",
}

func init() {
	RootCmd.AddCommand(httpCmd)
}

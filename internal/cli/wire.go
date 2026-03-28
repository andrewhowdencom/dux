//go:build wireinject
// +build wireinject

package cli

import (
	"github.com/google/wire"
	"github.com/spf13/cobra"
)

// InitializeRootCmd uses Wire to inject dependencies and construct the root command.
func InitializeRootCmd() *cobra.Command {
	wire.Build(
		NewRootCommand,
	)
	return nil
}

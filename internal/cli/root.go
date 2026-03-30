package cli

import (
	"context"
	"log/slog"
	"os"

	"github.com/adrg/xdg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	logLevel string
	cfgFile  string
)

// RootCmd is the root command for the CLI application.
var RootCmd = InitializeRootCmd()

func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "dux",
		Short: "Fast harness for agents",
		Long:  `Fast harness for agents. Latin for "guide". Short enough for the CLI!`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := setupLogger(logLevel); err != nil {
				return err
			}
			return setupConfig(cfgFile)
		},
	}

	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Modifies the log/slog behavior for the default logger")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $XDG_CONFIG_HOME/dux/config.yaml)")

	return rootCmd
}

func setupLogger(levelStr string) error {
	var level slog.Level
	if err := level.UnmarshalText([]byte(levelStr)); err != nil {
		return err
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, opts))
	slog.SetDefault(logger)

	return nil
}

func setupConfig(cfgFile string) error {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		configFilePath, err := xdg.ConfigFile("dux/config.yaml")
		if err == nil {
			viper.SetConfigFile(configFilePath)
		} else {
			// Fallback if XDG lookup fails
			viper.AddConfigPath("$HOME/.config/dux")
			viper.SetConfigName("config")
			viper.SetConfigType("yaml")
		}
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if using default search
			slog.Debug("Config file not found, proceeding with defaults")
		} else if os.IsNotExist(err) {
			slog.Debug("Config file not found, proceeding with defaults")
		} else {
			// Config file was found but another error was produced
			slog.Warn("Failed to read config file", "error", err)
		}
	} else {
		slog.Debug("Using config file", "file", viper.ConfigFileUsed())
	}

	return nil
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(ctx context.Context) {
	if err := RootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

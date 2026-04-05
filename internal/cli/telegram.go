package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/andrewhowdencom/dux/internal/ui/telegram"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var telegramServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts the Telegram Bot Server",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigCh
			slog.Info("Shutting down Telegram Bot...")
			cancel()
			os.Exit(0)
		}()

		token := viper.GetString("telegram.token")
		if token == "" {
			return fmt.Errorf("telegram.token configuration is required. Please set it in your config.yaml")
		}

		cfg := telegram.Config{
			Token:      token,
			WebhookURL: viper.GetString("telegram.webhook_url"),
			AgentsFile: agentsFile,
		}

		for _, v := range viper.GetIntSlice("telegram.allowed_users") {
			cfg.AllowedUsers = append(cfg.AllowedUsers, int64(v))
		}

		srv, err := telegram.NewServer(cfg)
		if err != nil {
			return fmt.Errorf("failed to create telegram server: %w", err)
		}

		if cfg.WebhookURL != "" {
			address := viper.GetString("telegram.webhook_address")
			if address == "" {
				address = ":8443"
			}
			slog.Info("Starting Telegram Bot with Webhooks", "url", cfg.WebhookURL, "address", address)
			return srv.StartWebhook(ctx, cfg.WebhookURL, address)
		}

		// Fallback to polling
		slog.Info("Starting Telegram Bot with Long-Polling ...")
		return srv.Start(ctx)
	},
}

var telegramCmd = &cobra.Command{
	Use:   "telegram",
	Short: "Telegram bot commands",
}

func init() {
	telegramServeCmd.Flags().String("webhook-address", ":8443", "Address to listen on for the Telegram webhook server")
	_ = viper.BindPFlag("telegram.webhook_address", telegramServeCmd.Flags().Lookup("webhook-address"))

	telegramCmd.AddCommand(telegramServeCmd)
	RootCmd.AddCommand(telegramCmd)
}

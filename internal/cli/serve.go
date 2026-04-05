package cli

import (
	"context"
	"fmt"
	"log/slog"
	stdhttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/andrewhowdencom/dux/internal/config"
	"github.com/andrewhowdencom/dux/internal/ui/telegram"
	"github.com/andrewhowdencom/dux/internal/ui/web"
	"github.com/andrewhowdencom/stdlib/http"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var (
	serveAgent    string
	serveProvider string
)

var serveCmd = &cobra.Command{
	Use:   "serve [type]",
	Short: "Starts the Dux UI servers defined in config",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigCh
			slog.Info("Shutting down servers...")
			cancel()
			os.Exit(0)
		}()

		uis, err := config.LoadUIs()
		if err != nil {
			return fmt.Errorf("failed to load UI configs: %w", err)
		}

		// Filter
		var activeUIs []config.UIConfig
		for _, uiConf := range uis {
			if len(args) > 0 && uiConf.Type != args[0] {
				continue
			}
			if serveAgent != "" && uiConf.Agent != serveAgent {
				continue
			}
			if serveProvider != "" && uiConf.Provider != serveProvider {
				continue
			}
			activeUIs = append(activeUIs, uiConf)
		}

		if len(activeUIs) == 0 {
			return fmt.Errorf("no matching UI configurations found to start")
		}

		g, gCtx := errgroup.WithContext(ctx)

		for _, uiConf := range activeUIs {
			uiConf := uiConf // Shadow for goroutine
			switch uiConf.Type {
			case "web":
				g.Go(func() error {
					return startWebServer(gCtx, uiConf)
				})
			case "telegram":
				g.Go(func() error {
					return startTelegramServer(gCtx, uiConf)
				})
			default:
				slog.Warn("Unknown UI type in config", "type", uiConf.Type)
			}
		}

		if err := g.Wait(); err != nil {
			return fmt.Errorf("server group exited with error: %w", err)
		}
		return nil
	},
}

func init() {
	serveCmd.Flags().StringVar(&serveAgent, "agent", "", "Filter the loaded UI configs by specific agent")
	serveCmd.Flags().StringVar(&serveProvider, "provider", "", "Filter the loaded UI configs by specific provider")

	RootCmd.AddCommand(serveCmd)
}

func startWebServer(ctx context.Context, uic config.UIConfig) error {
	webCfg, err := uic.ParseWebConfig()
	if err != nil {
		return fmt.Errorf("failed to parse web config: %w", err)
	}

	mux := stdhttp.NewServeMux()
	mux.HandleFunc("/healthz", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// TODO: UnsafeAllTools via Web UI config? We will pass false for now unless configured.
	mux.Handle("/", web.NewMux(agentsFile, uic.Agent, uic.Provider))

	handler := loggingMiddleware(recoveryMiddleware(mux))

	srv, err := http.NewServer(webCfg.Address, handler,
		http.WithWriteTimeout(2*time.Hour),
		http.WithReadTimeout(2*time.Hour),
		http.WithIdleTimeout(2*time.Hour),
	)
	if err != nil {
		return fmt.Errorf("failed to create http server on %s: %w", webCfg.Address, err)
	}

	slog.Info("Starting Web UI server", "address", webCfg.Address, "agent", uic.Agent, "provider", uic.Provider)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Run()
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("web server exited with error: %w", err)
	case <-ctx.Done():
		return nil // context properly cancelled by errgroup
	}
}

func startTelegramServer(ctx context.Context, uic config.UIConfig) error {
	tgCfg, err := uic.ParseTelegramConfig()
	if err != nil {
		return fmt.Errorf("failed to parse telegram config: %w", err)
	}

	cfg := telegram.Config{
		Token:          tgCfg.Token,
		WebhookURL:     tgCfg.WebhookURL,
		WebhookAddress: tgCfg.WebhookAddress,
		AllowedUsers:   tgCfg.AllowedUsers,
		AgentsFile:     agentsFile,
		AgentName:      uic.Agent,
		ProviderID:     uic.Provider,
	}

	srv, err := telegram.NewServer(cfg)
	if err != nil {
		return fmt.Errorf("failed to create telegram server: %w", err)
	}

	if cfg.WebhookURL != "" {
		address := cfg.WebhookAddress
		if address == "" {
			address = ":8443"
		}
		slog.Info("Starting Telegram Bot with Webhook", "url", cfg.WebhookURL, "address", address, "agent", uic.Agent, "provider", uic.Provider)
		return srv.StartWebhook(ctx, cfg.WebhookURL, address)
	}

	slog.Info("Starting Telegram Bot with Long-Polling", "agent", uic.Agent, "provider", uic.Provider)
	return srv.Start(ctx)
}

func loggingMiddleware(next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Debug("request completed", "method", r.Method, "url", r.URL.Path, "duration", time.Since(start))
	})
}

func recoveryMiddleware(next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered during http block", "panic", rec, "method", r.Method, "url", r.URL.Path)
				stdhttp.Error(w, "Internal Server Error", stdhttp.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

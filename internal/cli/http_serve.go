package cli

import (
	"fmt"
	"log/slog"
	stdhttp "net/http"
	"time"

	"github.com/andrewhowdencom/dux/internal/ui/web"
	"github.com/andrewhowdencom/stdlib/http"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var httpServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts an HTTP server",
	RunE: func(cmd *cobra.Command, args []string) error {
		mux := stdhttp.NewServeMux()
		mux.HandleFunc("/healthz", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			w.WriteHeader(stdhttp.StatusOK)
			_, _ = w.Write([]byte("OK"))
		})

		if cmd.Flags().Lookup("with-ui").Changed {
			if withUI, _ := cmd.Flags().GetBool("with-ui"); withUI {
				mux.Handle("/", web.NewMux(agentsFile))
			}
		}

		handler := loggingMiddleware(recoveryMiddleware(mux))

		address := viper.GetString("http.address")

		srv, err := http.NewServer(address, handler,
			http.WithWriteTimeout(2*time.Hour),
			http.WithReadTimeout(2*time.Hour),
			http.WithIdleTimeout(2*time.Hour),
		)
		if err != nil {
			return fmt.Errorf("failed to create server: %w", err)
		}

		fmt.Printf("Starting server on %s\n", address)
		if err := srv.Run(); err != nil {
			return fmt.Errorf("server exited with error: %w", err)
		}
		return nil
	},
}

var withUI bool

func init() {
	httpServeCmd.Flags().String("address", ":8080", "Address to listen on for the HTTP server")
	_ = viper.BindPFlag("http.address", httpServeCmd.Flags().Lookup("address"))
	httpServeCmd.Flags().BoolVar(&withUI, "with-ui", false, "Start the web UI along with the HTTP server")
	httpCmd.AddCommand(httpServeCmd)
}

func loggingMiddleware(next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		start := time.Now()
		slog.Debug("incoming request", "method", r.Method, "url", r.URL.Path, "remote_addr", r.RemoteAddr)

		next.ServeHTTP(w, r)

		slog.Info("request completed", "method", r.Method, "url", r.URL.Path, "duration", time.Since(start))
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

package cli

import (
	"fmt"
	stdhttp "net/http"

	"github.com/andrewhowdencom/stdlib/http"
	"github.com/spf13/cobra"
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

		srv, err := http.NewServer(":8080", mux)
		if err != nil {
			return fmt.Errorf("failed to create server: %w", err)
		}

		fmt.Println("Starting server on :8080")
		if err := srv.Run(); err != nil {
			return fmt.Errorf("server exited with error: %w", err)
		}
		return nil
	},
}

func init() {
	httpCmd.AddCommand(httpServeCmd)
}

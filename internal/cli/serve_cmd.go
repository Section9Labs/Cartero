package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Section9Labs/Cartero/internal/web"
	"github.com/spf13/cobra"
)

func newServeCmd(streams IOStreams, opts *rootOptions) *cobra.Command {
	var addr string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the local Cartero admin UI",
		Long:  "Run a local web admin for templates, audiences, imports, campaign snapshots, and safe testing pages backed by the active workspace.",
		Example: strings.Join([]string{
			"cartero serve",
			"cartero serve --addr 127.0.0.1:8080",
			"cartero --root /path/to/workspace serve --addr :8080",
		}, "\n"),
		RunE: func(_ *cobra.Command, _ []string) error {
			root, err := resolveRoot(opts.root)
			if err != nil {
				return err
			}

			s, err := prepareWorkspaceStore(root)
			if err != nil {
				return err
			}
			defer s.Close()

			app, err := web.New(root, s)
			if err != nil {
				return err
			}

			server := &http.Server{
				Addr:              addr,
				Handler:           app.Handler(),
				ReadHeaderTimeout: 5 * time.Second,
			}

			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			go func() {
				<-ctx.Done()
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = server.Shutdown(shutdownCtx)
			}()

			fmt.Fprintf(streams.Out, "Cartero admin available at http://%s\n", displayAddress(addr))
			fmt.Fprintln(streams.Out, "Press Ctrl+C to stop.")

			err = server.ListenAndServe()
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			return err
		},
	}
	cmd.Flags().StringVar(&addr, "addr", "127.0.0.1:8080", "listen address for the local admin server")

	return cmd
}

func displayAddress(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "127.0.0.1" + addr
	}
	if strings.HasPrefix(addr, "0.0.0.0:") {
		return "127.0.0.1:" + strings.TrimPrefix(addr, "0.0.0.0:")
	}
	return addr
}

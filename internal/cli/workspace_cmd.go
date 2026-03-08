package cli

import (
	"fmt"
	"strings"

	"github.com/Section9Labs/Cartero/internal/catalog"
	"github.com/Section9Labs/Cartero/internal/ui"
	"github.com/spf13/cobra"
)

func newWorkspaceCmd(streams IOStreams, opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Bootstrap and inspect the local workspace",
		Long:  "Create the embedded Cartero workspace state and inspect the current local database-backed environment.",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "Initialize the embedded workspace state",
		Long:  "Create the embedded workspace database, sync first-party plugin manifests, and seed the built-in template library.",
		Example: strings.Join([]string{
			"cartero workspace init",
			"cartero --root /path/to/workspace workspace init",
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

			stats, err := s.Stats()
			if err != nil {
				return err
			}

			fmt.Fprintln(streams.Out, ui.NewRenderer(opts.plain).WorkspaceInit(root, stats.DatabasePath, len(catalog.BuiltinManifests()), stats.TemplateCount))
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show workspace state counts",
		Long:  "Show the embedded database path and current template, audience, import, campaign, and event counts for the workspace.",
		Example: strings.Join([]string{
			"cartero workspace status",
			"cartero --root /path/to/workspace workspace status",
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

			stats, err := s.Stats()
			if err != nil {
				return err
			}

			fmt.Fprintln(streams.Out, ui.NewRenderer(opts.plain).WorkspaceStatus(root, stats))
			return nil
		},
	})

	return cmd
}

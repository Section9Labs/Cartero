package cli

import (
	"fmt"
	"strings"

	"github.com/Section9Labs/Cartero/internal/audience"
	"github.com/Section9Labs/Cartero/internal/catalog"
	"github.com/Section9Labs/Cartero/internal/store"
	"github.com/Section9Labs/Cartero/internal/ui"
	"github.com/Section9Labs/Cartero/internal/workspace"
	"github.com/spf13/cobra"
)

func newAudienceCmd(streams IOStreams, opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audience",
		Short: "Manage workspace audiences",
		Long:  "Import and inspect local audience segments stored in the embedded Cartero workspace database.",
	}

	var csvPath string
	var segment string
	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Import a segment from CSV",
		Long:  "Import audience members from an operator-reviewed CSV file using the first-party audience-sync plugin.",
		Example: strings.Join([]string{
			"cartero audience import --segment finance-emea --csv audiences/finance-emea.csv",
		}, "\n"),
		RunE: func(_ *cobra.Command, _ []string) error {
			if strings.TrimSpace(segment) == "" {
				return fmt.Errorf("audience segment is required; use --segment <value>")
			}
			if strings.TrimSpace(csvPath) == "" {
				return fmt.Errorf("csv file is required; use --csv <path>")
			}

			root, err := resolveRoot(opts.root)
			if err != nil {
				return err
			}

			members, err := audience.LoadCSV(workspace.ResolveInputPath(root, csvPath))
			if err != nil {
				return err
			}

			s, err := prepareWorkspaceStore(root)
			if err != nil {
				return err
			}
			defer s.Close()

			result, err := s.UpsertAudienceMembers(segment, catalog.PluginAudienceSync, members)
			if err != nil {
				return err
			}

			fmt.Fprintln(streams.Out, ui.NewRenderer(opts.plain).AudienceImport(result))
			return nil
		},
	}
	importCmd.Flags().StringVar(&segment, "segment", "", "audience segment name")
	importCmd.Flags().StringVar(&csvPath, "csv", "", "path to a CSV file with an email column")
	cmd.AddCommand(importCmd)

	var filterSegment string
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List audience members",
		Long:  "List audience members from the embedded workspace database, optionally filtered to a segment.",
		Example: strings.Join([]string{
			"cartero audience list",
			"cartero audience list --segment finance-emea",
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

			members, err := s.ListAudienceMembers(store.AudienceFilter{Segment: filterSegment})
			if err != nil {
				return err
			}

			fmt.Fprintln(streams.Out, ui.NewRenderer(opts.plain).AudienceMembers(members))
			return nil
		},
	}
	listCmd.Flags().StringVar(&filterSegment, "segment", "", "segment name to filter on")
	cmd.AddCommand(listCmd)

	return cmd
}

package cli

import (
	"fmt"
	"strings"

	"github.com/Section9Labs/Cartero/internal/legacy"
	"github.com/Section9Labs/Cartero/internal/ui"
	"github.com/Section9Labs/Cartero/internal/workspace"
	"github.com/spf13/cobra"
)

func newMigrateCmd(streams IOStreams, opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Import legacy data into the workspace",
		Long:  "Import data from previous Cartero-era exports into the current embedded workspace store.",
	}

	var path string
	var segment string
	mongoCmd := &cobra.Command{
		Use:   "mongo-export",
		Short: "Import old Mongo export files",
		Long:  "Import legacy people, hits, and credential export files from an old Cartero Mongo workspace into the current SQLite workspace.",
		Example: strings.Join([]string{
			"cartero migrate mongo-export --path legacy-export",
			"cartero migrate mongo-export --path exports/people.json --segment legacy-users",
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

			resolvedPath := workspace.ResolveInputPath(root, path)
			report, err := legacy.ImportMongoExport(s, legacy.MongoImportOptions{
				Path:    resolvedPath,
				Segment: segment,
			})
			if err != nil {
				return err
			}

			summary := strings.Join([]string{
				fmt.Sprintf("Files processed: %d", report.FilesProcessed),
				fmt.Sprintf("Audience created: %d", report.AudienceCreated),
				fmt.Sprintf("Audience updated: %d", report.AudienceUpdated),
				fmt.Sprintf("Events imported: %d", report.EventsImported),
				fmt.Sprintf("Findings created: %d", report.FindingsCreated),
				fmt.Sprintf("Findings updated: %d", report.FindingsUpdated),
				fmt.Sprintf("Redacted credential artifacts: %d", report.RedactedCredentials),
			}, "\n")
			fmt.Fprintln(streams.Out, ui.NewRenderer(opts.plain).MigrationReport(resolvedPath, summary))
			return nil
		},
	}
	mongoCmd.Flags().StringVar(&path, "path", "", "path to a mongo export directory or JSON export file")
	mongoCmd.Flags().StringVar(&segment, "segment", "legacy-import", "segment to use when importing legacy people records")
	_ = mongoCmd.MarkFlagRequired("path")

	cmd.AddCommand(mongoCmd)

	return cmd
}

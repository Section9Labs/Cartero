package cli

import (
	"fmt"
	"strings"

	"github.com/Section9Labs/Cartero/internal/reporting"
	"github.com/Section9Labs/Cartero/internal/ui"
	"github.com/spf13/cobra"
)

func newReportCmd(streams IOStreams, opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Export workspace analytics",
		Long:  "Build operator-friendly analytics exports from the embedded Cartero workspace database.",
	}

	var format string
	var out string
	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export analytics to JSON or CSV",
		Long:  "Export workspace counts, segment summaries, imports, campaigns, and engagement events to JSON or CSV using the analytics-export plugin.",
		Example: strings.Join([]string{
			"cartero report export --format json",
			"cartero report export --format csv --out exports/weekly.csv",
		}, "\n"),
		RunE: func(_ *cobra.Command, _ []string) error {
			if format != "json" && format != "csv" {
				return fmt.Errorf("unsupported report format %q; use json or csv", format)
			}

			root, err := resolveRoot(opts.root)
			if err != nil {
				return err
			}

			s, err := prepareWorkspaceStore(root)
			if err != nil {
				return err
			}
			defer s.Close()

			export, err := reporting.Build(s)
			if err != nil {
				return err
			}

			target := resolveOutputPath(root, out, "exports", "workspace-report."+format)
			if err := reporting.Write(target, format, export); err != nil {
				return err
			}

			fmt.Fprintln(streams.Out, ui.NewRenderer(opts.plain).ReportExport(target, format, export.Stats))
			return nil
		},
	}
	exportCmd.Flags().StringVar(&format, "format", "json", "export format: json or csv")
	exportCmd.Flags().StringVar(&out, "out", "", "output path for the generated report")
	cmd.AddCommand(exportCmd)

	return cmd
}

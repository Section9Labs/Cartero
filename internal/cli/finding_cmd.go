package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Section9Labs/Cartero/internal/findings"
	"github.com/Section9Labs/Cartero/internal/store"
	"github.com/Section9Labs/Cartero/internal/ui"
	"github.com/Section9Labs/Cartero/internal/workspace"
	"github.com/spf13/cobra"
)

func newFindingCmd(streams IOStreams, opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "finding",
		Short: "Import and inspect external findings",
		Long:  "Import scanner or analysis output into the embedded workspace and inspect the normalized finding registry.",
	}

	var importFile string
	var importSource string
	var importTool string
	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Import findings from CSV, JSON, SARIF, or JSONL",
		Long:  "Load scanner or analysis output from disk, normalize it, and persist the findings in the active Cartero workspace.",
		Example: strings.Join([]string{
			"cartero finding import --file scans/nuclei.jsonl --source nightly-nuclei",
			"cartero finding import --file reports/findings.csv --tool nessus",
		}, "\n"),
		RunE: func(_ *cobra.Command, _ []string) error {
			if importFile == "" {
				return errors.New("findings file is required; use --file <path>")
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

			resolvedPath := workspace.ResolveInputPath(root, importFile)
			loaded, err := findings.Load(resolvedPath, importSource, importTool)
			if err != nil {
				return err
			}
			if len(loaded) == 0 {
				return errors.New("no findings were parsed from the input file")
			}

			source := importSource
			if source == "" {
				source = loaded[0].Source
			}
			result, err := s.ImportFindings(source, loaded)
			if err != nil {
				return err
			}

			tool := importTool
			if tool == "" {
				tool = loaded[0].Tool
			}
			fmt.Fprintln(streams.Out, ui.NewRenderer(opts.plain).FindingImport(resolvedPath, source, tool, result))
			return nil
		},
	}
	importCmd.Flags().StringVar(&importFile, "file", "", "path to the findings file to import")
	importCmd.Flags().StringVar(&importSource, "source", "", "source label to attach to the imported findings")
	importCmd.Flags().StringVar(&importTool, "tool", "", "override the tool name during import")
	cmd.AddCommand(importCmd)

	var listSource string
	var listTool string
	var listSeverity string
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Show imported findings",
		Long:  "List the normalized findings currently stored in the active workspace.",
		Example: strings.Join([]string{
			"cartero finding list",
			"cartero finding list --tool nuclei --severity high",
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

			findings, err := s.ListFindings(store.FindingFilter{
				Source:   listSource,
				Tool:     listTool,
				Severity: listSeverity,
			})
			if err != nil {
				return err
			}
			fmt.Fprintln(streams.Out, ui.NewRenderer(opts.plain).Findings(findings))
			return nil
		},
	}
	listCmd.Flags().StringVar(&listSource, "source", "", "filter findings by source label")
	listCmd.Flags().StringVar(&listTool, "tool", "", "filter findings by tool name")
	listCmd.Flags().StringVar(&listSeverity, "severity", "", "filter findings by severity")
	cmd.AddCommand(listCmd)

	return cmd
}

package cli

import (
	"fmt"
	"strings"

	"github.com/Section9Labs/Cartero/internal/store"
	"github.com/Section9Labs/Cartero/internal/ui"
	"github.com/spf13/cobra"
)

func newTemplateCmd(streams IOStreams, opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Browse the built-in template library",
		Long:  "List and inspect first-party template content seeded by the template-library plugin.",
	}

	var locale string
	var department string
	var scenario string
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available templates",
		Long:  "List templates from the embedded workspace database with optional locale, department, and scenario filtering.",
		Example: strings.Join([]string{
			"cartero template list",
			"cartero template list --department Finance",
			"cartero template list --locale en-GB --scenario invoice-fraud",
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

			templates, err := s.ListTemplates(store.TemplateFilter{
				Locale:     locale,
				Department: department,
				Scenario:   scenario,
			})
			if err != nil {
				return err
			}

			fmt.Fprintln(streams.Out, ui.NewRenderer(opts.plain).Templates(templates))
			return nil
		},
	}
	listCmd.Flags().StringVar(&locale, "locale", "", "filter templates by locale")
	listCmd.Flags().StringVar(&department, "department", "", "filter templates by department")
	listCmd.Flags().StringVar(&scenario, "scenario", "", "filter templates by scenario")
	cmd.AddCommand(listCmd)

	var slug string
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show a template in detail",
		Long:  "Show the subject, channels, body summary, and landing content for a single template slug.",
		Example: strings.Join([]string{
			"cartero template show --slug finance-payroll-review",
		}, "\n"),
		RunE: func(_ *cobra.Command, _ []string) error {
			if strings.TrimSpace(slug) == "" {
				return fmt.Errorf("template slug is required; use --slug <value>")
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

			templates, err := s.ListTemplates(store.TemplateFilter{})
			if err != nil {
				return err
			}
			for _, template := range templates {
				if template.Slug == slug {
					fmt.Fprintln(streams.Out, ui.NewRenderer(opts.plain).TemplateDetail(template))
					return nil
				}
			}

			return fmt.Errorf("template %q was not found", slug)
		},
	}
	showCmd.Flags().StringVar(&slug, "slug", "", "template slug to inspect")
	cmd.AddCommand(showCmd)

	return cmd
}

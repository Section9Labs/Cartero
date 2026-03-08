package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Section9Labs/Cartero/internal/app"
	"github.com/Section9Labs/Cartero/internal/catalog"
	"github.com/Section9Labs/Cartero/internal/doctor"
	"github.com/Section9Labs/Cartero/internal/plugin"
	"github.com/Section9Labs/Cartero/internal/ui"
	"github.com/Section9Labs/Cartero/internal/version"
	"github.com/Section9Labs/Cartero/internal/workspace"
	"github.com/spf13/cobra"
)

type IOStreams struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
}

type rootOptions struct {
	plain bool
	root  string
}

func Execute(streams IOStreams, build version.Info) error {
	return NewRootCmd(streams, build).Execute()
}

func NewRootCmd(streams IOStreams, build version.Info) *cobra.Command {
	opts := &rootOptions{}

	rootCmd := &cobra.Command{
		Use:   "cartero",
		Short: "Plan and validate security awareness exercises",
		Long:  "Cartero provides a modern CLI for campaign planning, safe validation, plugin discovery, and release-friendly local workflows.",
		Example: strings.Join([]string{
			"cartero workspace init",
			"cartero init",
			"cartero preview -f configs/campaign.example.yaml",
			"cartero validate -f configs/campaign.example.yaml",
			"cartero serve --addr 127.0.0.1:8080",
			"cartero template list",
			"cartero audience import --segment finance-emea --csv audiences/finance-emea.csv",
			"cartero import clone -f samples/reported.eml",
			"cartero finding import --file scans/nuclei.jsonl --source nightly-nuclei",
			"cartero migrate mongo-export --path legacy-export",
			"cartero report export --format json",
			"cartero doctor",
			"cartero plugin list",
			"cartero --root /path/to/workspace doctor",
		}, "\n"),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	rootCmd.SetIn(streams.In)
	rootCmd.SetOut(streams.Out)
	rootCmd.SetErr(streams.Err)
	rootCmd.PersistentFlags().BoolVar(&opts.plain, "plain", false, "disable styled terminal output")
	rootCmd.PersistentFlags().StringVar(&opts.root, "root", "", "workspace root to inspect")
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		renderer := ui.NewRenderer(opts.plain)
		fmt.Fprintln(streams.Out, renderer.Help(helpSpecFor(cmd)))
	})

	rootCmd.AddCommand(newInitCmd(streams, opts))
	rootCmd.AddCommand(newWorkspaceCmd(streams, opts))
	rootCmd.AddCommand(newPreviewCmd(streams, opts))
	rootCmd.AddCommand(newValidateCmd(streams, opts))
	rootCmd.AddCommand(newServeCmd(streams, opts))
	rootCmd.AddCommand(newTemplateCmd(streams, opts))
	rootCmd.AddCommand(newAudienceCmd(streams, opts))
	rootCmd.AddCommand(newImportCmd(streams, opts))
	rootCmd.AddCommand(newFindingCmd(streams, opts))
	rootCmd.AddCommand(newMigrateCmd(streams, opts))
	rootCmd.AddCommand(newReportCmd(streams, opts))
	rootCmd.AddCommand(newEventCmd(streams, opts))
	rootCmd.AddCommand(newDoctorCmd(streams, opts))
	rootCmd.AddCommand(newPluginCmd(streams, opts))
	rootCmd.AddCommand(newVersionCmd(streams, opts, build))

	return rootCmd
}

func newInitCmd(streams IOStreams, opts *rootOptions) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init [path]",
		Short: "Write a starter campaign file",
		Long:  "Create a starter campaign YAML file that operators can tailor before previewing or validating the exercise.",
		Example: strings.Join([]string{
			"cartero init",
			"cartero init drafts/q3-campaign.yaml",
			"cartero init --force campaign.yaml",
		}, "\n"),
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			target := "campaign.yaml"
			if len(args) == 1 {
				target = args[0]
			}
			if !force {
				if _, err := os.Stat(target); err == nil {
					return fmt.Errorf("%s already exists; use --force to overwrite", target)
				}
			}
			if err := os.WriteFile(target, []byte(app.ExampleCampaignYAML), 0o644); err != nil {
				return fmt.Errorf("write starter campaign: %w", err)
			}

			fmt.Fprintln(streams.Out, ui.NewRenderer(opts.plain).Init(target))
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite the target file if it already exists")

	return cmd
}

func newPreviewCmd(streams IOStreams, opts *rootOptions) *cobra.Command {
	var file string

	cmd := &cobra.Command{
		Use:   "preview",
		Short: "Render a campaign readiness preview",
		Long:  "Load a campaign definition, evaluate safety and completeness checks, and render a readiness overview for operator review.",
		Example: strings.Join([]string{
			"cartero preview -f campaign.yaml",
			"cartero --root /path/to/workspace preview -f configs/campaign.example.yaml",
		}, "\n"),
		RunE: func(_ *cobra.Command, _ []string) error {
			if file == "" {
				return errors.New("campaign file is required; use -f <path>")
			}
			root, err := resolveRoot(opts.root)
			if err != nil {
				return err
			}
			campaign, err := app.LoadCampaign(workspace.ResolveInputPath(root, file))
			if err != nil {
				return err
			}
			issues := app.ValidateCampaign(campaign)
			score := app.ReadinessScore(issues)
			if err := persistCampaignSnapshot(root, workspace.ResolveInputPath(root, file), campaign, score, issues, "preview"); err != nil {
				return err
			}

			fmt.Fprintln(streams.Out, ui.NewRenderer(opts.plain).CampaignPreview(campaign, score, issues))
			if app.HasErrors(issues) {
				return errors.New("campaign preview completed with validation errors")
			}

			return nil
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "path to a campaign YAML file")

	return cmd
}

func newValidateCmd(streams IOStreams, opts *rootOptions) *cobra.Command {
	var file string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Lint a campaign definition",
		Long:  "Validate a campaign YAML file and return a non-zero exit code when blocking issues are present.",
		Example: strings.Join([]string{
			"cartero validate -f campaign.yaml",
			"cartero --root /path/to/workspace validate -f configs/campaign.example.yaml",
		}, "\n"),
		RunE: func(_ *cobra.Command, _ []string) error {
			if file == "" {
				return errors.New("campaign file is required; use -f <path>")
			}
			root, err := resolveRoot(opts.root)
			if err != nil {
				return err
			}
			campaign, err := app.LoadCampaign(workspace.ResolveInputPath(root, file))
			if err != nil {
				return err
			}
			issues := app.ValidateCampaign(campaign)
			score := app.ReadinessScore(issues)
			if err := persistCampaignSnapshot(root, workspace.ResolveInputPath(root, file), campaign, score, issues, "validate"); err != nil {
				return err
			}
			fmt.Fprintln(streams.Out, ui.NewRenderer(opts.plain).Validation(issues))
			if app.HasErrors(issues) {
				return errors.New("validation failed")
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "path to a campaign YAML file")

	return cmd
}

func newDoctorCmd(streams IOStreams, opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Inspect local project health",
		Long:  "Inspect the active workspace, auto-discover the Cartero project root, and report missing files or malformed local setup.",
		Example: strings.Join([]string{
			"cartero doctor",
			"cartero --root /path/to/workspace doctor",
		}, "\n"),
		RunE: func(_ *cobra.Command, _ []string) error {
			root, err := resolveRoot(opts.root)
			if err != nil {
				return err
			}
			report := doctor.Run(root)
			fmt.Fprintln(streams.Out, ui.NewRenderer(opts.plain).DoctorReport(report))
			if doctor.HasFailures(report) {
				return errors.New("doctor found failing checks")
			}
			return nil
		},
	}
}

func newPluginCmd(streams IOStreams, opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Inspect local plugin manifests",
		Long:  "Inspect plugin manifests within the active Cartero workspace.",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "Show discovered plugin manifests",
		Long:  "Discover and list plugin manifests from the resolved workspace root.",
		Example: strings.Join([]string{
			"cartero plugin list",
			"cartero --root /path/to/workspace plugin list",
		}, "\n"),
		RunE: func(_ *cobra.Command, _ []string) error {
			root, err := resolveRoot(opts.root)
			if err != nil {
				return err
			}
			discovery, err := plugin.Discover(filepath.Join(root, "plugins"))
			if err != nil {
				return err
			}
			fmt.Fprintln(streams.Out, ui.NewRenderer(opts.plain).Plugins(discovery.Manifests, discovery.Warnings))
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "sync",
		Short: "Sync built-in plugin manifests",
		Long:  "Write the built-in first-party plugin manifests into the resolved workspace and seed template content when needed.",
		Example: strings.Join([]string{
			"cartero plugin sync",
			"cartero --root /path/to/workspace plugin sync",
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

	return cmd
}

func newVersionCmd(streams IOStreams, opts *rootOptions, build version.Info) *cobra.Command {
	return &cobra.Command{
		Use:     "version",
		Short:   "Print build metadata",
		Long:    "Print build metadata embedded at compile time.",
		Example: "cartero version",
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Fprintln(streams.Out, ui.NewRenderer(opts.plain).Version(build.Version, build.Commit, build.Date))
			return nil
		},
	}
}

func resolveRoot(explicit string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}

	root, err := workspace.ResolveRoot(cwd, explicit)
	if err != nil {
		return "", err
	}

	return root, nil
}

func helpSpecFor(cmd *cobra.Command) ui.HelpSpec {
	commands := make([]ui.CommandInfo, 0, len(cmd.Commands()))
	for _, child := range cmd.Commands() {
		if child.Hidden || !child.IsAvailableCommand() {
			continue
		}
		commands = append(commands, ui.CommandInfo{
			Use:   child.UseLine(),
			Short: child.Short,
		})
	}

	return ui.HelpSpec{
		Title:          cmd.CommandPath(),
		Summary:        cmd.Short,
		Details:        cleanDetails(cmd.Short, cmd.Long),
		Usage:          cmd.UseLine(),
		Commands:       commands,
		Flags:          splitLines(cmd.NonInheritedFlags().FlagUsagesWrapped(80)),
		InheritedFlags: splitLines(cmd.InheritedFlags().FlagUsagesWrapped(80)),
		Examples:       splitLines(cmd.Example),
	}
}

func cleanDetails(short, long string) string {
	details := strings.TrimSpace(long)
	if details == "" || details == strings.TrimSpace(short) {
		return ""
	}

	return details
}

func splitLines(block string) []string {
	raw := strings.Split(strings.TrimSpace(block), "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimRight(line, " ")
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines = append(lines, line)
	}

	return lines
}

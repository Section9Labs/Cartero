package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/Section9Labs/Cartero/internal/app"
	"github.com/Section9Labs/Cartero/internal/catalog"
	"github.com/Section9Labs/Cartero/internal/clone"
	"github.com/Section9Labs/Cartero/internal/store"
	"github.com/Section9Labs/Cartero/internal/ui"
	"github.com/Section9Labs/Cartero/internal/workspace"
	"github.com/spf13/cobra"
)

func newImportCmd(streams IOStreams, opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import reviewed source material into Cartero",
		Long:  "Import reviewed local source material and convert it into safe Cartero workspace artifacts.",
	}

	var file string
	var out string
	cloneCmd := &cobra.Command{
		Use:   "clone",
		Short: "Clone a reviewed message into a draft campaign",
		Long:  "Parse a local .eml or raw message file, generate a safe Cartero campaign draft, write it to disk, and store the import in the embedded workspace database.",
		Example: strings.Join([]string{
			"cartero import clone -f samples/reported.eml",
			"cartero import clone -f samples/reported.txt --out drafts/payroll-review.yaml",
		}, "\n"),
		RunE: func(_ *cobra.Command, _ []string) error {
			if strings.TrimSpace(file) == "" {
				return fmt.Errorf("source file is required; use -f <path>")
			}

			root, err := resolveRoot(opts.root)
			if err != nil {
				return err
			}

			message, err := clone.LoadMessage(workspace.ResolveInputPath(root, file))
			if err != nil {
				return err
			}

			campaign := clone.DraftCampaign(message)
			payload, err := clone.DraftYAML(campaign)
			if err != nil {
				return err
			}

			target := resolveOutputPath(root, out, "drafts", clone.OutputFilename(message))
			if err := ensureParentDir(target); err != nil {
				return err
			}
			if err := os.WriteFile(target, []byte(payload), 0o644); err != nil {
				return fmt.Errorf("write imported draft: %w", err)
			}

			issues := app.ValidateCampaign(campaign)
			score := app.ReadinessScore(issues)

			s, err := prepareWorkspaceStore(root)
			if err != nil {
				return err
			}
			defer s.Close()

			if _, err := s.SaveImportedMessage(store.ImportedMessage{
				Plugin:            catalog.PluginCloneImporter,
				SourcePath:        message.SourcePath,
				Sender:            message.Sender,
				Subject:           message.Subject,
				Body:              message.Body,
				GeneratedCampaign: payload,
			}); err != nil {
				return err
			}

			if _, err := s.SaveCampaignSnapshot(store.CampaignSnapshot{
				Name:       campaign.Metadata.Name,
				SourcePath: target,
				Owner:      campaign.Metadata.Owner,
				Audience:   campaign.Metadata.Audience,
				Region:     campaign.Metadata.Region,
				RiskLevel:  campaign.Metadata.RiskLevel,
				Readiness:  score,
				IssueCount: len(issues),
				Issues:     issueStrings(issues),
				Source:     catalog.PluginCloneImporter,
			}); err != nil {
				return err
			}

			fmt.Fprintln(streams.Out, ui.NewRenderer(opts.plain).CloneImport(message.SourcePath, message.Subject, target, score))
			return nil
		},
	}
	cloneCmd.Flags().StringVarP(&file, "file", "f", "", "path to a reviewed source message")
	cloneCmd.Flags().StringVar(&out, "out", "", "output path for the generated campaign draft")
	cmd.AddCommand(cloneCmd)

	return cmd
}

package cli

import (
	"fmt"
	"strings"

	"github.com/Section9Labs/Cartero/internal/catalog"
	"github.com/Section9Labs/Cartero/internal/store"
	"github.com/Section9Labs/Cartero/internal/ui"
	"github.com/spf13/cobra"
)

var allowedEventTypes = map[string]struct{}{
	"clicked":            {},
	"opened":             {},
	"reported":           {},
	"completed-training": {},
}

func newEventCmd(streams IOStreams, opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "event",
		Short: "Record engagement events",
		Long:  "Record safe engagement events in the embedded workspace store for reporting and rehearsal analytics.",
	}

	var campaignName string
	var email string
	var eventType string
	var source string
	recordCmd := &cobra.Command{
		Use:   "record",
		Short: "Record an engagement event",
		Long:  "Record an engagement event such as reported, opened, clicked, or completed-training.",
		Example: strings.Join([]string{
			"cartero event record --campaign \"Q2 Awareness Rehearsal\" --email analyst@example.com --type reported",
		}, "\n"),
		RunE: func(_ *cobra.Command, _ []string) error {
			if strings.TrimSpace(campaignName) == "" {
				return fmt.Errorf("campaign name is required; use --campaign <value>")
			}
			if strings.TrimSpace(email) == "" {
				return fmt.Errorf("audience email is required; use --email <value>")
			}
			if _, ok := allowedEventTypes[eventType]; !ok {
				return fmt.Errorf("unsupported event type %q", eventType)
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

			event, err := s.SaveEvent(store.Event{
				CampaignName:  campaignName,
				AudienceEmail: email,
				Type:          eventType,
				Source:        source,
			})
			if err != nil {
				return err
			}

			fmt.Fprintln(streams.Out, ui.NewRenderer(opts.plain).EventRecorded(event))
			return nil
		},
	}
	recordCmd.Flags().StringVar(&campaignName, "campaign", "", "campaign name for the event")
	recordCmd.Flags().StringVar(&email, "email", "", "audience email")
	recordCmd.Flags().StringVar(&eventType, "type", "", "event type: reported, opened, clicked, completed-training")
	recordCmd.Flags().StringVar(&source, "source", catalog.PluginEngagementRecorder, "event source label")
	cmd.AddCommand(recordCmd)

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List recorded engagement events",
		Long:  "List engagement events currently stored in the embedded workspace database.",
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

			events, err := s.ListEvents()
			if err != nil {
				return err
			}

			fmt.Fprintln(streams.Out, ui.NewRenderer(opts.plain).Events(events))
			return nil
		},
	}
	cmd.AddCommand(listCmd)

	return cmd
}

package catalog

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Section9Labs/Cartero/internal/plugin"
	"github.com/Section9Labs/Cartero/internal/store"
	"gopkg.in/yaml.v3"
)

const (
	PluginLocalPreview       = "local-preview"
	PluginTemplateLibrary    = "template-library"
	PluginCloneImporter      = "clone-importer"
	PluginAnalyticsExport    = "analytics-export"
	PluginAudienceSync       = "audience-sync"
	PluginEngagementRecorder = "engagement-recorder"
)

func BuiltinManifests() []plugin.Manifest {
	return []plugin.Manifest{
		{
			SchemaVersion: plugin.SchemaVersionV1,
			Name:          PluginLocalPreview,
			Version:       "1.0.0",
			Kind:          "renderer",
			Mode:          "local-only",
			Safe:          boolPtr(true),
			Capabilities:  []string{"preview.render"},
			Trust:         plugin.Trust{Level: "first-party", ReviewRequired: boolPtr(false)},
			Description:   "Writes previews into a local review sink for operator approval.",
		},
		{
			SchemaVersion: plugin.SchemaVersionV1,
			Name:          PluginTemplateLibrary,
			Version:       "1.0.0",
			Kind:          "content-pack",
			Mode:          "local-only",
			Safe:          boolPtr(true),
			Capabilities:  []string{"campaign.template"},
			Trust:         plugin.Trust{Level: "first-party", ReviewRequired: boolPtr(false)},
			Description:   "Seeds a curated template pack for common training and rehearsal scenarios.",
		},
		{
			SchemaVersion: plugin.SchemaVersionV1,
			Name:          PluginCloneImporter,
			Version:       "1.0.0",
			Kind:          "importer",
			Mode:          "operator-review",
			Safe:          boolPtr(true),
			Capabilities:  []string{"campaign.import"},
			Trust:         plugin.Trust{Level: "first-party", ReviewRequired: boolPtr(false)},
			Description:   "Converts reviewed local messages into safe Cartero campaign drafts.",
		},
		{
			SchemaVersion: plugin.SchemaVersionV1,
			Name:          PluginAnalyticsExport,
			Version:       "1.0.0",
			Kind:          "exporter",
			Mode:          "local-only",
			Safe:          boolPtr(true),
			Capabilities:  []string{"results.export"},
			Trust:         plugin.Trust{Level: "first-party", ReviewRequired: boolPtr(false)},
			Description:   "Exports workspace analytics and readiness snapshots to JSON or CSV.",
		},
		{
			SchemaVersion: plugin.SchemaVersionV1,
			Name:          PluginAudienceSync,
			Version:       "1.0.0",
			Kind:          "integration",
			Mode:          "operator-review",
			Safe:          boolPtr(true),
			Capabilities:  []string{"audience.sync"},
			Trust:         plugin.Trust{Level: "first-party", ReviewRequired: boolPtr(false)},
			Description:   "Imports audience segments from operator-reviewed local CSV files.",
		},
		{
			SchemaVersion: plugin.SchemaVersionV1,
			Name:          PluginEngagementRecorder,
			Version:       "1.0.0",
			Kind:          "integration",
			Mode:          "operator-review",
			Safe:          boolPtr(true),
			Capabilities:  []string{"events.ingest"},
			Trust:         plugin.Trust{Level: "first-party", ReviewRequired: boolPtr(false)},
			Description:   "Records engagement events in the embedded workspace store for reporting.",
		},
	}
}

func SyncManifests(root string) (int, error) {
	targetDir := filepath.Join(root, "plugins")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return 0, fmt.Errorf("create plugin directory: %w", err)
	}

	count := 0
	for _, manifest := range BuiltinManifests() {
		payload, err := yaml.Marshal(manifest)
		if err != nil {
			return count, fmt.Errorf("encode builtin manifest %s: %w", manifest.Name, err)
		}

		path := filepath.Join(targetDir, manifest.Name+".yaml")
		if err := os.WriteFile(path, payload, 0o644); err != nil {
			return count, fmt.Errorf("write builtin manifest %s: %w", manifest.Name, err)
		}
		count++
	}

	return count, nil
}

func SeedTemplateLibrary(s *store.Store) (store.SeedReport, error) {
	return s.SeedTemplates(PluginTemplateLibrary, []store.Template{
		{
			Slug:        "finance-payroll-review",
			Name:        "Finance Payroll Review",
			Locale:      "en-US",
			Department:  "Finance",
			Scenario:    "approval-bypass",
			Channels:    []string{"email"},
			Subject:     "Action required: payroll review before close",
			Body:        "Please review the attached payroll summary before end of day and report any unexpected approval requests through the official escalation path.",
			LandingPage: "Manager approval confirmation and reporting reminder.",
		},
		{
			Slug:        "support-ticket-escalation",
			Name:        "Support Ticket Escalation",
			Locale:      "en-US",
			Department:  "Customer Support",
			Scenario:    "urgency-lure",
			Channels:    []string{"email", "chat"},
			Subject:     "Urgent: executive escalation on ticket backlog",
			Body:        "Responders verify urgency claims, pause before opening external systems, and route the request to approved support leadership contacts.",
			LandingPage: "Escalation path refresher with manager verification checklist.",
		},
		{
			Slug:        "hr-benefits-qna",
			Name:        "HR Benefits Q&A",
			Locale:      "en-US",
			Department:  "People Operations",
			Scenario:    "credential-harvest-pretext",
			Channels:    []string{"email", "intranet"},
			Subject:     "Benefits enrollment follow-up",
			Body:        "Staff rehearse reporting suspicious enrollment prompts and confirm all changes through the internal benefits portal rather than email links.",
			LandingPage: "Benefits verification checklist and reporting steps.",
		},
		{
			Slug:        "emea-vendor-bank-change",
			Name:        "EMEA Vendor Bank Change",
			Locale:      "en-GB",
			Department:  "Procurement",
			Scenario:    "invoice-fraud",
			Channels:    []string{"email"},
			Subject:     "Vendor banking update before payment run",
			Body:        "Procurement analysts verify change requests through known contacts and documented supplier controls before approving payment updates.",
			LandingPage: "Supplier verification workflow and escalation guidance.",
		},
		{
			Slug:        "global-qr-dropoff",
			Name:        "Global QR Drop-off",
			Locale:      "en-US",
			Department:  "All Hands",
			Scenario:    "qr-phishing",
			Channels:    []string{"email", "lms"},
			Subject:     "Poster follow-up: verify the QR destination",
			Body:        "Operators rehearse QR phishing response steps and verify destinations through approved learning and reporting channels.",
			LandingPage: "QR phishing cues and safe reporting examples.",
		},
	})
}

func boolPtr(v bool) *bool {
	return &v
}

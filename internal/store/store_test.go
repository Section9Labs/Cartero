package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	bolt "go.etcd.io/bbolt"
)

func TestStoreSeedsTemplatesAndTracksStats(t *testing.T) {
	root := t.TempDir()
	s, err := Open(root)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})

	report, err := s.SeedTemplates("template-library", []Template{
		{
			Slug:       "payroll-review",
			Name:       "Payroll Review",
			Locale:     "en-US",
			Department: "Finance",
			Scenario:   "approval-bypass",
			Channels:   []string{"email"},
			Subject:    "Review payroll",
			Body:       "Please review payroll",
		},
	})
	if err != nil {
		t.Fatalf("SeedTemplates() error = %v", err)
	}
	if report.Created != 1 || report.Updated != 0 {
		t.Fatalf("unexpected seed report: %+v", report)
	}

	templates, err := s.ListTemplates(TemplateFilter{Department: "finance"})
	if err != nil {
		t.Fatalf("ListTemplates() error = %v", err)
	}
	if len(templates) != 1 || templates[0].Slug != "payroll-review" {
		t.Fatalf("unexpected templates: %+v", templates)
	}

	stats, err := s.Stats()
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if stats.TemplateCount != 1 {
		t.Fatalf("expected 1 template, got %+v", stats)
	}
	if stats.DatabasePath != filepath.Join(root, ".cartero", "cartero.sqlite") {
		t.Fatalf("unexpected database path: %s", stats.DatabasePath)
	}
}

func TestStoreAudienceCampaignsAndEvents(t *testing.T) {
	root := t.TempDir()
	s, err := Open(root)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})

	importResult, err := s.UpsertAudienceMembers("finance-emea", "audience-sync", []AudienceMember{
		{Email: "analyst@example.com", DisplayName: "Analyst", Department: "Finance", Title: "Analyst"},
		{Email: "manager@example.com", DisplayName: "Manager", Department: "Finance", Title: "Manager"},
	})
	if err != nil {
		t.Fatalf("UpsertAudienceMembers() error = %v", err)
	}
	if importResult.Created != 2 {
		t.Fatalf("unexpected import result: %+v", importResult)
	}

	if _, err := s.SaveCampaignSnapshot(CampaignSnapshot{
		Name:       "Quarterly Approval Drill",
		SourcePath: "/tmp/campaign.yaml",
		Owner:      "Security",
		Audience:   "Finance",
		Region:     "EMEA",
		RiskLevel:  "medium",
		Readiness:  94,
		IssueCount: 1,
		Issues:     []string{"warning: assets missing"},
		Source:     "validate",
	}); err != nil {
		t.Fatalf("SaveCampaignSnapshot() error = %v", err)
	}

	if _, err := s.SaveImportedMessage(ImportedMessage{
		Plugin:            "clone-importer",
		SourcePath:        "/tmp/source.eml",
		Sender:            "alerts@example.com",
		Subject:           "Verify your payroll account",
		Body:              "Action required",
		GeneratedCampaign: "metadata:\n  name: imported\n",
	}); err != nil {
		t.Fatalf("SaveImportedMessage() error = %v", err)
	}

	if _, err := s.SaveEvent(Event{
		CampaignName:  "Quarterly Approval Drill",
		AudienceEmail: "analyst@example.com",
		Type:          "reported",
		Source:        "engagement",
	}); err != nil {
		t.Fatalf("SaveEvent() error = %v", err)
	}

	findingResult, err := s.ImportFindings("scan-import", []Finding{{
		Tool:     "nuclei",
		Rule:     "open-redirect",
		Severity: "medium",
		Target:   "https://example.com/login",
		Summary:  "Review redirect handling on the login route.",
		Metadata: map[string]string{"template": "open-redirect"},
	}})
	if err != nil {
		t.Fatalf("ImportFindings() error = %v", err)
	}
	if findingResult.Created != 1 {
		t.Fatalf("unexpected finding import result: %+v", findingResult)
	}

	findings, err := s.ListFindings(FindingFilter{Tool: "nuclei"})
	if err != nil {
		t.Fatalf("ListFindings() error = %v", err)
	}
	if len(findings) != 1 || findings[0].Rule != "open-redirect" {
		t.Fatalf("unexpected findings: %+v", findings)
	}

	segments, err := s.SegmentSummaries()
	if err != nil {
		t.Fatalf("SegmentSummaries() error = %v", err)
	}
	if len(segments) != 1 || segments[0].Members != 2 {
		t.Fatalf("unexpected segment summaries: %+v", segments)
	}

	eventSummaries, err := s.EventSummaries()
	if err != nil {
		t.Fatalf("EventSummaries() error = %v", err)
	}
	if len(eventSummaries) != 1 || eventSummaries[0].Type != "reported" {
		t.Fatalf("unexpected event summaries: %+v", eventSummaries)
	}

	stats, err := s.Stats()
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if stats.AudienceCount != 2 || stats.CampaignCount != 1 || stats.ImportCount != 1 || stats.EventCount != 1 || stats.FindingCount != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestStoreMigratesLegacyBoltWorkspace(t *testing.T) {
	root := t.TempDir()
	legacyPath := LegacyDatabasePath(root)
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	legacy, err := bolt.Open(legacyPath, 0o600, nil)
	if err != nil {
		t.Fatalf("bolt.Open() error = %v", err)
	}
	if err := legacy.Update(func(tx *bolt.Tx) error {
		templates, err := tx.CreateBucketIfNotExists([]byte("templates"))
		if err != nil {
			return err
		}
		events, err := tx.CreateBucketIfNotExists([]byte("events"))
		if err != nil {
			return err
		}

		templatePayload, err := json.Marshal(Template{
			ID:         7,
			Plugin:     "template-library",
			Slug:       "legacy-template",
			Name:       "Legacy Template",
			Locale:     "en-US",
			Department: "Finance",
			Scenario:   "invoice-fraud",
			Channels:   []string{"email"},
			Subject:    "Legacy subject",
			Body:       "Legacy body",
			CreatedAt:  "2026-01-02T03:04:05Z",
		})
		if err != nil {
			return err
		}
		if err := templates.Put([]byte("legacy-template"), templatePayload); err != nil {
			return err
		}

		eventPayload, err := json.Marshal(Event{
			ID:            3,
			CampaignName:  "Legacy Drill",
			AudienceEmail: "analyst@example.com",
			Type:          "reported",
			Source:        "legacy",
			CreatedAt:     "2026-01-03T03:04:05Z",
		})
		if err != nil {
			return err
		}
		return events.Put([]byte("00000000000000000003"), eventPayload)
	}); err != nil {
		t.Fatalf("legacy.Update() error = %v", err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatalf("legacy.Close() error = %v", err)
	}

	s, err := Open(root)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})

	templates, err := s.ListTemplates(TemplateFilter{})
	if err != nil {
		t.Fatalf("ListTemplates() error = %v", err)
	}
	if len(templates) != 1 || templates[0].Slug != "legacy-template" || templates[0].ID != 7 {
		t.Fatalf("unexpected migrated templates: %+v", templates)
	}

	events, err := s.ListEvents()
	if err != nil {
		t.Fatalf("ListEvents() error = %v", err)
	}
	if len(events) != 1 || events[0].CampaignName != "Legacy Drill" || events[0].ID != 3 {
		t.Fatalf("unexpected migrated events: %+v", events)
	}

	if _, err := os.Stat(DatabasePath(root)); err != nil {
		t.Fatalf("expected sqlite database to exist: %v", err)
	}
}

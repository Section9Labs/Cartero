package store

import (
	"path/filepath"
	"testing"
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
	if stats.DatabasePath != filepath.Join(root, ".cartero", "cartero.db") {
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
	if stats.AudienceCount != 2 || stats.CampaignCount != 1 || stats.ImportCount != 1 || stats.EventCount != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

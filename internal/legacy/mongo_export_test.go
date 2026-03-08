package legacy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Section9Labs/Cartero/internal/store"
)

func TestImportMongoExport(t *testing.T) {
	root := t.TempDir()
	exportDir := filepath.Join(root, "mongo")
	if err := os.MkdirAll(exportDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(exportDir, "people.json"), []byte(`[{"email":"analyst@example.com","created_at":"2025-01-02T03:04:05Z"}]`), 0o644); err != nil {
		t.Fatalf("WriteFile(people) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(exportDir, "hits.json"), []byte(`[{"domain":"legacy.example.com","path":"/login","ip":"10.0.0.5","created_at":"2025-01-02T03:04:05Z"}]`), 0o644); err != nil {
		t.Fatalf("WriteFile(hits) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(exportDir, "credentials.json"), []byte(`[{"domain":"legacy.example.com","path":"/login","username":"user","password":"secret","data":{"department":"finance"},"created_at":"2025-01-02T03:04:05Z"}]`), 0o644); err != nil {
		t.Fatalf("WriteFile(credentials) error = %v", err)
	}

	s, err := store.Open(root)
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})

	report, err := ImportMongoExport(s, MongoImportOptions{
		Path:    exportDir,
		Segment: "legacy-users",
	})
	if err != nil {
		t.Fatalf("ImportMongoExport() error = %v", err)
	}
	if report.FilesProcessed != 3 || report.AudienceCreated != 1 || report.EventsImported != 1 || report.FindingsCreated != 1 || report.RedactedCredentials != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}

	audience, err := s.ListAudienceMembers(store.AudienceFilter{Segment: "legacy-users"})
	if err != nil {
		t.Fatalf("ListAudienceMembers() error = %v", err)
	}
	if len(audience) != 1 || audience[0].Email != "analyst@example.com" {
		t.Fatalf("unexpected audience: %+v", audience)
	}

	events, err := s.ListEvents()
	if err != nil {
		t.Fatalf("ListEvents() error = %v", err)
	}
	if len(events) != 1 || events[0].Type != "legacy-hit" {
		t.Fatalf("unexpected events: %+v", events)
	}

	findings, err := s.ListFindings(store.FindingFilter{Tool: "legacy-cartero"})
	if err != nil {
		t.Fatalf("ListFindings() error = %v", err)
	}
	if len(findings) != 1 || findings[0].Metadata["password_present"] != "true" {
		t.Fatalf("unexpected findings: %+v", findings)
	}
}

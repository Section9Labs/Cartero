package web

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Section9Labs/Cartero/internal/store"
)

func TestServerDashboardAndTestingRuntime(t *testing.T) {
	root := t.TempDir()
	s, err := store.Open(root)
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})

	if _, err := s.SeedTemplates("template-library", []store.Template{{
		Slug:        "payroll-review",
		Name:        "Payroll Review",
		Locale:      "en-US",
		Department:  "Finance",
		Scenario:    "approval-bypass",
		Channels:    []string{"email"},
		Subject:     "Review payroll changes",
		LandingPage: "Review the payroll change request in this safe training route.",
	}}); err != nil {
		t.Fatalf("SeedTemplates() error = %v", err)
	}
	if _, err := s.UpsertAudienceMembers("finance-emea", "audience-sync", []store.AudienceMember{{
		Email:       "analyst@example.com",
		DisplayName: "Finance Analyst",
		Department:  "Finance",
		Title:       "Analyst",
	}}); err != nil {
		t.Fatalf("UpsertAudienceMembers() error = %v", err)
	}
	if _, err := s.SaveCampaignSnapshot(store.CampaignSnapshot{
		Name:       "Quarterly Payroll Drill",
		SourcePath: "/tmp/payroll.yaml",
		Audience:   "finance-emea",
		Region:     "EMEA",
		RiskLevel:  "medium",
		Readiness:  92,
		Source:     "preview",
	}); err != nil {
		t.Fatalf("SaveCampaignSnapshot() error = %v", err)
	}
	if _, err := s.SaveImportedMessage(store.ImportedMessage{
		SourcePath:        "/tmp/reported.eml",
		Sender:            "alerts@example.com",
		Subject:           "Payroll alert",
		Body:              "Review the submitted payroll request.",
		GeneratedCampaign: "metadata:\n  name: payroll\n",
	}); err != nil {
		t.Fatalf("SaveImportedMessage() error = %v", err)
	}

	app, err := New(root, s)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	server := httptest.NewServer(app.Handler())
	t.Cleanup(server.Close)

	resp, err := http.Get(server.URL + "/")
	if err != nil {
		t.Fatalf("GET / error = %v", err)
	}
	body := readBody(t, resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET / status = %d body=%s", resp.StatusCode, body)
	}
	if !strings.Contains(body, "Operations Board") || !strings.Contains(body, "Payroll Review") {
		t.Fatalf("unexpected dashboard body: %s", body)
	}

	resp, err = http.Get(server.URL + "/testing/payroll-review?campaign=Quarterly+Payroll+Drill&email=analyst@example.com")
	if err != nil {
		t.Fatalf("GET /testing error = %v", err)
	}
	body = readBody(t, resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /testing status = %d body=%s", resp.StatusCode, body)
	}
	if !strings.Contains(body, "No submitted values are retained") || !strings.Contains(body, "Interaction probe") {
		t.Fatalf("unexpected testing page body: %s", body)
	}

	payload, err := json.Marshal(testingEventPayload{
		Slug:     "payroll-review",
		Campaign: "Quarterly Payroll Drill",
		Email:    "analyst@example.com",
		Type:     "submit-attempt",
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	resp, err = http.Post(server.URL+"/api/testing/event", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("POST /api/testing/event error = %v", err)
	}
	readBody(t, resp.Body)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("POST /api/testing/event status = %d", resp.StatusCode)
	}

	events, err := s.ListEvents()
	if err != nil {
		t.Fatalf("ListEvents() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %+v", events)
	}
	if events[0].Type != "submit-attempt" || events[0].Source != "testing-runtime/payroll-review" {
		t.Fatalf("unexpected event: %+v", events[0])
	}
}

func readBody(t *testing.T, body io.ReadCloser) string {
	t.Helper()
	defer body.Close()

	payload, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	return string(payload)
}

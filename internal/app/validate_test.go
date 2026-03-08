package app_test

import (
	"path/filepath"
	"testing"

	"github.com/Section9Labs/Cartero/internal/app"
)

func TestValidateCampaignValidFixture(t *testing.T) {
	t.Parallel()

	campaign, err := app.LoadCampaign(filepath.Join("testdata", "valid.yaml"))
	if err != nil {
		t.Fatalf("LoadCampaign() error = %v", err)
	}

	issues := app.ValidateCampaign(campaign)
	if app.HasErrors(issues) {
		t.Fatalf("expected no validation errors, got %+v", issues)
	}
	if score := app.ReadinessScore(issues); score != 100 {
		t.Fatalf("expected readiness score 100, got %d", score)
	}
}

func TestValidateCampaignInvalidFixture(t *testing.T) {
	t.Parallel()

	campaign, err := app.LoadCampaign(filepath.Join("testdata", "invalid.yaml"))
	if err != nil {
		t.Fatalf("LoadCampaign() error = %v", err)
	}

	issues := app.ValidateCampaign(campaign)
	if !app.HasErrors(issues) {
		t.Fatalf("expected validation errors, got %+v", issues)
	}
	if score := app.ReadinessScore(issues); score >= 100 {
		t.Fatalf("expected degraded readiness score, got %d", score)
	}
}

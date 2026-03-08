package app_test

import (
	"path/filepath"
	"testing"

	"github.com/Section9Labs/Cartero/internal/app"
)

func TestLoadCampaign(t *testing.T) {
	t.Parallel()

	path := filepath.Join("testdata", "valid.yaml")
	campaign, err := app.LoadCampaign(path)
	if err != nil {
		t.Fatalf("LoadCampaign() error = %v", err)
	}

	if campaign.Metadata.Name != "Quarterly rehearsal" {
		t.Fatalf("unexpected campaign name: %s", campaign.Metadata.Name)
	}
	if len(campaign.Channels) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(campaign.Channels))
	}
}

package plugin

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateManifestRequiresSchemaCapabilitiesAndTrust(t *testing.T) {
	err := ValidateManifest(Manifest{
		Name:    "legacy-plugin",
		Version: "1.0.0",
		Kind:    "renderer",
		Mode:    "local-only",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}

	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	assertIssueFields(t, validationErr, "schema_version", "safe", "capabilities", "trust.level", "trust.review_required")
}

func TestValidateManifestAllowsReviewedOperatorReviewPlugin(t *testing.T) {
	safe := false
	reviewRequired := true

	err := ValidateManifest(Manifest{
		SchemaVersion: SchemaVersionV1,
		Name:          "clone-importer",
		Version:       "1.2.0",
		Kind:          "importer",
		Mode:          "operator-review",
		Safe:          &safe,
		Capabilities:  []string{"campaign.import"},
		Trust: Trust{
			Level:          "reviewed",
			ReviewRequired: &reviewRequired,
		},
	})
	if err != nil {
		t.Fatalf("ValidateManifest() error = %v", err)
	}
}

func TestLoadManifestFixtures(t *testing.T) {
	tests := []struct {
		name             string
		path             string
		wantIssues       []string
		wantCapabilities []string
	}{
		{
			name:             "valid v1",
			path:             filepath.Join("testdata", "fixtures", "valid-v1.yaml"),
			wantCapabilities: []string{"preview.render"},
		},
		{
			name:       "legacy manifest",
			path:       filepath.Join("testdata", "fixtures", "legacy-v0.yaml"),
			wantIssues: []string{"schema_version: is required", "capabilities: must declare at least one capability", "trust.level: is required", "trust.review_required: is required"},
		},
		{
			name:       "future schema",
			path:       filepath.Join("testdata", "fixtures", "future-v2.yaml"),
			wantIssues: []string{"schema_version: unsupported schema version \"v2\""},
		},
		{
			name:       "invalid capabilities",
			path:       filepath.Join("testdata", "fixtures", "invalid-capabilities.yaml"),
			wantIssues: []string{"capabilities[1]: unsupported capability \"preview.rendered\"", "capabilities[2]: duplicates capability \"results.export\""},
		},
		{
			name:       "invalid trust",
			path:       filepath.Join("testdata", "fixtures", "invalid-trust.yaml"),
			wantIssues: []string{"safe: must be false when mode is external-service", "trust.review_required: must be true when mode is external-service", "trust.review_required: must be true when trust.level is unreviewed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest, err := loadManifest(tt.path)
			if len(tt.wantIssues) == 0 {
				if err != nil {
					t.Fatalf("loadManifest() error = %v", err)
				}
				if strings.Join(manifest.Capabilities, ",") != strings.Join(tt.wantCapabilities, ",") {
					t.Fatalf("unexpected capabilities: %+v", manifest.Capabilities)
				}
				return
			}

			if err == nil {
				t.Fatal("expected validation error")
			}
			for _, issue := range tt.wantIssues {
				if !strings.Contains(err.Error(), issue) {
					t.Fatalf("expected error to contain %q, got %v", issue, err)
				}
			}
		})
	}
}

func assertIssueFields(t *testing.T, err *ValidationError, fields ...string) {
	t.Helper()

	for _, field := range fields {
		found := false
		for _, issue := range err.Issues {
			if issue.Field == field {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected issue for field %q, got %+v", field, err.Issues)
		}
	}
}

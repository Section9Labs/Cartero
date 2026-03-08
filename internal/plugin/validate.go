package plugin

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const SchemaVersionV1 = "v1"

var (
	namePattern    = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	versionPattern = regexp.MustCompile(`^v?\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$`)

	validKinds = map[string][]string{
		"renderer":     {"preview.render"},
		"content-pack": {"campaign.template"},
		"importer":     {"campaign.import"},
		"exporter":     {"results.export"},
		"integration":  {"audience.sync", "events.ingest", "webhook.deliver"},
	}
	validModes = map[string]struct{}{
		"local-only":       {},
		"operator-review":  {},
		"external-service": {},
	}
	validTrustLevels = map[string]struct{}{
		"first-party": {},
		"reviewed":    {},
		"unreviewed":  {},
	}
	validCapabilities = map[string]struct{}{
		"preview.render":    {},
		"campaign.template": {},
		"campaign.import":   {},
		"audience.sync":     {},
		"results.export":    {},
		"events.ingest":     {},
		"webhook.deliver":   {},
	}
)

type Issue struct {
	Field   string
	Message string
}

type ValidationError struct {
	Issues []Issue
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Issues))
	for _, issue := range e.Issues {
		parts = append(parts, fmt.Sprintf("%s: %s", issue.Field, issue.Message))
	}

	return strings.Join(parts, "; ")
}

func ValidateManifest(manifest Manifest) error {
	var issues []Issue

	if strings.TrimSpace(manifest.SchemaVersion) == "" {
		issues = append(issues, Issue{Field: "schema_version", Message: "is required"})
	} else if manifest.SchemaVersion != SchemaVersionV1 {
		issues = append(issues, Issue{
			Field:   "schema_version",
			Message: fmt.Sprintf("unsupported schema version %q", manifest.SchemaVersion),
		})
	}

	if strings.TrimSpace(manifest.Name) == "" {
		issues = append(issues, Issue{Field: "name", Message: "is required"})
	} else if !namePattern.MatchString(manifest.Name) {
		issues = append(issues, Issue{Field: "name", Message: "must use lowercase letters, digits, and hyphens"})
	}

	if strings.TrimSpace(manifest.Version) == "" {
		issues = append(issues, Issue{Field: "version", Message: "is required"})
	} else if !versionPattern.MatchString(manifest.Version) {
		issues = append(issues, Issue{Field: "version", Message: "must be a semantic version like 1.2.3"})
	}

	kindAllowedCapabilities, kindValid := validKinds[manifest.Kind]
	if strings.TrimSpace(manifest.Kind) == "" {
		issues = append(issues, Issue{Field: "kind", Message: "is required"})
	} else if !kindValid {
		issues = append(issues, Issue{
			Field:   "kind",
			Message: fmt.Sprintf("must be one of %s", joinKeys(validKinds)),
		})
	}

	if strings.TrimSpace(manifest.Mode) == "" {
		issues = append(issues, Issue{Field: "mode", Message: "is required"})
	} else if _, ok := validModes[manifest.Mode]; !ok {
		issues = append(issues, Issue{
			Field:   "mode",
			Message: fmt.Sprintf("must be one of %s", joinKeys(validModes)),
		})
	}

	if manifest.Safe == nil {
		issues = append(issues, Issue{Field: "safe", Message: "is required"})
	}

	if len(manifest.Capabilities) == 0 {
		issues = append(issues, Issue{Field: "capabilities", Message: "must declare at least one capability"})
	} else {
		seen := make(map[string]struct{}, len(manifest.Capabilities))
		kindCapabilitySet := make(map[string]struct{}, len(kindAllowedCapabilities))
		for _, capability := range kindAllowedCapabilities {
			kindCapabilitySet[capability] = struct{}{}
		}

		kindCompatible := false
		for i, capability := range manifest.Capabilities {
			field := fmt.Sprintf("capabilities[%d]", i)
			capability = strings.TrimSpace(capability)

			if capability == "" {
				issues = append(issues, Issue{Field: field, Message: "must not be empty"})
				continue
			}
			if _, ok := validCapabilities[capability]; !ok {
				issues = append(issues, Issue{
					Field:   field,
					Message: fmt.Sprintf("unsupported capability %q", capability),
				})
				continue
			}
			if _, ok := seen[capability]; ok {
				issues = append(issues, Issue{
					Field:   field,
					Message: fmt.Sprintf("duplicates capability %q", capability),
				})
				continue
			}

			seen[capability] = struct{}{}
			if _, ok := kindCapabilitySet[capability]; ok {
				kindCompatible = true
			}
		}

		if kindValid && !kindCompatible {
			issues = append(issues, Issue{
				Field:   "capabilities",
				Message: fmt.Sprintf("must include at least one capability compatible with kind %q: %s", manifest.Kind, strings.Join(kindAllowedCapabilities, ", ")),
			})
		}
	}

	if strings.TrimSpace(manifest.Trust.Level) == "" {
		issues = append(issues, Issue{Field: "trust.level", Message: "is required"})
	} else if _, ok := validTrustLevels[manifest.Trust.Level]; !ok {
		issues = append(issues, Issue{
			Field:   "trust.level",
			Message: fmt.Sprintf("must be one of %s", joinKeys(validTrustLevels)),
		})
	}

	if manifest.Trust.ReviewRequired == nil {
		issues = append(issues, Issue{Field: "trust.review_required", Message: "is required"})
	}

	if manifest.Safe != nil && *manifest.Safe {
		if manifest.Mode == "external-service" {
			issues = append(issues, Issue{
				Field:   "safe",
				Message: "must be false when mode is external-service",
			})
		}
	}
	if manifest.Safe != nil && !*manifest.Safe && manifest.Trust.ReviewRequired != nil && !*manifest.Trust.ReviewRequired {
		issues = append(issues, Issue{
			Field:   "trust.review_required",
			Message: "must be true when safe is false",
		})
	}
	if manifest.Mode == "external-service" && manifest.Trust.ReviewRequired != nil && !*manifest.Trust.ReviewRequired {
		issues = append(issues, Issue{
			Field:   "trust.review_required",
			Message: "must be true when mode is external-service",
		})
	}
	if manifest.Trust.Level == "unreviewed" && manifest.Trust.ReviewRequired != nil && !*manifest.Trust.ReviewRequired {
		issues = append(issues, Issue{
			Field:   "trust.review_required",
			Message: "must be true when trust.level is unreviewed",
		})
	}

	if len(issues) == 0 {
		return nil
	}

	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].Field == issues[j].Field {
			return issues[i].Message < issues[j].Message
		}
		return issues[i].Field < issues[j].Field
	})

	return &ValidationError{Issues: issues}
}

func joinKeys[K comparable, V any](values map[K]V) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, fmt.Sprint(key))
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}

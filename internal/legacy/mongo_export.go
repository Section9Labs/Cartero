package legacy

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Section9Labs/Cartero/internal/store"
)

type MongoImportReport struct {
	FilesProcessed      int
	AudienceCreated     int
	AudienceUpdated     int
	EventsImported      int
	FindingsCreated     int
	FindingsUpdated     int
	RedactedCredentials int
}

type MongoImportOptions struct {
	Path    string
	Segment string
}

func ImportMongoExport(s *store.Store, opts MongoImportOptions) (MongoImportReport, error) {
	if strings.TrimSpace(opts.Path) == "" {
		return MongoImportReport{}, errors.New("mongo export path is required")
	}
	if strings.TrimSpace(opts.Segment) == "" {
		opts.Segment = "legacy-import"
	}

	info, err := os.Stat(opts.Path)
	if err != nil {
		return MongoImportReport{}, fmt.Errorf("stat mongo export path: %w", err)
	}

	report := MongoImportReport{}
	if info.IsDir() {
		for _, candidate := range []struct {
			name string
			kind string
		}{
			{name: "people.json", kind: "people"},
			{name: "persons.json", kind: "people"},
			{name: "hits.json", kind: "hits"},
			{name: "credentials.json", kind: "credentials"},
		} {
			path := filepath.Join(opts.Path, candidate.name)
			if _, err := os.Stat(path); err != nil {
				continue
			}
			if err := importMongoFile(s, path, candidate.kind, opts.Segment, &report); err != nil {
				return report, err
			}
			report.FilesProcessed++
		}
		if report.FilesProcessed == 0 {
			return report, errors.New("no supported mongo export files found")
		}
		return report, nil
	}

	kind := inferMongoKind(filepath.Base(opts.Path))
	if kind == "" {
		return report, errors.New("unable to infer mongo export kind from filename")
	}
	if err := importMongoFile(s, opts.Path, kind, opts.Segment, &report); err != nil {
		return report, err
	}
	report.FilesProcessed = 1

	return report, nil
}

func importMongoFile(s *store.Store, path, kind, segment string, report *MongoImportReport) error {
	payload, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read mongo export %s: %w", path, err)
	}

	switch kind {
	case "people":
		var people []struct {
			Email     string   `json:"email"`
			Campaigns []string `json:"campaigns"`
			Responded []string `json:"responded"`
			CreatedAt string   `json:"created_at"`
		}
		if err := json.Unmarshal(payload, &people); err != nil {
			return fmt.Errorf("parse people export: %w", err)
		}

		members := make([]store.AudienceMember, 0, len(people))
		for _, person := range people {
			members = append(members, store.AudienceMember{
				Email:     person.Email,
				Segment:   segment,
				Source:    "mongo-export",
				CreatedAt: person.CreatedAt,
			})
		}
		result, err := s.UpsertAudienceMembers(segment, "mongo-export", members)
		if err != nil {
			return err
		}
		report.AudienceCreated += result.Created
		report.AudienceUpdated += result.Updated
		return nil
	case "hits":
		var hits []struct {
			Domain    string `json:"domain"`
			Path      string `json:"path"`
			IP        string `json:"ip"`
			CreatedAt string `json:"created_at"`
		}
		if err := json.Unmarshal(payload, &hits); err != nil {
			return fmt.Errorf("parse hits export: %w", err)
		}

		for _, hit := range hits {
			if _, err := s.SaveEvent(store.Event{
				CampaignName:  nonEmpty(hit.Domain, "legacy-hit"),
				AudienceEmail: "",
				Type:          "legacy-hit",
				Source:        buildLegacySource(hit.Domain, hit.Path, hit.IP),
				CreatedAt:     hit.CreatedAt,
			}); err != nil {
				return err
			}
			report.EventsImported++
		}
		return nil
	case "credentials":
		var credentials []struct {
			Domain    string         `json:"domain"`
			Path      string         `json:"path"`
			IP        string         `json:"ip"`
			Username  string         `json:"username"`
			Password  string         `json:"password"`
			Data      map[string]any `json:"data"`
			CreatedAt string         `json:"created_at"`
		}
		if err := json.Unmarshal(payload, &credentials); err != nil {
			return fmt.Errorf("parse credentials export: %w", err)
		}

		findings := make([]store.Finding, 0, len(credentials))
		for _, credential := range credentials {
			findings = append(findings, store.Finding{
				Source:   "mongo-export",
				Tool:     "legacy-cartero",
				Rule:     "legacy-credential-artifact",
				Severity: "high",
				Target:   buildLegacyTarget(credential.Domain, credential.Path, credential.IP),
				Summary:  "Legacy credential artifact detected during migration; submitted values were intentionally redacted.",
				Metadata: map[string]string{
					"username_present": strconvBool(credential.Username != ""),
					"password_present": strconvBool(credential.Password != ""),
					"field_count":      fmt.Sprintf("%d", len(credential.Data)),
				},
				CreatedAt: credential.CreatedAt,
			})
		}
		result, err := s.ImportFindings("mongo-export", findings)
		if err != nil {
			return err
		}
		report.FindingsCreated += result.Created
		report.FindingsUpdated += result.Updated
		report.RedactedCredentials += len(credentials)
		return nil
	default:
		return fmt.Errorf("unsupported mongo export kind %q", kind)
	}
}

func inferMongoKind(name string) string {
	name = strings.ToLower(name)
	switch {
	case strings.Contains(name, "people"), strings.Contains(name, "person"):
		return "people"
	case strings.Contains(name, "hit"):
		return "hits"
	case strings.Contains(name, "credential"):
		return "credentials"
	default:
		return ""
	}
}

func buildLegacySource(domain, path, ip string) string {
	parts := []string{"mongo-export"}
	if domain != "" {
		parts = append(parts, domain)
	}
	if path != "" {
		parts = append(parts, path)
	}
	if ip != "" {
		parts = append(parts, ip)
	}
	return strings.Join(parts, "|")
}

func buildLegacyTarget(domain, path, ip string) string {
	return nonEmpty(strings.TrimSpace(domain)+strings.TrimSpace(path), ip, "legacy-target")
}

func nonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func strconvBool(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

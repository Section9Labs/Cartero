package reporting

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/Section9Labs/Cartero/internal/store"
)

type Export struct {
	GeneratedAt string                   `json:"generated_at"`
	Database    string                   `json:"database"`
	Stats       store.WorkspaceStats     `json:"stats"`
	Segments    []store.SegmentSummary   `json:"segments"`
	Events      []store.EventSummary     `json:"events"`
	Campaigns   []store.CampaignSnapshot `json:"campaigns"`
	Imports     []store.ImportedMessage  `json:"imports"`
	Findings    []store.Finding          `json:"findings"`
}

func Build(s *store.Store) (Export, error) {
	stats, err := s.Stats()
	if err != nil {
		return Export{}, err
	}
	segments, err := s.SegmentSummaries()
	if err != nil {
		return Export{}, err
	}
	events, err := s.EventSummaries()
	if err != nil {
		return Export{}, err
	}
	campaigns, err := s.ListCampaignSnapshots()
	if err != nil {
		return Export{}, err
	}
	imports, err := s.ListImportedMessages()
	if err != nil {
		return Export{}, err
	}
	findings, err := s.ListFindings(store.FindingFilter{})
	if err != nil {
		return Export{}, err
	}

	return Export{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Database:    s.Path(),
		Stats:       stats,
		Segments:    segments,
		Events:      events,
		Campaigns:   campaigns,
		Imports:     imports,
		Findings:    findings,
	}, nil
}

func Write(path, format string, export Export) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create export directory: %w", err)
	}

	switch format {
	case "json":
		payload, err := json.MarshalIndent(export, "", "  ")
		if err != nil {
			return fmt.Errorf("encode json export: %w", err)
		}
		if err := os.WriteFile(path, append(payload, '\n'), 0o644); err != nil {
			return fmt.Errorf("write json export: %w", err)
		}
		return nil
	case "csv":
		file, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("create csv export: %w", err)
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		rows := [][]string{
			{"kind", "name", "value", "detail"},
			{"metric", "templates", strconv.Itoa(export.Stats.TemplateCount), ""},
			{"metric", "audience_members", strconv.Itoa(export.Stats.AudienceCount), ""},
			{"metric", "segments", strconv.Itoa(export.Stats.SegmentCount), ""},
			{"metric", "imports", strconv.Itoa(export.Stats.ImportCount), ""},
			{"metric", "campaigns", strconv.Itoa(export.Stats.CampaignCount), ""},
			{"metric", "events", strconv.Itoa(export.Stats.EventCount), ""},
			{"metric", "findings", strconv.Itoa(export.Stats.FindingCount), ""},
		}
		for _, segment := range export.Segments {
			rows = append(rows, []string{"segment", segment.Segment, strconv.Itoa(segment.Members), ""})
		}
		for _, event := range export.Events {
			rows = append(rows, []string{"event", event.Type, strconv.Itoa(event.Count), ""})
		}
		for _, campaign := range export.Campaigns {
			rows = append(rows, []string{"campaign", campaign.Name, strconv.Itoa(campaign.Readiness), campaign.SourcePath})
		}
		for _, message := range export.Imports {
			rows = append(rows, []string{"import", message.Subject, message.Sender, message.SourcePath})
		}
		for _, finding := range export.Findings {
			rows = append(rows, []string{"finding", finding.Rule, finding.Severity, finding.Target})
		}

		if err := writer.WriteAll(rows); err != nil {
			return fmt.Errorf("write csv export: %w", err)
		}
		writer.Flush()
		return writer.Error()
	default:
		return fmt.Errorf("unsupported export format %q", format)
	}
}

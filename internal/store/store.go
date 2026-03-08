package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	bucketMeta      = []byte("meta")
	bucketTemplates = []byte("templates")
	bucketAudiences = []byte("audiences")
	bucketImports   = []byte("imports")
	bucketCampaigns = []byte("campaigns")
	bucketEvents    = []byte("events")
)

const schemaVersion = "1"

type Store struct {
	db   *bolt.DB
	path string
}

type Template struct {
	ID          uint64   `json:"id"`
	Plugin      string   `json:"plugin"`
	Slug        string   `json:"slug"`
	Name        string   `json:"name"`
	Locale      string   `json:"locale"`
	Department  string   `json:"department"`
	Scenario    string   `json:"scenario"`
	Channels    []string `json:"channels"`
	Subject     string   `json:"subject"`
	Body        string   `json:"body"`
	LandingPage string   `json:"landing_page"`
	CreatedAt   string   `json:"created_at"`
}

type TemplateFilter struct {
	Locale     string
	Department string
	Scenario   string
}

type AudienceMember struct {
	ID          uint64 `json:"id"`
	Segment     string `json:"segment"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Department  string `json:"department"`
	Title       string `json:"title"`
	Source      string `json:"source"`
	CreatedAt   string `json:"created_at"`
}

type AudienceFilter struct {
	Segment string
}

type ImportedMessage struct {
	ID                uint64 `json:"id"`
	Plugin            string `json:"plugin"`
	SourcePath        string `json:"source_path"`
	Sender            string `json:"sender"`
	Subject           string `json:"subject"`
	Body              string `json:"body"`
	GeneratedCampaign string `json:"generated_campaign"`
	CreatedAt         string `json:"created_at"`
}

type CampaignSnapshot struct {
	ID            uint64   `json:"id"`
	Name          string   `json:"name"`
	SourcePath    string   `json:"source_path"`
	Owner         string   `json:"owner"`
	Audience      string   `json:"audience"`
	Region        string   `json:"region"`
	RiskLevel     string   `json:"risk_level"`
	Readiness     int      `json:"readiness"`
	IssueCount    int      `json:"issue_count"`
	Issues        []string `json:"issues"`
	Source        string   `json:"source"`
	CreatedAt     string   `json:"created_at"`
	LastValidated string   `json:"last_validated"`
}

type Event struct {
	ID            uint64 `json:"id"`
	CampaignName  string `json:"campaign_name"`
	AudienceEmail string `json:"audience_email"`
	Type          string `json:"type"`
	Source        string `json:"source"`
	CreatedAt     string `json:"created_at"`
}

type SeedReport struct {
	Created int
	Updated int
}

type AudienceImportResult struct {
	Segment string
	Created int
	Updated int
}

type WorkspaceStats struct {
	DatabasePath  string
	TemplateCount int
	AudienceCount int
	SegmentCount  int
	ImportCount   int
	CampaignCount int
	EventCount    int
}

type SegmentSummary struct {
	Segment string `json:"segment"`
	Members int    `json:"members"`
}

type EventSummary struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

func DatabasePath(root string) string {
	return filepath.Join(root, ".cartero", "cartero.db")
}

func EnsureWorkspace(root string) error {
	for _, dir := range []string{
		filepath.Join(root, ".cartero"),
		filepath.Join(root, "plugins"),
		filepath.Join(root, "exports"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("ensure workspace directory %s: %w", dir, err)
		}
	}

	return nil
}

func Open(root string) (*Store, error) {
	if err := EnsureWorkspace(root); err != nil {
		return nil, err
	}

	path := DatabasePath(root)
	db, err := bolt.Open(path, 0o600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, fmt.Errorf("open workspace database: %w", err)
	}

	store := &Store{db: db, path: path}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) migrate() error {
	return s.db.Update(func(tx *bolt.Tx) error {
		for _, bucket := range [][]byte{
			bucketMeta,
			bucketTemplates,
			bucketAudiences,
			bucketImports,
			bucketCampaigns,
			bucketEvents,
		} {
			if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
				return fmt.Errorf("create bucket %s: %w", bucket, err)
			}
		}

		meta := tx.Bucket(bucketMeta)
		if err := meta.Put([]byte("schema_version"), []byte(schemaVersion)); err != nil {
			return fmt.Errorf("write schema version: %w", err)
		}

		return nil
	})
}

func (s *Store) Stats() (WorkspaceStats, error) {
	var stats WorkspaceStats
	stats.DatabasePath = s.path

	err := s.db.View(func(tx *bolt.Tx) error {
		stats.TemplateCount = bucketLen(tx.Bucket(bucketTemplates))
		stats.AudienceCount = bucketLen(tx.Bucket(bucketAudiences))
		stats.ImportCount = bucketLen(tx.Bucket(bucketImports))
		stats.CampaignCount = bucketLen(tx.Bucket(bucketCampaigns))
		stats.EventCount = bucketLen(tx.Bucket(bucketEvents))

		segments := make(map[string]struct{})
		if err := tx.Bucket(bucketAudiences).ForEach(func(_, value []byte) error {
			var member AudienceMember
			if err := json.Unmarshal(value, &member); err != nil {
				return err
			}
			segments[member.Segment] = struct{}{}
			return nil
		}); err != nil {
			return err
		}
		stats.SegmentCount = len(segments)

		return nil
	})
	if err != nil {
		return stats, fmt.Errorf("read workspace stats: %w", err)
	}

	return stats, nil
}

func (s *Store) SeedTemplates(pluginName string, templates []Template) (SeedReport, error) {
	var report SeedReport

	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketTemplates)
		for _, template := range templates {
			key := []byte(strings.TrimSpace(template.Slug))
			if len(key) == 0 {
				return errors.New("template slug is required")
			}

			now := nowRFC3339()
			template.Plugin = emptyDefault(template.Plugin, pluginName)
			template.CreatedAt = emptyDefault(template.CreatedAt, now)

			if raw := bucket.Get(key); raw != nil {
				var existing Template
				if err := json.Unmarshal(raw, &existing); err != nil {
					return err
				}
				template.ID = existing.ID
				template.CreatedAt = existing.CreatedAt
				report.Updated++
			} else {
				id, err := bucket.NextSequence()
				if err != nil {
					return err
				}
				template.ID = id
				report.Created++
			}

			encoded, err := json.Marshal(template)
			if err != nil {
				return err
			}
			if err := bucket.Put(key, encoded); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return report, fmt.Errorf("seed templates: %w", err)
	}

	return report, nil
}

func (s *Store) ListTemplates(filter TemplateFilter) ([]Template, error) {
	var templates []Template

	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketTemplates).ForEach(func(_, value []byte) error {
			var template Template
			if err := json.Unmarshal(value, &template); err != nil {
				return err
			}
			if !matchesTemplateFilter(template, filter) {
				return nil
			}
			templates = append(templates, template)
			return nil
		})
	})
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}

	slices.SortFunc(templates, func(a, b Template) int {
		if a.Locale == b.Locale {
			return strings.Compare(a.Slug, b.Slug)
		}
		return strings.Compare(a.Locale, b.Locale)
	})

	return templates, nil
}

func (s *Store) UpsertAudienceMembers(segment, source string, members []AudienceMember) (AudienceImportResult, error) {
	var result AudienceImportResult
	result.Segment = segment

	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketAudiences)
		for _, member := range members {
			email := strings.ToLower(strings.TrimSpace(member.Email))
			if email == "" {
				return errors.New("audience email is required")
			}

			now := nowRFC3339()
			member.Email = email
			member.Segment = segment
			member.Source = emptyDefault(member.Source, source)
			member.DisplayName = emptyDefault(member.DisplayName, email)
			member.CreatedAt = emptyDefault(member.CreatedAt, now)

			key := []byte(segment + "\x00" + email)
			if raw := bucket.Get(key); raw != nil {
				var existing AudienceMember
				if err := json.Unmarshal(raw, &existing); err != nil {
					return err
				}
				member.ID = existing.ID
				member.CreatedAt = existing.CreatedAt
				result.Updated++
			} else {
				id, err := bucket.NextSequence()
				if err != nil {
					return err
				}
				member.ID = id
				result.Created++
			}

			encoded, err := json.Marshal(member)
			if err != nil {
				return err
			}
			if err := bucket.Put(key, encoded); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return result, fmt.Errorf("upsert audience members: %w", err)
	}

	return result, nil
}

func (s *Store) ListAudienceMembers(filter AudienceFilter) ([]AudienceMember, error) {
	var members []AudienceMember

	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketAudiences).ForEach(func(_, value []byte) error {
			var member AudienceMember
			if err := json.Unmarshal(value, &member); err != nil {
				return err
			}
			if filter.Segment != "" && !strings.EqualFold(member.Segment, filter.Segment) {
				return nil
			}
			members = append(members, member)
			return nil
		})
	})
	if err != nil {
		return nil, fmt.Errorf("list audience members: %w", err)
	}

	slices.SortFunc(members, func(a, b AudienceMember) int {
		if a.Segment == b.Segment {
			return strings.Compare(a.Email, b.Email)
		}
		return strings.Compare(a.Segment, b.Segment)
	})

	return members, nil
}

func (s *Store) SaveImportedMessage(message ImportedMessage) (ImportedMessage, error) {
	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketImports)
		key := []byte(strings.TrimSpace(message.SourcePath))
		if len(key) == 0 {
			return errors.New("import source path is required")
		}

		now := nowRFC3339()
		message.Plugin = emptyDefault(message.Plugin, "clone-importer")
		message.CreatedAt = emptyDefault(message.CreatedAt, now)

		if raw := bucket.Get(key); raw != nil {
			var existing ImportedMessage
			if err := json.Unmarshal(raw, &existing); err != nil {
				return err
			}
			message.ID = existing.ID
			message.CreatedAt = existing.CreatedAt
		} else {
			id, err := bucket.NextSequence()
			if err != nil {
				return err
			}
			message.ID = id
		}

		encoded, err := json.Marshal(message)
		if err != nil {
			return err
		}

		return bucket.Put(key, encoded)
	})
	if err != nil {
		return ImportedMessage{}, fmt.Errorf("save imported message: %w", err)
	}

	return message, nil
}

func (s *Store) ListImportedMessages() ([]ImportedMessage, error) {
	var imports []ImportedMessage

	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketImports).ForEach(func(_, value []byte) error {
			var message ImportedMessage
			if err := json.Unmarshal(value, &message); err != nil {
				return err
			}
			imports = append(imports, message)
			return nil
		})
	})
	if err != nil {
		return nil, fmt.Errorf("list imported messages: %w", err)
	}

	slices.SortFunc(imports, func(a, b ImportedMessage) int {
		return strings.Compare(a.SourcePath, b.SourcePath)
	})

	return imports, nil
}

func (s *Store) SaveCampaignSnapshot(snapshot CampaignSnapshot) (CampaignSnapshot, error) {
	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketCampaigns)
		key := []byte(strings.TrimSpace(snapshot.SourcePath))
		if len(key) == 0 {
			return errors.New("campaign snapshot source path is required")
		}

		now := nowRFC3339()
		snapshot.CreatedAt = emptyDefault(snapshot.CreatedAt, now)
		snapshot.LastValidated = now

		if raw := bucket.Get(key); raw != nil {
			var existing CampaignSnapshot
			if err := json.Unmarshal(raw, &existing); err != nil {
				return err
			}
			snapshot.ID = existing.ID
			snapshot.CreatedAt = existing.CreatedAt
		} else {
			id, err := bucket.NextSequence()
			if err != nil {
				return err
			}
			snapshot.ID = id
		}

		encoded, err := json.Marshal(snapshot)
		if err != nil {
			return err
		}

		return bucket.Put(key, encoded)
	})
	if err != nil {
		return CampaignSnapshot{}, fmt.Errorf("save campaign snapshot: %w", err)
	}

	return snapshot, nil
}

func (s *Store) ListCampaignSnapshots() ([]CampaignSnapshot, error) {
	var snapshots []CampaignSnapshot

	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketCampaigns).ForEach(func(_, value []byte) error {
			var snapshot CampaignSnapshot
			if err := json.Unmarshal(value, &snapshot); err != nil {
				return err
			}
			snapshots = append(snapshots, snapshot)
			return nil
		})
	})
	if err != nil {
		return nil, fmt.Errorf("list campaign snapshots: %w", err)
	}

	slices.SortFunc(snapshots, func(a, b CampaignSnapshot) int {
		return strings.Compare(a.Name, b.Name)
	})

	return snapshots, nil
}

func (s *Store) SaveEvent(event Event) (Event, error) {
	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketEvents)
		id, err := bucket.NextSequence()
		if err != nil {
			return err
		}

		event.ID = id
		event.Type = strings.ToLower(strings.TrimSpace(event.Type))
		event.AudienceEmail = strings.ToLower(strings.TrimSpace(event.AudienceEmail))
		event.CreatedAt = emptyDefault(event.CreatedAt, nowRFC3339())

		encoded, err := json.Marshal(event)
		if err != nil {
			return err
		}

		return bucket.Put(itob(id), encoded)
	})
	if err != nil {
		return Event{}, fmt.Errorf("save event: %w", err)
	}

	return event, nil
}

func (s *Store) ListEvents() ([]Event, error) {
	var events []Event

	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketEvents).ForEach(func(_, value []byte) error {
			var event Event
			if err := json.Unmarshal(value, &event); err != nil {
				return err
			}
			events = append(events, event)
			return nil
		})
	})
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}

	slices.SortFunc(events, func(a, b Event) int {
		if a.CampaignName == b.CampaignName {
			if a.Type == b.Type {
				return strings.Compare(a.AudienceEmail, b.AudienceEmail)
			}
			return strings.Compare(a.Type, b.Type)
		}
		return strings.Compare(a.CampaignName, b.CampaignName)
	})

	return events, nil
}

func (s *Store) SegmentSummaries() ([]SegmentSummary, error) {
	members, err := s.ListAudienceMembers(AudienceFilter{})
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int)
	for _, member := range members {
		counts[member.Segment]++
	}

	summaries := make([]SegmentSummary, 0, len(counts))
	for segment, count := range counts {
		summaries = append(summaries, SegmentSummary{Segment: segment, Members: count})
	}
	slices.SortFunc(summaries, func(a, b SegmentSummary) int {
		return strings.Compare(a.Segment, b.Segment)
	})

	return summaries, nil
}

func (s *Store) EventSummaries() ([]EventSummary, error) {
	events, err := s.ListEvents()
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int)
	for _, event := range events {
		counts[event.Type]++
	}

	summaries := make([]EventSummary, 0, len(counts))
	for eventType, count := range counts {
		summaries = append(summaries, EventSummary{Type: eventType, Count: count})
	}
	slices.SortFunc(summaries, func(a, b EventSummary) int {
		return strings.Compare(a.Type, b.Type)
	})

	return summaries, nil
}

func matchesTemplateFilter(template Template, filter TemplateFilter) bool {
	if filter.Locale != "" && !strings.EqualFold(template.Locale, filter.Locale) {
		return false
	}
	if filter.Department != "" && !strings.EqualFold(template.Department, filter.Department) {
		return false
	}
	if filter.Scenario != "" && !strings.EqualFold(template.Scenario, filter.Scenario) {
		return false
	}

	return true
}

func bucketLen(bucket *bolt.Bucket) int {
	if bucket == nil {
		return 0
	}

	count := 0
	_ = bucket.ForEach(func(_, _ []byte) error {
		count++
		return nil
	})

	return count
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func emptyDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	return value
}

func itob(v uint64) []byte {
	return []byte(fmt.Sprintf("%020d", v))
}

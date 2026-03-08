package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const schemaVersion = "3"

type Store struct {
	db   *sql.DB
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

type Finding struct {
	ID        uint64            `json:"id"`
	Source    string            `json:"source"`
	Tool      string            `json:"tool"`
	Rule      string            `json:"rule"`
	Severity  string            `json:"severity"`
	Target    string            `json:"target"`
	Summary   string            `json:"summary"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt string            `json:"created_at"`
}

type FindingFilter struct {
	Source   string
	Tool     string
	Severity string
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

type FindingImportResult struct {
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
	FindingCount  int
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
	return filepath.Join(root, ".cartero", "cartero.sqlite")
}

func LegacyDatabasePath(root string) string {
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
	needsLegacyImport := !pathExists(path) && pathExists(LegacyDatabasePath(root))

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open workspace database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	store := &Store{db: db, path: path}
	if err := store.configure(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if needsLegacyImport {
		if err := importLegacyBolt(LegacyDatabasePath(root), store); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("import legacy Bolt workspace: %w", err)
		}
	}

	return store, nil
}

func (s *Store) configure() error {
	for _, pragma := range []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA foreign_keys = ON",
	} {
		if _, err := s.db.Exec(pragma); err != nil {
			return fmt.Errorf("configure sqlite workspace: %w", err)
		}
	}

	return nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) migrate() error {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin workspace migration: %w", err)
	}
	defer tx.Rollback()

	for _, statement := range []string{
		`CREATE TABLE IF NOT EXISTS meta (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS templates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			plugin TEXT NOT NULL,
			slug TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			locale TEXT NOT NULL,
			department TEXT NOT NULL,
			scenario TEXT NOT NULL,
			channels_json TEXT NOT NULL,
			subject TEXT NOT NULL,
			body TEXT NOT NULL,
			landing_page TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS audiences (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			segment TEXT NOT NULL,
			email TEXT NOT NULL,
			display_name TEXT NOT NULL,
			department TEXT NOT NULL,
			title TEXT NOT NULL,
			source TEXT NOT NULL,
			created_at TEXT NOT NULL,
			UNIQUE(segment, email)
		)`,
		`CREATE TABLE IF NOT EXISTS imports (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			plugin TEXT NOT NULL,
			source_path TEXT NOT NULL UNIQUE,
			sender TEXT NOT NULL,
			subject TEXT NOT NULL,
			body TEXT NOT NULL,
			generated_campaign TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS campaigns (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			source_path TEXT NOT NULL UNIQUE,
			owner TEXT NOT NULL,
			audience TEXT NOT NULL,
			region TEXT NOT NULL,
			risk_level TEXT NOT NULL,
			readiness INTEGER NOT NULL,
			issue_count INTEGER NOT NULL,
			issues_json TEXT NOT NULL,
			source TEXT NOT NULL,
			created_at TEXT NOT NULL,
			last_validated TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			campaign_name TEXT NOT NULL,
			audience_email TEXT NOT NULL,
			type TEXT NOT NULL,
			source TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS findings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source TEXT NOT NULL,
			tool TEXT NOT NULL,
			rule_name TEXT NOT NULL,
			severity TEXT NOT NULL,
			target TEXT NOT NULL,
			summary TEXT NOT NULL,
			metadata_json TEXT NOT NULL,
			created_at TEXT NOT NULL,
			UNIQUE(source, tool, rule_name, target)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_templates_locale ON templates(locale)`,
		`CREATE INDEX IF NOT EXISTS idx_templates_department ON templates(department)`,
		`CREATE INDEX IF NOT EXISTS idx_templates_scenario ON templates(scenario)`,
		`CREATE INDEX IF NOT EXISTS idx_audiences_segment ON audiences(segment)`,
		`CREATE INDEX IF NOT EXISTS idx_imports_source_path ON imports(source_path)`,
		`CREATE INDEX IF NOT EXISTS idx_campaigns_source_path ON campaigns(source_path)`,
		`CREATE INDEX IF NOT EXISTS idx_events_campaign_name ON events(campaign_name)`,
		`CREATE INDEX IF NOT EXISTS idx_events_type ON events(type)`,
		`CREATE INDEX IF NOT EXISTS idx_findings_tool ON findings(tool)`,
		`CREATE INDEX IF NOT EXISTS idx_findings_severity ON findings(severity)`,
		`CREATE INDEX IF NOT EXISTS idx_findings_source ON findings(source)`,
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("apply workspace migration: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO meta (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		"schema_version",
		schemaVersion,
	); err != nil {
		return fmt.Errorf("write schema version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit workspace migration: %w", err)
	}

	return nil
}

func (s *Store) Stats() (WorkspaceStats, error) {
	stats := WorkspaceStats{DatabasePath: s.path}
	counts := []struct {
		query string
		dst   *int
	}{
		{query: `SELECT COUNT(*) FROM templates`, dst: &stats.TemplateCount},
		{query: `SELECT COUNT(*) FROM audiences`, dst: &stats.AudienceCount},
		{query: `SELECT COUNT(DISTINCT segment) FROM audiences`, dst: &stats.SegmentCount},
		{query: `SELECT COUNT(*) FROM imports`, dst: &stats.ImportCount},
		{query: `SELECT COUNT(*) FROM campaigns`, dst: &stats.CampaignCount},
		{query: `SELECT COUNT(*) FROM events`, dst: &stats.EventCount},
		{query: `SELECT COUNT(*) FROM findings`, dst: &stats.FindingCount},
	}

	for _, count := range counts {
		if err := s.db.QueryRow(count.query).Scan(count.dst); err != nil {
			return stats, fmt.Errorf("read workspace stats: %w", err)
		}
	}

	return stats, nil
}

func (s *Store) SeedTemplates(pluginName string, templates []Template) (SeedReport, error) {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return SeedReport{}, fmt.Errorf("begin template seed: %w", err)
	}
	defer tx.Rollback()

	var report SeedReport
	for _, template := range templates {
		created, err := upsertTemplateTx(tx, pluginName, &template)
		if err != nil {
			return report, fmt.Errorf("seed templates: %w", err)
		}
		if created {
			report.Created++
		} else {
			report.Updated++
		}
	}

	if err := tx.Commit(); err != nil {
		return report, fmt.Errorf("commit template seed: %w", err)
	}

	return report, nil
}

func (s *Store) ListTemplates(filter TemplateFilter) ([]Template, error) {
	rows, err := s.db.Query(`
		SELECT id, plugin, slug, name, locale, department, scenario, channels_json, subject, body, landing_page, created_at
		FROM templates
	`)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()

	var templates []Template
	for rows.Next() {
		var template Template
		var channelsJSON string
		if err := rows.Scan(
			&template.ID,
			&template.Plugin,
			&template.Slug,
			&template.Name,
			&template.Locale,
			&template.Department,
			&template.Scenario,
			&channelsJSON,
			&template.Subject,
			&template.Body,
			&template.LandingPage,
			&template.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("list templates: %w", err)
		}
		template.Channels = unmarshalStrings(channelsJSON)
		if !matchesTemplateFilter(template, filter) {
			continue
		}
		templates = append(templates, template)
	}
	if err := rows.Err(); err != nil {
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
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return AudienceImportResult{Segment: segment}, fmt.Errorf("begin audience import: %w", err)
	}
	defer tx.Rollback()

	result := AudienceImportResult{Segment: segment}
	for _, member := range members {
		created, err := upsertAudienceMemberTx(tx, segment, source, &member)
		if err != nil {
			return result, fmt.Errorf("upsert audience members: %w", err)
		}
		if created {
			result.Created++
		} else {
			result.Updated++
		}
	}

	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("commit audience import: %w", err)
	}

	return result, nil
}

func (s *Store) ListAudienceMembers(filter AudienceFilter) ([]AudienceMember, error) {
	query := `
		SELECT id, segment, email, display_name, department, title, source, created_at
		FROM audiences
	`
	args := []any{}
	if filter.Segment != "" {
		query += ` WHERE LOWER(segment) = LOWER(?)`
		args = append(args, filter.Segment)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list audience members: %w", err)
	}
	defer rows.Close()

	var members []AudienceMember
	for rows.Next() {
		var member AudienceMember
		if err := rows.Scan(
			&member.ID,
			&member.Segment,
			&member.Email,
			&member.DisplayName,
			&member.Department,
			&member.Title,
			&member.Source,
			&member.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("list audience members: %w", err)
		}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
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
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ImportedMessage{}, fmt.Errorf("begin import save: %w", err)
	}
	defer tx.Rollback()

	if _, err := upsertImportedMessageTx(tx, &message); err != nil {
		return ImportedMessage{}, fmt.Errorf("save imported message: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return ImportedMessage{}, fmt.Errorf("commit imported message: %w", err)
	}

	return message, nil
}

func (s *Store) ListImportedMessages() ([]ImportedMessage, error) {
	rows, err := s.db.Query(`
		SELECT id, plugin, source_path, sender, subject, body, generated_campaign, created_at
		FROM imports
	`)
	if err != nil {
		return nil, fmt.Errorf("list imported messages: %w", err)
	}
	defer rows.Close()

	var imports []ImportedMessage
	for rows.Next() {
		var message ImportedMessage
		if err := rows.Scan(
			&message.ID,
			&message.Plugin,
			&message.SourcePath,
			&message.Sender,
			&message.Subject,
			&message.Body,
			&message.GeneratedCampaign,
			&message.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("list imported messages: %w", err)
		}
		imports = append(imports, message)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list imported messages: %w", err)
	}

	slices.SortFunc(imports, func(a, b ImportedMessage) int {
		return strings.Compare(a.SourcePath, b.SourcePath)
	})

	return imports, nil
}

func (s *Store) SaveCampaignSnapshot(snapshot CampaignSnapshot) (CampaignSnapshot, error) {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return CampaignSnapshot{}, fmt.Errorf("begin campaign save: %w", err)
	}
	defer tx.Rollback()

	if _, err := upsertCampaignSnapshotTx(tx, &snapshot); err != nil {
		return CampaignSnapshot{}, fmt.Errorf("save campaign snapshot: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return CampaignSnapshot{}, fmt.Errorf("commit campaign snapshot: %w", err)
	}

	return snapshot, nil
}

func (s *Store) ListCampaignSnapshots() ([]CampaignSnapshot, error) {
	rows, err := s.db.Query(`
		SELECT id, name, source_path, owner, audience, region, risk_level, readiness, issue_count, issues_json, source, created_at, last_validated
		FROM campaigns
	`)
	if err != nil {
		return nil, fmt.Errorf("list campaign snapshots: %w", err)
	}
	defer rows.Close()

	var snapshots []CampaignSnapshot
	for rows.Next() {
		var snapshot CampaignSnapshot
		var issuesJSON string
		if err := rows.Scan(
			&snapshot.ID,
			&snapshot.Name,
			&snapshot.SourcePath,
			&snapshot.Owner,
			&snapshot.Audience,
			&snapshot.Region,
			&snapshot.RiskLevel,
			&snapshot.Readiness,
			&snapshot.IssueCount,
			&issuesJSON,
			&snapshot.Source,
			&snapshot.CreatedAt,
			&snapshot.LastValidated,
		); err != nil {
			return nil, fmt.Errorf("list campaign snapshots: %w", err)
		}
		snapshot.Issues = unmarshalStrings(issuesJSON)
		snapshots = append(snapshots, snapshot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list campaign snapshots: %w", err)
	}

	slices.SortFunc(snapshots, func(a, b CampaignSnapshot) int {
		return strings.Compare(a.Name, b.Name)
	})

	return snapshots, nil
}

func (s *Store) SaveEvent(event Event) (Event, error) {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Event{}, fmt.Errorf("begin event save: %w", err)
	}
	defer tx.Rollback()

	if err := insertEventTx(tx, &event); err != nil {
		return Event{}, fmt.Errorf("save event: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Event{}, fmt.Errorf("commit event: %w", err)
	}

	return event, nil
}

func (s *Store) ListEvents() ([]Event, error) {
	rows, err := s.db.Query(`
		SELECT id, campaign_name, audience_email, type, source, created_at
		FROM events
	`)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var event Event
		if err := rows.Scan(
			&event.ID,
			&event.CampaignName,
			&event.AudienceEmail,
			&event.Type,
			&event.Source,
			&event.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("list events: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
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
	rows, err := s.db.Query(`
		SELECT segment, COUNT(*)
		FROM audiences
		GROUP BY segment
		ORDER BY segment
	`)
	if err != nil {
		return nil, fmt.Errorf("segment summaries: %w", err)
	}
	defer rows.Close()

	var summaries []SegmentSummary
	for rows.Next() {
		var summary SegmentSummary
		if err := rows.Scan(&summary.Segment, &summary.Members); err != nil {
			return nil, fmt.Errorf("segment summaries: %w", err)
		}
		summaries = append(summaries, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("segment summaries: %w", err)
	}

	return summaries, nil
}

func (s *Store) EventSummaries() ([]EventSummary, error) {
	rows, err := s.db.Query(`
		SELECT type, COUNT(*)
		FROM events
		GROUP BY type
		ORDER BY type
	`)
	if err != nil {
		return nil, fmt.Errorf("event summaries: %w", err)
	}
	defer rows.Close()

	var summaries []EventSummary
	for rows.Next() {
		var summary EventSummary
		if err := rows.Scan(&summary.Type, &summary.Count); err != nil {
			return nil, fmt.Errorf("event summaries: %w", err)
		}
		summaries = append(summaries, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("event summaries: %w", err)
	}

	return summaries, nil
}

func (s *Store) ImportFindings(source string, findings []Finding) (FindingImportResult, error) {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FindingImportResult{}, fmt.Errorf("begin finding import: %w", err)
	}
	defer tx.Rollback()

	var result FindingImportResult
	for _, finding := range findings {
		created, err := upsertFindingTx(tx, source, &finding)
		if err != nil {
			return result, fmt.Errorf("import findings: %w", err)
		}
		if created {
			result.Created++
		} else {
			result.Updated++
		}
	}

	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("commit finding import: %w", err)
	}

	return result, nil
}

func (s *Store) ListFindings(filter FindingFilter) ([]Finding, error) {
	query := `
		SELECT id, source, tool, rule_name, severity, target, summary, metadata_json, created_at
		FROM findings
		WHERE 1 = 1
	`
	args := []any{}
	if filter.Source != "" {
		query += ` AND LOWER(source) = LOWER(?)`
		args = append(args, filter.Source)
	}
	if filter.Tool != "" {
		query += ` AND LOWER(tool) = LOWER(?)`
		args = append(args, filter.Tool)
	}
	if filter.Severity != "" {
		query += ` AND LOWER(severity) = LOWER(?)`
		args = append(args, filter.Severity)
	}
	query += ` ORDER BY severity DESC, tool ASC, target ASC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list findings: %w", err)
	}
	defer rows.Close()

	var findings []Finding
	for rows.Next() {
		var finding Finding
		var metadataJSON string
		if err := rows.Scan(
			&finding.ID,
			&finding.Source,
			&finding.Tool,
			&finding.Rule,
			&finding.Severity,
			&finding.Target,
			&finding.Summary,
			&metadataJSON,
			&finding.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("list findings: %w", err)
		}
		finding.Metadata = unmarshalStringMap(metadataJSON)
		findings = append(findings, finding)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list findings: %w", err)
	}

	return findings, nil
}

func upsertTemplateTx(tx *sql.Tx, pluginName string, template *Template) (bool, error) {
	slug := strings.TrimSpace(template.Slug)
	if slug == "" {
		return false, errors.New("template slug is required")
	}

	now := nowRFC3339()
	template.Slug = slug
	template.Plugin = emptyDefault(template.Plugin, pluginName)
	template.CreatedAt = emptyDefault(template.CreatedAt, now)
	channelsJSON, err := marshalStrings(template.Channels)
	if err != nil {
		return false, err
	}

	var existingID uint64
	var existingCreated string
	err = tx.QueryRow(`SELECT id, created_at FROM templates WHERE slug = ?`, template.Slug).Scan(&existingID, &existingCreated)
	switch {
	case err == nil:
		template.ID = existingID
		template.CreatedAt = existingCreated
		_, err = tx.Exec(`
			UPDATE templates
			SET plugin = ?, name = ?, locale = ?, department = ?, scenario = ?, channels_json = ?, subject = ?, body = ?, landing_page = ?, created_at = ?
			WHERE slug = ?
		`,
			template.Plugin,
			template.Name,
			template.Locale,
			template.Department,
			template.Scenario,
			channelsJSON,
			template.Subject,
			template.Body,
			template.LandingPage,
			template.CreatedAt,
			template.Slug,
		)
		return false, err
	case errors.Is(err, sql.ErrNoRows):
		if template.ID > 0 {
			_, err = tx.Exec(`
				INSERT INTO templates (id, plugin, slug, name, locale, department, scenario, channels_json, subject, body, landing_page, created_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`,
				template.ID,
				template.Plugin,
				template.Slug,
				template.Name,
				template.Locale,
				template.Department,
				template.Scenario,
				channelsJSON,
				template.Subject,
				template.Body,
				template.LandingPage,
				template.CreatedAt,
			)
			return true, err
		}

		result, err := tx.Exec(`
			INSERT INTO templates (plugin, slug, name, locale, department, scenario, channels_json, subject, body, landing_page, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			template.Plugin,
			template.Slug,
			template.Name,
			template.Locale,
			template.Department,
			template.Scenario,
			channelsJSON,
			template.Subject,
			template.Body,
			template.LandingPage,
			template.CreatedAt,
		)
		if err != nil {
			return false, err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return false, err
		}
		template.ID = uint64(id)
		return true, nil
	default:
		return false, err
	}
}

func upsertAudienceMemberTx(tx *sql.Tx, segment, source string, member *AudienceMember) (bool, error) {
	email := strings.ToLower(strings.TrimSpace(member.Email))
	if email == "" {
		return false, errors.New("audience email is required")
	}

	now := nowRFC3339()
	member.Email = email
	member.Segment = segment
	member.Source = emptyDefault(member.Source, source)
	member.DisplayName = emptyDefault(member.DisplayName, email)
	member.CreatedAt = emptyDefault(member.CreatedAt, now)

	var existingID uint64
	var existingCreated string
	err := tx.QueryRow(`SELECT id, created_at FROM audiences WHERE segment = ? AND email = ?`, member.Segment, member.Email).Scan(&existingID, &existingCreated)
	switch {
	case err == nil:
		member.ID = existingID
		member.CreatedAt = existingCreated
		_, err = tx.Exec(`
			UPDATE audiences
			SET display_name = ?, department = ?, title = ?, source = ?, created_at = ?
			WHERE segment = ? AND email = ?
		`,
			member.DisplayName,
			member.Department,
			member.Title,
			member.Source,
			member.CreatedAt,
			member.Segment,
			member.Email,
		)
		return false, err
	case errors.Is(err, sql.ErrNoRows):
		if member.ID > 0 {
			_, err = tx.Exec(`
				INSERT INTO audiences (id, segment, email, display_name, department, title, source, created_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`,
				member.ID,
				member.Segment,
				member.Email,
				member.DisplayName,
				member.Department,
				member.Title,
				member.Source,
				member.CreatedAt,
			)
			return true, err
		}

		result, err := tx.Exec(`
			INSERT INTO audiences (segment, email, display_name, department, title, source, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`,
			member.Segment,
			member.Email,
			member.DisplayName,
			member.Department,
			member.Title,
			member.Source,
			member.CreatedAt,
		)
		if err != nil {
			return false, err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return false, err
		}
		member.ID = uint64(id)
		return true, nil
	default:
		return false, err
	}
}

func upsertImportedMessageTx(tx *sql.Tx, message *ImportedMessage) (bool, error) {
	sourcePath := strings.TrimSpace(message.SourcePath)
	if sourcePath == "" {
		return false, errors.New("import source path is required")
	}

	now := nowRFC3339()
	message.SourcePath = sourcePath
	message.Plugin = emptyDefault(message.Plugin, "clone-importer")
	message.CreatedAt = emptyDefault(message.CreatedAt, now)

	var existingID uint64
	var existingCreated string
	err := tx.QueryRow(`SELECT id, created_at FROM imports WHERE source_path = ?`, message.SourcePath).Scan(&existingID, &existingCreated)
	switch {
	case err == nil:
		message.ID = existingID
		message.CreatedAt = existingCreated
		_, err = tx.Exec(`
			UPDATE imports
			SET plugin = ?, sender = ?, subject = ?, body = ?, generated_campaign = ?, created_at = ?
			WHERE source_path = ?
		`,
			message.Plugin,
			message.Sender,
			message.Subject,
			message.Body,
			message.GeneratedCampaign,
			message.CreatedAt,
			message.SourcePath,
		)
		return false, err
	case errors.Is(err, sql.ErrNoRows):
		if message.ID > 0 {
			_, err = tx.Exec(`
				INSERT INTO imports (id, plugin, source_path, sender, subject, body, generated_campaign, created_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`,
				message.ID,
				message.Plugin,
				message.SourcePath,
				message.Sender,
				message.Subject,
				message.Body,
				message.GeneratedCampaign,
				message.CreatedAt,
			)
			return true, err
		}

		result, err := tx.Exec(`
			INSERT INTO imports (plugin, source_path, sender, subject, body, generated_campaign, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`,
			message.Plugin,
			message.SourcePath,
			message.Sender,
			message.Subject,
			message.Body,
			message.GeneratedCampaign,
			message.CreatedAt,
		)
		if err != nil {
			return false, err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return false, err
		}
		message.ID = uint64(id)
		return true, nil
	default:
		return false, err
	}
}

func upsertCampaignSnapshotTx(tx *sql.Tx, snapshot *CampaignSnapshot) (bool, error) {
	sourcePath := strings.TrimSpace(snapshot.SourcePath)
	if sourcePath == "" {
		return false, errors.New("campaign snapshot source path is required")
	}

	now := nowRFC3339()
	snapshot.SourcePath = sourcePath
	snapshot.CreatedAt = emptyDefault(snapshot.CreatedAt, now)
	snapshot.LastValidated = emptyDefault(snapshot.LastValidated, now)
	issuesJSON, err := marshalStrings(snapshot.Issues)
	if err != nil {
		return false, err
	}

	var existingID uint64
	var existingCreated string
	err = tx.QueryRow(`SELECT id, created_at FROM campaigns WHERE source_path = ?`, snapshot.SourcePath).Scan(&existingID, &existingCreated)
	switch {
	case err == nil:
		snapshot.ID = existingID
		snapshot.CreatedAt = existingCreated
		_, err = tx.Exec(`
			UPDATE campaigns
			SET name = ?, owner = ?, audience = ?, region = ?, risk_level = ?, readiness = ?, issue_count = ?, issues_json = ?, source = ?, created_at = ?, last_validated = ?
			WHERE source_path = ?
		`,
			snapshot.Name,
			snapshot.Owner,
			snapshot.Audience,
			snapshot.Region,
			snapshot.RiskLevel,
			snapshot.Readiness,
			snapshot.IssueCount,
			issuesJSON,
			snapshot.Source,
			snapshot.CreatedAt,
			snapshot.LastValidated,
			snapshot.SourcePath,
		)
		return false, err
	case errors.Is(err, sql.ErrNoRows):
		if snapshot.ID > 0 {
			_, err = tx.Exec(`
				INSERT INTO campaigns (id, name, source_path, owner, audience, region, risk_level, readiness, issue_count, issues_json, source, created_at, last_validated)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`,
				snapshot.ID,
				snapshot.Name,
				snapshot.SourcePath,
				snapshot.Owner,
				snapshot.Audience,
				snapshot.Region,
				snapshot.RiskLevel,
				snapshot.Readiness,
				snapshot.IssueCount,
				issuesJSON,
				snapshot.Source,
				snapshot.CreatedAt,
				snapshot.LastValidated,
			)
			return true, err
		}

		result, err := tx.Exec(`
			INSERT INTO campaigns (name, source_path, owner, audience, region, risk_level, readiness, issue_count, issues_json, source, created_at, last_validated)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			snapshot.Name,
			snapshot.SourcePath,
			snapshot.Owner,
			snapshot.Audience,
			snapshot.Region,
			snapshot.RiskLevel,
			snapshot.Readiness,
			snapshot.IssueCount,
			issuesJSON,
			snapshot.Source,
			snapshot.CreatedAt,
			snapshot.LastValidated,
		)
		if err != nil {
			return false, err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return false, err
		}
		snapshot.ID = uint64(id)
		return true, nil
	default:
		return false, err
	}
}

func insertEventTx(tx *sql.Tx, event *Event) error {
	now := nowRFC3339()
	event.Type = strings.ToLower(strings.TrimSpace(event.Type))
	event.AudienceEmail = strings.ToLower(strings.TrimSpace(event.AudienceEmail))
	event.CreatedAt = emptyDefault(event.CreatedAt, now)

	var (
		result sql.Result
		err    error
	)
	if event.ID > 0 {
		_, err = tx.Exec(`
			INSERT INTO events (id, campaign_name, audience_email, type, source, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`,
			event.ID,
			event.CampaignName,
			event.AudienceEmail,
			event.Type,
			event.Source,
			event.CreatedAt,
		)
		return err
	}

	result, err = tx.Exec(`
		INSERT INTO events (campaign_name, audience_email, type, source, created_at)
		VALUES (?, ?, ?, ?, ?)
	`,
		event.CampaignName,
		event.AudienceEmail,
		event.Type,
		event.Source,
		event.CreatedAt,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	event.ID = uint64(id)

	return nil
}

func upsertFindingTx(tx *sql.Tx, source string, finding *Finding) (bool, error) {
	finding.Source = emptyDefault(finding.Source, source)
	finding.Source = strings.TrimSpace(finding.Source)
	finding.Tool = strings.TrimSpace(finding.Tool)
	finding.Rule = strings.TrimSpace(finding.Rule)
	finding.Severity = strings.ToLower(strings.TrimSpace(finding.Severity))
	finding.Target = strings.TrimSpace(finding.Target)
	finding.Summary = strings.TrimSpace(finding.Summary)
	finding.CreatedAt = emptyDefault(finding.CreatedAt, nowRFC3339())

	if finding.Source == "" {
		return false, errors.New("finding source is required")
	}
	if finding.Tool == "" {
		return false, errors.New("finding tool is required")
	}
	if finding.Rule == "" {
		return false, errors.New("finding rule is required")
	}
	if finding.Target == "" {
		return false, errors.New("finding target is required")
	}
	if finding.Severity == "" {
		finding.Severity = "info"
	}
	if finding.Summary == "" {
		finding.Summary = finding.Rule
	}

	metadataJSON, err := marshalStringMap(finding.Metadata)
	if err != nil {
		return false, err
	}

	var existingID uint64
	var existingCreated string
	err = tx.QueryRow(`
		SELECT id, created_at
		FROM findings
		WHERE source = ? AND tool = ? AND rule_name = ? AND target = ?
	`,
		finding.Source,
		finding.Tool,
		finding.Rule,
		finding.Target,
	).Scan(&existingID, &existingCreated)
	switch {
	case err == nil:
		finding.ID = existingID
		finding.CreatedAt = existingCreated
		_, err = tx.Exec(`
			UPDATE findings
			SET severity = ?, summary = ?, metadata_json = ?, created_at = ?
			WHERE source = ? AND tool = ? AND rule_name = ? AND target = ?
		`,
			finding.Severity,
			finding.Summary,
			metadataJSON,
			finding.CreatedAt,
			finding.Source,
			finding.Tool,
			finding.Rule,
			finding.Target,
		)
		return false, err
	case errors.Is(err, sql.ErrNoRows):
		if finding.ID > 0 {
			_, err = tx.Exec(`
				INSERT INTO findings (id, source, tool, rule_name, severity, target, summary, metadata_json, created_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`,
				finding.ID,
				finding.Source,
				finding.Tool,
				finding.Rule,
				finding.Severity,
				finding.Target,
				finding.Summary,
				metadataJSON,
				finding.CreatedAt,
			)
			return true, err
		}

		result, err := tx.Exec(`
			INSERT INTO findings (source, tool, rule_name, severity, target, summary, metadata_json, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`,
			finding.Source,
			finding.Tool,
			finding.Rule,
			finding.Severity,
			finding.Target,
			finding.Summary,
			metadataJSON,
			finding.CreatedAt,
		)
		if err != nil {
			return false, err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return false, err
		}
		finding.ID = uint64(id)
		return true, nil
	default:
		return false, err
	}
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

func marshalStrings(values []string) (string, error) {
	payload, err := json.Marshal(values)
	if err != nil {
		return "", err
	}

	return string(payload), nil
}

func unmarshalStrings(payload string) []string {
	if strings.TrimSpace(payload) == "" {
		return nil
	}

	var values []string
	if err := json.Unmarshal([]byte(payload), &values); err != nil {
		return nil
	}

	return values
}

func marshalStringMap(values map[string]string) (string, error) {
	if values == nil {
		values = map[string]string{}
	}

	payload, err := json.Marshal(values)
	if err != nil {
		return "", err
	}

	return string(payload), nil
}

func unmarshalStringMap(payload string) map[string]string {
	if strings.TrimSpace(payload) == "" {
		return map[string]string{}
	}

	values := map[string]string{}
	if err := json.Unmarshal([]byte(payload), &values); err != nil {
		return map[string]string{}
	}

	return values
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

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

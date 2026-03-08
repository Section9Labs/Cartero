package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/Section9Labs/Cartero/internal/doctor"
	"github.com/Section9Labs/Cartero/internal/plugin"
	"github.com/Section9Labs/Cartero/internal/store"
)

type Server struct {
	root  string
	store *store.Store
	tmpl  *template.Template
}

type pageData struct {
	Title          string
	Active         string
	BodyTemplate   string
	BodyHTML       template.HTML
	Root           string
	DatabasePath   string
	GeneratedAt    string
	Stats          store.WorkspaceStats
	Segments       []store.SegmentSummary
	EventsByType   []store.EventSummary
	Templates      []store.Template
	Audience       []store.AudienceMember
	Campaigns      []store.CampaignSnapshot
	Events         []store.Event
	Imports        []store.ImportedMessage
	Findings       []store.Finding
	Plugins        []plugin.Manifest
	PluginWarnings []plugin.Warning
	Checks         []doctor.Check
	Testing        testingData
}

type testingData struct {
	Template store.Template
	Campaign string
	Email    string
}

type testingEventPayload struct {
	Slug     string `json:"slug"`
	Campaign string `json:"campaign"`
	Email    string `json:"email"`
	Type     string `json:"type"`
}

func New(root string, workspaceStore *store.Store) (*Server, error) {
	tmpl, err := template.New("base").Funcs(template.FuncMap{
		"join":          strings.Join,
		"truncate":      truncate,
		"eventBadge":    eventBadge,
		"checkBadge":    checkBadge,
		"readinessTone": readinessTone,
	}).Parse(baseTemplate + dashboardTemplate + templatesTemplate + audiencesTemplate + campaignsTemplate + eventsTemplate + findingsTemplate + importsTemplate + testingTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse web templates: %w", err)
	}

	return &Server{
		root:  root,
		store: workspaceStore,
		tmpl:  tmpl,
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleDashboard)
	mux.HandleFunc("/templates", s.handleTemplates)
	mux.HandleFunc("/audiences", s.handleAudiences)
	mux.HandleFunc("/campaigns", s.handleCampaigns)
	mux.HandleFunc("/events", s.handleEvents)
	mux.HandleFunc("/findings", s.handleFindings)
	mux.HandleFunc("/imports", s.handleImports)
	mux.HandleFunc("/testing/", s.handleTestingPage)
	mux.HandleFunc("/api/testing/event", s.handleTestingEvent)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		mux.ServeHTTP(w, r)
	})
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data, err := s.basePage("dashboard", "Operations Board")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data.BodyTemplate = "dashboard"
	if data.Templates, err = s.topTemplates(4); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if data.Campaigns, err = s.topCampaigns(4); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if data.Imports, err = s.topImports(4); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if data.Findings, err = s.topFindings(4); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if data.Audience, err = s.topAudience(6); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	report := doctor.Run(s.root)
	data.Checks = report.Checks

	discovery, discoveryErr := plugin.Discover(filepath.Join(s.root, "plugins"))
	if discoveryErr != nil {
		data.PluginWarnings = []plugin.Warning{{
			Path:    filepath.Join(s.root, "plugins"),
			Message: discoveryErr.Error(),
		}}
	} else {
		data.Plugins = discovery.Manifests
		data.PluginWarnings = discovery.Warnings
	}

	s.render(w, "base", data)
}

func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request) {
	data, err := s.basePage("templates", "Template Library")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data.BodyTemplate = "templates"
	data.Templates, err = s.store.ListTemplates(store.TemplateFilter{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.render(w, "base", data)
}

func (s *Server) handleAudiences(w http.ResponseWriter, r *http.Request) {
	data, err := s.basePage("audiences", "Audience Segments")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data.BodyTemplate = "audiences"
	data.Audience, err = s.store.ListAudienceMembers(store.AudienceFilter{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.render(w, "base", data)
}

func (s *Server) handleCampaigns(w http.ResponseWriter, r *http.Request) {
	data, err := s.basePage("campaigns", "Campaign Timeline")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data.BodyTemplate = "campaigns"
	data.Campaigns, err = s.store.ListCampaignSnapshots()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.render(w, "base", data)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	data, err := s.basePage("events", "Signal Ledger")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data.BodyTemplate = "events"
	data.Events, err = s.store.ListEvents()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.render(w, "base", data)
}

func (s *Server) handleFindings(w http.ResponseWriter, r *http.Request) {
	data, err := s.basePage("findings", "Imported Findings")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data.BodyTemplate = "findings"
	data.Findings, err = s.store.ListFindings(store.FindingFilter{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.render(w, "base", data)
}

func (s *Server) handleImports(w http.ResponseWriter, r *http.Request) {
	data, err := s.basePage("imports", "Reviewed Imports")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data.BodyTemplate = "imports"
	data.Imports, err = s.store.ListImportedMessages()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.render(w, "base", data)
}

func (s *Server) handleTestingPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	slug := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/testing/"))
	if slug == "" {
		http.NotFound(w, r)
		return
	}

	selected, ok, err := s.findTemplate(slug)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		http.NotFound(w, r)
		return
	}

	data, err := s.basePage("testing", selected.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data.Title = selected.Name
	data.Testing = testingData{
		Template: selected,
		Campaign: strings.TrimSpace(r.URL.Query().Get("campaign")),
		Email:    strings.TrimSpace(r.URL.Query().Get("email")),
	}
	if data.Testing.Campaign == "" {
		data.Testing.Campaign = selected.Name
	}

	s.render(w, "testing-root", data)
}

func (s *Server) handleTestingEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()

	var payload testingEventPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}

	eventType := strings.ToLower(strings.TrimSpace(payload.Type))
	if _, ok := allowedTestingEvents[eventType]; !ok {
		http.Error(w, "unsupported event type", http.StatusBadRequest)
		return
	}

	selected, ok, err := s.findTemplate(strings.TrimSpace(payload.Slug))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "unknown template", http.StatusNotFound)
		return
	}

	campaign := strings.TrimSpace(payload.Campaign)
	if campaign == "" {
		campaign = selected.Name
	}

	if _, err := s.store.SaveEvent(store.Event{
		CampaignName:  campaign,
		AudienceEmail: strings.TrimSpace(payload.Email),
		Type:          eventType,
		Source:        "testing-runtime/" + selected.Slug,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) basePage(active, title string) (pageData, error) {
	stats, err := s.store.Stats()
	if err != nil {
		return pageData{}, err
	}
	segments, err := s.store.SegmentSummaries()
	if err != nil {
		return pageData{}, err
	}
	eventsByType, err := s.store.EventSummaries()
	if err != nil {
		return pageData{}, err
	}

	return pageData{
		Title:        title,
		Active:       active,
		Root:         s.root,
		DatabasePath: stats.DatabasePath,
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		Stats:        stats,
		Segments:     segments,
		EventsByType: eventsByType,
	}, nil
}

func (s *Server) findTemplate(slug string) (store.Template, bool, error) {
	templates, err := s.store.ListTemplates(store.TemplateFilter{})
	if err != nil {
		return store.Template{}, false, err
	}
	for _, candidate := range templates {
		if strings.EqualFold(candidate.Slug, slug) {
			return candidate, true, nil
		}
	}

	return store.Template{}, false, nil
}

func (s *Server) topTemplates(limit int) ([]store.Template, error) {
	templates, err := s.store.ListTemplates(store.TemplateFilter{})
	if err != nil {
		return nil, err
	}
	return clipTemplates(templates, limit), nil
}

func (s *Server) topCampaigns(limit int) ([]store.CampaignSnapshot, error) {
	campaigns, err := s.store.ListCampaignSnapshots()
	if err != nil {
		return nil, err
	}
	return clipCampaigns(campaigns, limit), nil
}

func (s *Server) topImports(limit int) ([]store.ImportedMessage, error) {
	imports, err := s.store.ListImportedMessages()
	if err != nil {
		return nil, err
	}
	return clipImports(imports, limit), nil
}

func (s *Server) topAudience(limit int) ([]store.AudienceMember, error) {
	audience, err := s.store.ListAudienceMembers(store.AudienceFilter{})
	if err != nil {
		return nil, err
	}
	return clipAudience(audience, limit), nil
}

func (s *Server) topFindings(limit int) ([]store.Finding, error) {
	findings, err := s.store.ListFindings(store.FindingFilter{})
	if err != nil {
		return nil, err
	}
	return clipFindings(findings, limit), nil
}

func (s *Server) render(w http.ResponseWriter, name string, data pageData) {
	if name == "base" && data.BodyTemplate != "" {
		var body bytes.Buffer
		if err := s.tmpl.ExecuteTemplate(&body, data.BodyTemplate, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data.BodyHTML = template.HTML(body.String())
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var allowedTestingEvents = map[string]struct{}{
	"page-view":      {},
	"field-focus":    {},
	"submit-attempt": {},
	"completion":     {},
}

func clipTemplates(items []store.Template, limit int) []store.Template {
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}

func clipCampaigns(items []store.CampaignSnapshot, limit int) []store.CampaignSnapshot {
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}

func clipImports(items []store.ImportedMessage, limit int) []store.ImportedMessage {
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}

func clipAudience(items []store.AudienceMember, limit int) []store.AudienceMember {
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}

func clipFindings(items []store.Finding, limit int) []store.Finding {
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}

func truncate(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	if limit <= 3 {
		return value[:limit]
	}
	return value[:limit-3] + "..."
}

func eventBadge(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "page-view":
		return "badge badge-blue"
	case "field-focus":
		return "badge badge-sand"
	case "submit-attempt":
		return "badge badge-red"
	case "completion":
		return "badge badge-green"
	default:
		return "badge"
	}
}

func checkBadge(status doctor.Status) string {
	switch status {
	case doctor.StatusPass:
		return "badge badge-green"
	case doctor.StatusWarn:
		return "badge badge-sand"
	default:
		return "badge badge-red"
	}
}

func readinessTone(score int) string {
	switch {
	case score >= 90:
		return "tone-good"
	case score >= 75:
		return "tone-watch"
	default:
		return "tone-risk"
	}
}

const baseTemplate = `
{{define "base"}}
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}} - Cartero</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Fraunces:opsz,wght@9..144,600;9..144,700&family=IBM+Plex+Mono:wght@400;500&family=IBM+Plex+Sans:wght@400;500;600;700&display=swap" rel="stylesheet">
  <style>
    :root {
      --paper: #f4ede1;
      --paper-strong: #efe4d4;
      --ink: #11212d;
      --ink-soft: rgba(17, 33, 45, 0.72);
      --signal: #ca4d2d;
      --signal-soft: rgba(202, 77, 45, 0.14);
      --teal: #1f6f78;
      --moss: #4e6c50;
      --gold: #b5822b;
      --line: rgba(17, 33, 45, 0.14);
      --panel: rgba(255, 251, 245, 0.82);
      --shadow: 0 18px 50px rgba(17, 33, 45, 0.12);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      color: var(--ink);
      font-family: "IBM Plex Sans", sans-serif;
      background:
        radial-gradient(circle at top left, rgba(202, 77, 45, 0.18), transparent 32%),
        linear-gradient(130deg, rgba(31, 111, 120, 0.07), transparent 45%),
        repeating-linear-gradient(90deg, rgba(17, 33, 45, 0.03) 0 1px, transparent 1px 92px),
        var(--paper);
    }
    a { color: inherit; }
    .shell {
      width: min(1180px, calc(100vw - 40px));
      margin: 0 auto;
      padding: 28px 0 44px;
    }
    .masthead {
      position: relative;
      overflow: hidden;
      border: 1px solid var(--line);
      background:
        linear-gradient(135deg, rgba(255,255,255,0.7), rgba(255,255,255,0.2)),
        linear-gradient(90deg, rgba(202, 77, 45, 0.08), rgba(31, 111, 120, 0.08));
      box-shadow: var(--shadow);
      border-radius: 26px;
      padding: 30px;
    }
    .masthead::after {
      content: "";
      position: absolute;
      inset: 18px;
      border: 1px dashed rgba(17, 33, 45, 0.12);
      border-radius: 18px;
      pointer-events: none;
    }
    .eyebrow, .meta, .nav a, .stat-kicker, .table-label, .pill, .helper, .timestamp {
      font-family: "IBM Plex Mono", monospace;
      letter-spacing: 0.08em;
      text-transform: uppercase;
    }
    .eyebrow {
      font-size: 0.72rem;
      color: var(--signal);
      margin-bottom: 12px;
    }
    h1, h2, h3 {
      margin: 0;
      font-family: "Fraunces", serif;
      font-weight: 700;
      line-height: 0.96;
    }
    .hero {
      display: grid;
      gap: 18px;
      grid-template-columns: 2.2fr 1fr;
      align-items: end;
    }
    .hero-copy h1 {
      font-size: clamp(3rem, 7vw, 5.6rem);
      max-width: 10ch;
    }
    .hero-copy p {
      margin: 18px 0 0;
      max-width: 55ch;
      line-height: 1.6;
      color: var(--ink-soft);
      font-size: 1rem;
    }
    .hero-meta {
      display: grid;
      gap: 10px;
      align-self: stretch;
      justify-items: end;
    }
    .hero-chip {
      display: inline-flex;
      align-items: center;
      gap: 10px;
      padding: 12px 14px;
      border: 1px solid var(--line);
      border-radius: 14px;
      background: rgba(255, 251, 245, 0.72);
      backdrop-filter: blur(10px);
      min-width: min(100%, 340px);
    }
    .hero-chip strong {
      display: block;
      font-size: 0.86rem;
      margin-top: 4px;
      letter-spacing: normal;
      text-transform: none;
      font-family: "IBM Plex Sans", sans-serif;
    }
    .nav {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
      margin-top: 22px;
    }
    .nav a {
      text-decoration: none;
      font-size: 0.76rem;
      color: var(--ink-soft);
      padding: 12px 14px;
      border-radius: 999px;
      border: 1px solid transparent;
      background: rgba(17, 33, 45, 0.05);
      transition: transform 160ms ease, border-color 160ms ease, background 160ms ease;
    }
    .nav a:hover {
      transform: translateY(-1px);
      border-color: var(--line);
    }
    .nav a.active {
      color: #fff;
      background: var(--ink);
      border-color: var(--ink);
    }
    .board {
      display: grid;
      gap: 18px;
      margin-top: 18px;
    }
    .stats {
      display: grid;
      gap: 14px;
      grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
    }
    .stat {
      border: 1px solid var(--line);
      border-radius: 20px;
      padding: 18px;
      background: var(--panel);
      box-shadow: var(--shadow);
      min-height: 128px;
      display: flex;
      flex-direction: column;
      justify-content: space-between;
    }
    .stat strong {
      font-size: clamp(2rem, 3.2vw, 3rem);
      font-family: "Fraunces", serif;
      line-height: 1;
    }
    .stat-kicker {
      font-size: 0.72rem;
      color: var(--ink-soft);
    }
    .panel-grid {
      display: grid;
      gap: 18px;
      grid-template-columns: 1.35fr 1fr;
    }
    .panel {
      border: 1px solid var(--line);
      border-radius: 24px;
      background: var(--panel);
      box-shadow: var(--shadow);
      padding: 22px;
    }
    .panel h2 {
      font-size: clamp(1.8rem, 3vw, 2.8rem);
    }
    .panel-head {
      display: flex;
      gap: 14px;
      justify-content: space-between;
      align-items: end;
      margin-bottom: 18px;
    }
    .panel-head p {
      margin: 8px 0 0;
      color: var(--ink-soft);
      max-width: 50ch;
      line-height: 1.55;
    }
    .stack {
      display: grid;
      gap: 12px;
    }
    .row {
      display: grid;
      gap: 14px;
      grid-template-columns: 1fr auto;
      align-items: center;
      border-top: 1px solid var(--line);
      padding-top: 12px;
    }
    .row:first-child {
      border-top: none;
      padding-top: 0;
    }
    .row strong {
      font-size: 1rem;
    }
    .row p {
      margin: 4px 0 0;
      color: var(--ink-soft);
      line-height: 1.55;
    }
    .badge {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      padding: 8px 10px;
      border-radius: 999px;
      font-size: 0.72rem;
      border: 1px solid var(--line);
      background: rgba(17, 33, 45, 0.06);
      font-family: "IBM Plex Mono", monospace;
      letter-spacing: 0.05em;
      text-transform: uppercase;
      white-space: nowrap;
    }
    .badge-red { background: var(--signal-soft); color: var(--signal); border-color: rgba(202, 77, 45, 0.28); }
    .badge-green { background: rgba(78, 108, 80, 0.12); color: var(--moss); border-color: rgba(78, 108, 80, 0.24); }
    .badge-sand { background: rgba(181, 130, 43, 0.12); color: var(--gold); border-color: rgba(181, 130, 43, 0.22); }
    .badge-blue { background: rgba(31, 111, 120, 0.12); color: var(--teal); border-color: rgba(31, 111, 120, 0.24); }
    .table-shell {
      overflow: hidden;
      border-radius: 22px;
      border: 1px solid var(--line);
      background: rgba(255, 251, 245, 0.82);
    }
    table {
      width: 100%;
      border-collapse: collapse;
    }
    th, td {
      padding: 16px 18px;
      text-align: left;
      vertical-align: top;
      border-bottom: 1px solid var(--line);
    }
    th {
      font-size: 0.74rem;
      color: var(--ink-soft);
      font-family: "IBM Plex Mono", monospace;
      letter-spacing: 0.08em;
      text-transform: uppercase;
      background: rgba(17, 33, 45, 0.04);
    }
    tr:last-child td { border-bottom: none; }
    .table-title {
      font-weight: 700;
      display: block;
      margin-bottom: 6px;
    }
    .table-note {
      color: var(--ink-soft);
      line-height: 1.5;
      max-width: 46ch;
    }
    .table-actions a {
      text-decoration: none;
      border-bottom: 1px solid currentColor;
      color: var(--signal);
    }
    .timeline {
      display: grid;
      gap: 14px;
    }
    .timeline-item {
      display: grid;
      gap: 10px;
      padding: 18px;
      border: 1px solid var(--line);
      border-radius: 18px;
      background: rgba(255, 255, 255, 0.55);
    }
    .timeline-top {
      display: flex;
      gap: 12px;
      align-items: center;
      justify-content: space-between;
    }
    .helper, .timestamp {
      color: var(--ink-soft);
      font-size: 0.74rem;
    }
    .grid-two {
      display: grid;
      gap: 18px;
      grid-template-columns: 1fr 1fr;
    }
    .empty {
      padding: 24px;
      border: 1px dashed var(--line);
      border-radius: 16px;
      color: var(--ink-soft);
      line-height: 1.6;
      background: rgba(255,255,255,0.45);
    }
    .tone-good { color: var(--moss); }
    .tone-watch { color: var(--gold); }
    .tone-risk { color: var(--signal); }
    @media (max-width: 1080px) {
      .hero, .panel-grid, .grid-two, .stats { grid-template-columns: 1fr 1fr; }
      .hero-meta { justify-items: start; }
    }
    @media (max-width: 760px) {
      .shell { width: min(100vw - 24px, 1180px); padding-top: 16px; }
      .masthead, .panel, .stat { padding: 18px; }
      .hero, .panel-grid, .grid-two, .stats { grid-template-columns: 1fr; }
      .hero-copy h1 { max-width: none; }
      th:nth-child(4), td:nth-child(4) { display: none; }
    }
  </style>
</head>
<body>
  <div class="shell">
    <header class="masthead">
      <div class="eyebrow">Cartero local control room</div>
      <div class="hero">
        <div class="hero-copy">
          <h1>{{.Title}}</h1>
          <p>Single-binary workspace operations for templates, audiences, reviewed imports, campaign snapshots, and safe interaction telemetry.</p>
        </div>
        <div class="hero-meta">
          <div class="hero-chip">
            <div>
              <div class="helper">Workspace root</div>
              <strong>{{.Root}}</strong>
            </div>
          </div>
          <div class="hero-chip">
            <div>
              <div class="helper">Embedded database</div>
              <strong>{{.DatabasePath}}</strong>
            </div>
          </div>
        </div>
      </div>
      <nav class="nav">
        <a href="/" class="{{if eq .Active "dashboard"}}active{{end}}">Dashboard</a>
        <a href="/templates" class="{{if eq .Active "templates"}}active{{end}}">Templates</a>
        <a href="/audiences" class="{{if eq .Active "audiences"}}active{{end}}">Audiences</a>
        <a href="/campaigns" class="{{if eq .Active "campaigns"}}active{{end}}">Campaigns</a>
        <a href="/events" class="{{if eq .Active "events"}}active{{end}}">Events</a>
        <a href="/findings" class="{{if eq .Active "findings"}}active{{end}}">Findings</a>
        <a href="/imports" class="{{if eq .Active "imports"}}active{{end}}">Imports</a>
      </nav>
    </header>

    <main class="board">
      <section class="stats">
        <article class="stat">
          <div class="stat-kicker">Templates</div>
          <strong>{{.Stats.TemplateCount}}</strong>
          <div class="helper">Seeded scenarios ready for testing pages</div>
        </article>
        <article class="stat">
          <div class="stat-kicker">Audience</div>
          <strong>{{.Stats.AudienceCount}}</strong>
          <div class="helper">{{.Stats.SegmentCount}} active segments</div>
        </article>
        <article class="stat">
          <div class="stat-kicker">Imports</div>
          <strong>{{.Stats.ImportCount}}</strong>
          <div class="helper">Reviewed message drafts in workspace</div>
        </article>
        <article class="stat">
          <div class="stat-kicker">Campaigns</div>
          <strong>{{.Stats.CampaignCount}}</strong>
          <div class="helper">Persisted preview and validation snapshots</div>
        </article>
        <article class="stat">
          <div class="stat-kicker">Events</div>
          <strong>{{.Stats.EventCount}}</strong>
          <div class="helper">Safe interaction telemetry only</div>
        </article>
        <article class="stat">
          <div class="stat-kicker">Findings</div>
          <strong>{{.Stats.FindingCount}}</strong>
          <div class="helper">Imported scan and migration signals</div>
        </article>
        <article class="stat">
          <div class="stat-kicker">Generated</div>
          <strong>#{{len .EventsByType}}</strong>
          <div class="helper timestamp">{{.GeneratedAt}}</div>
        </article>
      </section>

      {{.BodyHTML}}
    </main>
  </div>
</body>
</html>
{{end}}
`

const dashboardTemplate = `
{{define "dashboard"}}
<section class="panel-grid">
  <article class="panel">
    <div class="panel-head">
      <div>
        <div class="eyebrow">Signals at a glance</div>
        <h2>Operator board</h2>
        <p>Health checks, first-party plugins, and the most recent workspace material in one place.</p>
      </div>
      <span class="badge">{{len .Checks}} checks</span>
    </div>
    <div class="stack">
      {{if .Checks}}
        {{range .Checks}}
          <div class="row">
            <div>
              <strong>{{.Name}}</strong>
              <p>{{.Detail}}{{if .Hint}} | {{.Hint}}{{end}}</p>
            </div>
            <span class="{{checkBadge .Status}}">{{.Status}}</span>
          </div>
        {{end}}
      {{else}}
        <div class="empty">No doctor checks are available yet for this workspace.</div>
      {{end}}
    </div>
  </article>

  <article class="panel">
    <div class="panel-head">
      <div>
        <div class="eyebrow">Plugin registry</div>
        <h2>Active manifests</h2>
      </div>
      <span class="badge">{{len .Plugins}} loaded</span>
    </div>
    <div class="stack">
      {{if .Plugins}}
        {{range .Plugins}}
          <div class="row">
            <div>
              <strong>{{.Name}}</strong>
              <p>{{.Description}}</p>
            </div>
            <span class="badge">{{join .Capabilities ", "}}</span>
          </div>
        {{end}}
      {{else}}
        <div class="empty">No plugin manifests were discovered in this workspace.</div>
      {{end}}
      {{if .PluginWarnings}}
        {{range .PluginWarnings}}
          <div class="row">
            <div>
              <strong>{{.Path}}</strong>
              <p>{{.Message}}</p>
            </div>
            <span class="badge badge-sand">warning</span>
          </div>
        {{end}}
      {{end}}
    </div>
  </article>
</section>

<section class="panel-grid">
  <article class="panel">
    <div class="panel-head">
      <div>
        <div class="eyebrow">Templates</div>
        <h2>Ready to launch</h2>
      </div>
      <a class="badge" href="/templates">Full library</a>
    </div>
    <div class="stack">
      {{if .Templates}}
        {{range .Templates}}
          <div class="row">
            <div>
              <strong>{{.Name}}</strong>
              <p>{{.Subject}} | {{.Locale}} / {{.Department}} / {{.Scenario}}</p>
            </div>
            <a class="badge badge-blue" href="/testing/{{.Slug}}?campaign={{.Name}}">Launch safe page</a>
          </div>
        {{end}}
      {{else}}
        <div class="empty">No templates are seeded yet.</div>
      {{end}}
    </div>
  </article>

  <article class="panel">
    <div class="panel-head">
      <div>
        <div class="eyebrow">Recent campaigns</div>
        <h2>Readiness snapshots</h2>
      </div>
      <a class="badge" href="/campaigns">All snapshots</a>
    </div>
    <div class="timeline">
      {{if .Campaigns}}
        {{range .Campaigns}}
          <article class="timeline-item">
            <div class="timeline-top">
              <strong>{{.Name}}</strong>
              <span class="{{readinessTone .Readiness}}">{{.Readiness}} / 100</span>
            </div>
            <div class="helper">{{.Audience}} | {{.Region}} | {{.RiskLevel}} | {{.Source}}</div>
            <div class="table-note">{{truncate .SourcePath 82}}</div>
          </article>
        {{end}}
      {{else}}
        <div class="empty">Preview or validate a campaign to start building the history.</div>
      {{end}}
    </div>
  </article>
</section>

<section class="grid-two">
  <article class="panel">
    <div class="panel-head">
      <div>
        <div class="eyebrow">Segments</div>
        <h2>Audience density</h2>
      </div>
    </div>
    <div class="stack">
      {{if .Segments}}
        {{range .Segments}}
          <div class="row">
            <div>
              <strong>{{.Segment}}</strong>
              <p>Segment members available for sync and event correlation.</p>
            </div>
            <span class="badge badge-green">{{.Members}} members</span>
          </div>
        {{end}}
      {{else}}
        <div class="empty">Import a CSV audience segment to populate this board.</div>
      {{end}}
    </div>
  </article>

  <article class="panel">
    <div class="panel-head">
      <div>
        <div class="eyebrow">Engagement mix</div>
        <h2>Events by type</h2>
      </div>
    </div>
    <div class="stack">
      {{if .EventsByType}}
        {{range .EventsByType}}
          <div class="row">
            <div>
              <strong>{{.Type}}</strong>
              <p>Recorded through the CLI or safe testing runtime.</p>
            </div>
            <span class="{{eventBadge .Type}}">{{.Count}} hits</span>
          </div>
        {{end}}
      {{else}}
        <div class="empty">Event telemetry appears here after running cartero event record or a testing page interaction.</div>
      {{end}}
    </div>
  </article>
</section>

<section class="panel">
  <div class="panel-head">
    <div>
      <div class="eyebrow">Imported findings</div>
      <h2>Correlated scan signals</h2>
      <p>Scanner output and legacy migration artifacts land in the same workspace for operator review and export.</p>
    </div>
    <a class="badge" href="/findings">Finding registry</a>
  </div>
  <div class="stack">
    {{if .Findings}}
      {{range .Findings}}
        <div class="row">
          <div>
            <strong>{{.Rule}}</strong>
            <p>{{.Summary}}</p>
          </div>
          <span class="badge">{{.Tool}} | {{.Severity}}</span>
        </div>
      {{end}}
    {{else}}
      <div class="empty">Import scanner output with cartero finding import or migrate a legacy export to populate the registry.</div>
    {{end}}
  </div>
</section>

<section class="panel">
  <div class="panel-head">
    <div>
      <div class="eyebrow">Recent imports</div>
      <h2>Reviewed messages</h2>
      <p>Imported drafts become campaign material without leaving the local workspace.</p>
    </div>
    <a class="badge" href="/imports">Import ledger</a>
  </div>
  <div class="table-shell">
    <table>
      <thead>
        <tr>
          <th>Subject</th>
          <th>Sender</th>
          <th>Draft</th>
          <th>Created</th>
        </tr>
      </thead>
      <tbody>
        {{if .Imports}}
          {{range .Imports}}
            <tr>
              <td>
                <span class="table-title">{{.Subject}}</span>
                <span class="table-note">{{truncate .Body 96}}</span>
              </td>
              <td>{{.Sender}}</td>
              <td class="table-note">{{truncate .GeneratedCampaign 78}}</td>
              <td class="timestamp">{{.CreatedAt}}</td>
            </tr>
          {{end}}
        {{else}}
          <tr>
            <td colspan="4"><div class="empty">Import a reviewed .eml file with cartero import clone to start a local message ledger.</div></td>
          </tr>
        {{end}}
      </tbody>
    </table>
  </div>
</section>
{{end}}
`

const templatesTemplate = `
{{define "templates"}}
<section class="panel">
  <div class="panel-head">
    <div>
      <div class="eyebrow">First-party library</div>
      <h2>Scenario templates</h2>
      <p>Each template can be launched as a safe testing page that records interaction events without retaining submitted values.</p>
    </div>
    <span class="badge">{{len .Templates}} total</span>
  </div>
  <div class="table-shell">
    <table>
      <thead>
        <tr>
          <th>Template</th>
          <th>Context</th>
          <th>Channels</th>
          <th>Preview</th>
        </tr>
      </thead>
      <tbody>
        {{if .Templates}}
          {{range .Templates}}
            <tr>
              <td>
                <span class="table-title">{{.Name}}</span>
                <span class="table-note">{{.Subject}}</span>
              </td>
              <td>{{.Locale}} | {{.Department}} | {{.Scenario}}</td>
              <td>{{join .Channels ", "}}</td>
              <td class="table-actions"><a href="/testing/{{.Slug}}?campaign={{.Name}}">Launch safe page</a></td>
            </tr>
          {{end}}
        {{else}}
          <tr>
            <td colspan="4"><div class="empty">Initialize the workspace or sync plugins to populate the template library.</div></td>
          </tr>
        {{end}}
      </tbody>
    </table>
  </div>
</section>
{{end}}
`

const audiencesTemplate = `
{{define "audiences"}}
<section class="panel">
  <div class="panel-head">
    <div>
      <div class="eyebrow">Audience registry</div>
      <h2>Synced recipients</h2>
      <p>CSV imports and future sync adapters land here with segment-level visibility for review and export.</p>
    </div>
    <span class="badge">{{len .Audience}} members</span>
  </div>
  <div class="table-shell">
    <table>
      <thead>
        <tr>
          <th>Recipient</th>
          <th>Segment</th>
          <th>Department</th>
          <th>Source</th>
        </tr>
      </thead>
      <tbody>
        {{if .Audience}}
          {{range .Audience}}
            <tr>
              <td>
                <span class="table-title">{{.DisplayName}}</span>
                <span class="table-note">{{.Email}}</span>
              </td>
              <td>{{.Segment}}</td>
              <td>{{.Department}} {{if .Title}}| {{.Title}}{{end}}</td>
              <td class="timestamp">{{.Source}}</td>
            </tr>
          {{end}}
        {{else}}
          <tr>
            <td colspan="4"><div class="empty">Run cartero audience import --segment ... --csv ... to seed the registry.</div></td>
          </tr>
        {{end}}
      </tbody>
    </table>
  </div>
</section>
{{end}}
`

const campaignsTemplate = `
{{define "campaigns"}}
<section class="panel">
  <div class="panel-head">
    <div>
      <div class="eyebrow">Campaign memory</div>
      <h2>Snapshot history</h2>
      <p>Every preview and validate run leaves a structured readiness checkpoint in the embedded workspace.</p>
    </div>
            <span class="badge">{{len .Campaigns}} snapshots</span>
  </div>
  <div class="timeline">
    {{if .Campaigns}}
      {{range .Campaigns}}
        <article class="timeline-item">
          <div class="timeline-top">
            <strong>{{.Name}}</strong>
            <span class="{{readinessTone .Readiness}}">{{.Readiness}} / 100</span>
          </div>
          <div class="helper">{{.Audience}} | {{.Region}} | {{.RiskLevel}} | {{.Source}}</div>
          <div class="table-note">{{truncate .SourcePath 110}}</div>
          <div class="stack">
            {{if .Issues}}
              {{range .Issues}}
                <div class="row">
                  <div><p>{{.}}</p></div>
                  <span class="badge badge-sand">issue</span>
                </div>
              {{end}}
            {{else}}
              <div class="helper">No issues were persisted with this snapshot.</div>
            {{end}}
          </div>
        </article>
      {{end}}
    {{else}}
      <div class="empty">Use cartero preview or cartero validate to populate campaign history.</div>
    {{end}}
  </div>
</section>
{{end}}
`

const eventsTemplate = `
{{define "events"}}
<section class="grid-two">
  <article class="panel">
    <div class="panel-head">
      <div>
        <div class="eyebrow">Event mix</div>
        <h2>By type</h2>
      </div>
    </div>
    <div class="stack">
      {{if .EventsByType}}
        {{range .EventsByType}}
          <div class="row">
            <div>
              <strong>{{.Type}}</strong>
              <p>Interaction count recorded from CLI and testing routes.</p>
            </div>
            <span class="{{eventBadge .Type}}">{{.Count}} hits</span>
          </div>
        {{end}}
      {{else}}
        <div class="empty">There are no events yet in this workspace.</div>
      {{end}}
    </div>
  </article>

  <article class="panel">
    <div class="panel-head">
      <div>
        <div class="eyebrow">Segments</div>
        <h2>Coverage view</h2>
      </div>
    </div>
    <div class="stack">
      {{if .Segments}}
        {{range .Segments}}
          <div class="row">
            <div>
              <strong>{{.Segment}}</strong>
              <p>Available audience records correlated against events.</p>
            </div>
            <span class="badge badge-green">{{.Members}} members</span>
          </div>
        {{end}}
      {{else}}
        <div class="empty">Segments appear here after an audience import.</div>
      {{end}}
    </div>
  </article>
</section>

<section class="panel">
  <div class="panel-head">
    <div>
      <div class="eyebrow">Full ledger</div>
      <h2>Recorded events</h2>
      <p>Safe runtime pages record page views, field focus, submit attempts, and completions without storing submitted values.</p>
    </div>
    <span class="badge">{{len .Events}} rows</span>
  </div>
  <div class="table-shell">
    <table>
      <thead>
        <tr>
          <th>Campaign</th>
          <th>Recipient</th>
          <th>Type</th>
          <th>Source</th>
        </tr>
      </thead>
      <tbody>
        {{if .Events}}
          {{range .Events}}
            <tr>
              <td>{{.CampaignName}}</td>
              <td>{{if .AudienceEmail}}{{.AudienceEmail}}{{else}}-{{end}}</td>
              <td><span class="{{eventBadge .Type}}">{{.Type}}</span></td>
              <td class="timestamp">{{.Source}} | {{.CreatedAt}}</td>
            </tr>
          {{end}}
        {{else}}
          <tr>
            <td colspan="4"><div class="empty">Use cartero event record or launch a testing page to start the ledger.</div></td>
          </tr>
        {{end}}
      </tbody>
    </table>
  </div>
</section>
{{end}}
`

const findingsTemplate = `
{{define "findings"}}
<section class="panel">
  <div class="panel-head">
    <div>
      <div class="eyebrow">Normalized registry</div>
      <h2>Imported findings</h2>
      <p>CSV, JSON, SARIF, JSONL, and legacy migration artifacts land here with a normalized shape for export and review.</p>
    </div>
    <span class="badge">{{len .Findings}} findings</span>
  </div>
  <div class="table-shell">
    <table>
      <thead>
        <tr>
          <th>Rule</th>
          <th>Tool</th>
          <th>Severity</th>
          <th>Target</th>
        </tr>
      </thead>
      <tbody>
        {{if .Findings}}
          {{range .Findings}}
            <tr>
              <td>
                <span class="table-title">{{.Rule}}</span>
                <span class="table-note">{{.Summary}}</span>
              </td>
              <td>{{.Tool}}</td>
              <td><span class="badge">{{.Severity}}</span></td>
              <td class="table-note">{{truncate .Target 82}}</td>
            </tr>
          {{end}}
        {{else}}
          <tr>
            <td colspan="4"><div class="empty">Import findings with cartero finding import or migrate a legacy export to populate this registry.</div></td>
          </tr>
        {{end}}
      </tbody>
    </table>
  </div>
</section>
{{end}}
`

const importsTemplate = `
{{define "imports"}}
<section class="panel">
  <div class="panel-head">
    <div>
      <div class="eyebrow">Reviewed sources</div>
      <h2>Import ledger</h2>
      <p>Clone imports stay local, link back to their source files, and preserve the generated campaign draft for operator review.</p>
    </div>
    <span class="badge">{{len .Imports}} imports</span>
  </div>
  <div class="table-shell">
    <table>
      <thead>
        <tr>
          <th>Source</th>
          <th>Subject</th>
          <th>Draft</th>
          <th>Created</th>
        </tr>
      </thead>
      <tbody>
        {{if .Imports}}
          {{range .Imports}}
            <tr>
              <td class="table-note">{{truncate .SourcePath 64}}</td>
              <td>
                <span class="table-title">{{.Subject}}</span>
                <span class="table-note">{{.Sender}}</span>
              </td>
              <td class="table-note">{{truncate .GeneratedCampaign 92}}</td>
              <td class="timestamp">{{.CreatedAt}}</td>
            </tr>
          {{end}}
        {{else}}
          <tr>
            <td colspan="4"><div class="empty">Imports appear here after running cartero import clone -f path/to/reviewed.eml.</div></td>
          </tr>
        {{end}}
      </tbody>
    </table>
  </div>
</section>
{{end}}
`

const testingTemplate = `
{{define "testing-root"}}
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Testing.Template.Name}} - Cartero testing page</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Fraunces:opsz,wght@9..144,600;9..144,700&family=IBM+Plex+Mono:wght@400;500&family=IBM+Plex+Sans:wght@400;500;600;700&display=swap" rel="stylesheet">
  <style>
    :root {
      --night: #0b141d;
      --night-soft: rgba(11, 20, 29, 0.72);
      --panel: rgba(244, 237, 225, 0.92);
      --panel-line: rgba(11, 20, 29, 0.12);
      --signal: #ca4d2d;
      --teal: #1f6f78;
      --paper: #f4ede1;
      --success: #4e6c50;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      display: grid;
      place-items: center;
      padding: 24px;
      color: var(--night);
      font-family: "IBM Plex Sans", sans-serif;
      background:
        radial-gradient(circle at top right, rgba(202, 77, 45, 0.34), transparent 26%),
        radial-gradient(circle at bottom left, rgba(31, 111, 120, 0.28), transparent 32%),
        linear-gradient(160deg, #0f1b27, #162533 48%, #13202b);
    }
    .shell {
      width: min(1040px, 100%);
      display: grid;
      gap: 18px;
      grid-template-columns: 1.2fr 0.8fr;
      align-items: stretch;
    }
    .hero, .probe {
      border-radius: 28px;
      padding: 28px;
      background: var(--panel);
      border: 1px solid var(--panel-line);
      box-shadow: 0 24px 60px rgba(0, 0, 0, 0.18);
    }
    .eyebrow, .meta, .status, button, label {
      font-family: "IBM Plex Mono", monospace;
      letter-spacing: 0.08em;
      text-transform: uppercase;
    }
    .eyebrow {
      font-size: 0.72rem;
      color: var(--signal);
      margin-bottom: 14px;
    }
    h1, h2 {
      margin: 0;
      font-family: "Fraunces", serif;
      line-height: 0.98;
    }
    h1 {
      font-size: clamp(3rem, 6vw, 5rem);
      max-width: 8ch;
    }
    .hero p {
      margin: 18px 0 0;
      line-height: 1.7;
      color: var(--night-soft);
      max-width: 48ch;
    }
    .meta {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
      margin-top: 18px;
      font-size: 0.72rem;
      color: var(--night-soft);
    }
    .meta span {
      display: inline-flex;
      gap: 8px;
      padding: 10px 12px;
      border-radius: 999px;
      border: 1px solid var(--panel-line);
      background: rgba(255,255,255,0.45);
    }
    .probe {
      display: grid;
      gap: 16px;
      align-content: start;
    }
    .probe h2 {
      font-size: 2rem;
    }
    .status {
      display: inline-flex;
      padding: 8px 10px;
      border-radius: 999px;
      background: rgba(202, 77, 45, 0.12);
      color: var(--signal);
      width: fit-content;
      font-size: 0.72rem;
    }
    .explain {
      padding: 14px 16px;
      border-radius: 18px;
      background: rgba(11, 20, 29, 0.05);
      line-height: 1.6;
      color: var(--night-soft);
    }
    form {
      display: grid;
      gap: 14px;
    }
    label {
      display: grid;
      gap: 8px;
      font-size: 0.72rem;
      color: var(--night-soft);
    }
    input {
      width: 100%;
      padding: 14px 15px;
      border-radius: 16px;
      border: 1px solid var(--panel-line);
      background: rgba(255,255,255,0.82);
      color: var(--night);
      font-size: 1rem;
      font-family: "IBM Plex Sans", sans-serif;
    }
    button {
      border: none;
      padding: 14px 18px;
      border-radius: 16px;
      background: linear-gradient(135deg, var(--signal), #df724d);
      color: #fff;
      cursor: pointer;
      font-size: 0.8rem;
    }
    .notice {
      padding: 12px 14px;
      border-radius: 16px;
      background: rgba(78, 108, 80, 0.12);
      color: var(--success);
      display: none;
      line-height: 1.5;
    }
    .notice.visible {
      display: block;
    }
    .helper {
      color: var(--night-soft);
      line-height: 1.6;
      font-size: 0.94rem;
    }
    @media (max-width: 900px) {
      .shell { grid-template-columns: 1fr; }
      h1 { max-width: none; }
    }
  </style>
</head>
<body>
  <div class="shell">
    <section class="hero">
      <div class="eyebrow">Safe testing route</div>
      <h1>{{.Testing.Template.Name}}</h1>
      <p>{{if .Testing.Template.LandingPage}}{{.Testing.Template.LandingPage}}{{else}}This page is a controlled training surface. It records page views, field focus, and submit attempts without keeping any submitted values.{{end}}</p>
      <div class="meta">
        <span>{{.Testing.Template.Locale}}</span>
        <span>{{.Testing.Template.Department}}</span>
        <span>{{.Testing.Template.Scenario}}</span>
        <span>{{if .Testing.Email}}{{.Testing.Email}}{{else}}anonymous preview{{end}}</span>
      </div>
    </section>

    <section class="probe">
      <div class="status">No submitted values are retained</div>
      <div>
        <h2>Interaction probe</h2>
        <p class="helper">This exercise never stores submitted values. It only records event metadata so analysts can review interaction patterns inside the local workspace.</p>
      </div>
      <div class="explain">
        Campaign: {{.Testing.Campaign}}<br>
        Template slug: {{.Testing.Template.Slug}}<br>
        Subject line: {{.Testing.Template.Subject}}
      </div>
      <form id="probe-form" data-slug="{{.Testing.Template.Slug}}" data-campaign="{{.Testing.Campaign}}" data-email="{{.Testing.Email}}">
        <label>
          Verification keyword
          <input type="text" name="keyword" autocomplete="off" placeholder="Type anything to simulate focus">
        </label>
        <label>
          Access note
          <input type="text" name="note" autocomplete="off" placeholder="This value is never persisted">
        </label>
        <button type="submit">Record submit attempt</button>
      </form>
      <div id="notice" class="notice">Submit attempt recorded. Cartero stored only event metadata for analyst review.</div>
    </section>
  </div>

  <script>
    const form = document.getElementById("probe-form");
    const notice = document.getElementById("notice");
    let focusSent = false;

    const postEvent = async (type) => {
      await fetch("/api/testing/event", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          slug: form.dataset.slug,
          campaign: form.dataset.campaign,
          email: form.dataset.email,
          type
        })
      });
    };

    window.addEventListener("load", () => { postEvent("page-view"); });
    form.addEventListener("focusin", () => {
      if (focusSent) return;
      focusSent = true;
      postEvent("field-focus");
    });
    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      await postEvent("submit-attempt");
      notice.classList.add("visible");
    });
  </script>
</body>
</html>
{{end}}
`

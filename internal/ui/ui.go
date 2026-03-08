package ui

import (
	"fmt"
	"strings"

	"github.com/Section9Labs/Cartero/internal/app"
	"github.com/Section9Labs/Cartero/internal/doctor"
	"github.com/Section9Labs/Cartero/internal/plugin"
	"github.com/Section9Labs/Cartero/internal/store"
	"github.com/charmbracelet/lipgloss"
)

type Renderer struct {
	plain       bool
	title       lipgloss.Style
	subtitle    lipgloss.Style
	panel       lipgloss.Style
	label       lipgloss.Style
	pass        lipgloss.Style
	warn        lipgloss.Style
	fail        lipgloss.Style
	muted       lipgloss.Style
	accent      lipgloss.Style
	scoreStrong lipgloss.Style
}

type CommandInfo struct {
	Use   string
	Short string
}

type HelpSpec struct {
	Title          string
	Summary        string
	Details        string
	Usage          string
	Commands       []CommandInfo
	Flags          []string
	InheritedFlags []string
	Examples       []string
}

func NewRenderer(plain bool) Renderer {
	return Renderer{
		plain:       plain,
		title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F4A261")),
		subtitle:    lipgloss.NewStyle().Foreground(lipgloss.Color("#D8E2DC")),
		panel:       lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#3A5A40")).Padding(0, 1),
		label:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#84A59D")),
		pass:        lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#2A9D8F")),
		warn:        lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E9C46A")),
		fail:        lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E76F51")),
		muted:       lipgloss.NewStyle().Foreground(lipgloss.Color("#B7B7A4")),
		accent:      lipgloss.NewStyle().Foreground(lipgloss.Color("#F4A261")),
		scoreStrong: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#264653")),
	}
}

func (r Renderer) Help(spec HelpSpec) string {
	var lines []string
	lines = append(lines, r.banner(spec.Title, spec.Summary))
	if spec.Details != "" {
		lines = append(lines, r.block("About", []string{spec.Details}))
	}

	if spec.Usage != "" {
		lines = append(lines, r.block("Usage", []string{spec.Usage}))
	}

	if len(spec.Commands) > 0 {
		cmdLines := make([]string, 0, len(spec.Commands))
		for _, cmd := range spec.Commands {
			cmdLines = append(cmdLines, fmt.Sprintf("%-16s %s", cmd.Use, cmd.Short))
		}
		lines = append(lines, r.block("Commands", cmdLines))
	}

	if len(spec.Flags) > 0 {
		lines = append(lines, r.block("Flags", spec.Flags))
	}

	if len(spec.InheritedFlags) > 0 {
		lines = append(lines, r.block("Global Flags", spec.InheritedFlags))
	}

	if len(spec.Examples) > 0 {
		lines = append(lines, r.block("Examples", spec.Examples))
	}

	return strings.Join(lines, "\n\n")
}

func (r Renderer) CampaignPreview(c app.Campaign, score int, issues []app.Issue) string {
	overview := []string{
		kv("Owner", c.Metadata.Owner),
		kv("Audience", c.Metadata.Audience),
		kv("Region", c.Metadata.Region),
		kv("Risk", c.Metadata.RiskLevel),
		kv("Deadline", c.Metadata.Deadline),
		kv("Readiness", fmt.Sprintf("%d/100", score)),
	}

	channelLines := make([]string, 0, len(c.Channels))
	for _, channel := range c.Channels {
		channelLines = append(channelLines, fmt.Sprintf("%s | template=%s | goal=%s", channel.Type, channel.Template, channel.Goal))
	}
	if len(channelLines) == 0 {
		channelLines = append(channelLines, "No channels configured")
	}

	stageLines := make([]string, 0, len(c.Timeline))
	for _, stage := range c.Timeline {
		stageLines = append(stageLines, fmt.Sprintf("%s (%s) -> %s", stage.Name, stage.Duration, stage.SuccessMetric))
	}
	if len(stageLines) == 0 {
		stageLines = append(stageLines, "No timeline stages configured")
	}

	safeStatus := []string{
		kv("Dry run", fmt.Sprintf("%t", c.Safeguards.DryRun)),
		kv("Credential capture", fmt.Sprintf("%t", c.Safeguards.CaptureCredentials)),
		kv("External links", fmt.Sprintf("%t", c.Safeguards.AllowExternalLinks)),
		kv("Manager approval", fmt.Sprintf("%t", c.Safeguards.RequireManagerApproval)),
		kv("Approval ticket", c.Safeguards.ApprovalTicket),
	}

	sections := []string{
		r.banner(c.Metadata.Name, "Campaign readiness preview"),
		r.block("Overview", overview),
		r.block("Channels", channelLines),
		r.block("Timeline", stageLines),
		r.block("Safeguards", safeStatus),
		r.issueBlock(issues),
	}

	return strings.Join(sections, "\n\n")
}

func (r Renderer) Validation(issues []app.Issue) string {
	if len(issues) == 0 {
		return r.block("Validation", []string{r.statusText("pass") + " campaign passed all checks"})
	}

	return r.issueBlock(issues)
}

func (r Renderer) DoctorReport(report doctor.Report) string {
	lines := []string{
		kv("Root", report.Root),
		kv("Go", report.GoVersion),
		kv("Platform", report.Platform),
	}

	checkLines := make([]string, 0, len(report.Checks))
	for _, check := range report.Checks {
		line := fmt.Sprintf("%s %s: %s", r.statusText(string(check.Status)), check.Name, check.Detail)
		if check.Hint != "" {
			line += " | hint: " + check.Hint
		}
		checkLines = append(checkLines, line)
	}

	return strings.Join([]string{
		r.banner("Doctor", "Workspace health report"),
		r.block("Environment", lines),
		r.block("Checks", checkLines),
	}, "\n\n")
}

func (r Renderer) Plugins(manifests []plugin.Manifest, warnings []plugin.Warning) string {
	lines := make([]string, 0, len(manifests))
	for _, manifest := range manifests {
		lines = append(lines, fmt.Sprintf(
			"%s %s | kind=%s | mode=%s | trust=%s | safe=%t | capabilities=%s",
			manifest.Name,
			manifest.Version,
			manifest.Kind,
			manifest.Mode,
			manifest.Trust.Level,
			manifest.Safe != nil && *manifest.Safe,
			strings.Join(manifest.Capabilities, ","),
		))
		if manifest.Description != "" {
			lines = append(lines, "  "+manifest.Description)
		}
	}
	if len(lines) == 0 {
		lines = append(lines, "No plugins discovered")
	}

	sections := []string{
		r.banner("Plugins", "Installed local manifests"),
		r.block("Registry", lines),
	}
	if len(warnings) > 0 {
		warningLines := make([]string, 0, len(warnings))
		for _, warning := range warnings {
			warningLines = append(warningLines, fmt.Sprintf("%s %s: %s", r.statusText("warn"), warning.Path, warning.Message))
		}
		sections = append(sections, r.block("Warnings", warningLines))
	}

	return strings.Join(sections, "\n\n")
}

func (r Renderer) Version(version, commit, date string) string {
	return r.block("Build", []string{
		kv("Version", version),
		kv("Commit", commit),
		kv("Date", date),
	})
}

func (r Renderer) Init(path string) string {
	return r.block("Init", []string{
		"starter campaign written",
		kv("Path", path),
	})
}

func (r Renderer) WorkspaceInit(root, database string, manifestCount, templateCount int) string {
	return strings.Join([]string{
		r.banner("Workspace", "Embedded workspace initialized"),
		r.block("State", []string{
			kv("Root", root),
			kv("Database", database),
			kv("Plugin manifests", fmt.Sprintf("%d", manifestCount)),
			kv("Seeded templates", fmt.Sprintf("%d", templateCount)),
		}),
	}, "\n\n")
}

func (r Renderer) WorkspaceStatus(root string, stats store.WorkspaceStats) string {
	return strings.Join([]string{
		r.banner("Workspace", "Embedded workspace status"),
		r.block("State", []string{
			kv("Root", root),
			kv("Database", stats.DatabasePath),
			kv("Templates", fmt.Sprintf("%d", stats.TemplateCount)),
			kv("Audience members", fmt.Sprintf("%d", stats.AudienceCount)),
			kv("Segments", fmt.Sprintf("%d", stats.SegmentCount)),
			kv("Imports", fmt.Sprintf("%d", stats.ImportCount)),
			kv("Campaign snapshots", fmt.Sprintf("%d", stats.CampaignCount)),
			kv("Events", fmt.Sprintf("%d", stats.EventCount)),
			kv("Findings", fmt.Sprintf("%d", stats.FindingCount)),
		}),
	}, "\n\n")
}

func (r Renderer) Templates(templates []store.Template) string {
	lines := make([]string, 0, len(templates))
	for _, template := range templates {
		lines = append(lines, fmt.Sprintf("%s | locale=%s | department=%s | scenario=%s | channels=%s", template.Slug, template.Locale, template.Department, template.Scenario, strings.Join(template.Channels, ",")))
		lines = append(lines, "  "+template.Subject)
	}
	if len(lines) == 0 {
		lines = append(lines, "No templates matched the current filter")
	}

	return strings.Join([]string{
		r.banner("Templates", "Template-library content pack"),
		r.block("Catalog", lines),
	}, "\n\n")
}

func (r Renderer) TemplateDetail(template store.Template) string {
	return strings.Join([]string{
		r.banner(template.Name, "Template detail"),
		r.block("Metadata", []string{
			kv("Slug", template.Slug),
			kv("Locale", template.Locale),
			kv("Department", template.Department),
			kv("Scenario", template.Scenario),
			kv("Channels", strings.Join(template.Channels, ",")),
		}),
		r.block("Content", []string{
			kv("Subject", template.Subject),
			"Body: " + emptyDefault(template.Body),
			"Landing: " + emptyDefault(template.LandingPage),
		}),
	}, "\n\n")
}

func (r Renderer) AudienceImport(result store.AudienceImportResult) string {
	return r.block("Audience Import", []string{
		kv("Segment", result.Segment),
		kv("Created", fmt.Sprintf("%d", result.Created)),
		kv("Updated", fmt.Sprintf("%d", result.Updated)),
	})
}

func (r Renderer) AudienceMembers(members []store.AudienceMember) string {
	lines := make([]string, 0, len(members))
	for _, member := range members {
		lines = append(lines, fmt.Sprintf("%s | segment=%s | department=%s | title=%s", member.Email, member.Segment, emptyDefault(member.Department), emptyDefault(member.Title)))
	}
	if len(lines) == 0 {
		lines = append(lines, "No audience members found")
	}

	return strings.Join([]string{
		r.banner("Audience", "Embedded audience segments"),
		r.block("Members", lines),
	}, "\n\n")
}

func (r Renderer) CloneImport(sourcePath, subject, target string, readiness int) string {
	return strings.Join([]string{
		r.banner("Clone Import", "Reviewed message converted into a campaign draft"),
		r.block("Draft", []string{
			kv("Source", sourcePath),
			kv("Subject", subject),
			kv("Output", target),
			kv("Readiness", fmt.Sprintf("%d/100", readiness)),
		}),
	}, "\n\n")
}

func (r Renderer) ReportExport(path, format string, stats store.WorkspaceStats) string {
	return strings.Join([]string{
		r.banner("Report", "Workspace analytics exported"),
		r.block("Export", []string{
			kv("Path", path),
			kv("Format", format),
			kv("Campaign snapshots", fmt.Sprintf("%d", stats.CampaignCount)),
			kv("Audience members", fmt.Sprintf("%d", stats.AudienceCount)),
			kv("Events", fmt.Sprintf("%d", stats.EventCount)),
			kv("Findings", fmt.Sprintf("%d", stats.FindingCount)),
		}),
	}, "\n\n")
}

func (r Renderer) EventRecorded(event store.Event) string {
	return r.block("Event", []string{
		kv("Campaign", event.CampaignName),
		kv("Audience", event.AudienceEmail),
		kv("Type", event.Type),
		kv("Source", event.Source),
	})
}

func (r Renderer) Events(events []store.Event) string {
	lines := make([]string, 0, len(events))
	for _, event := range events {
		lines = append(lines, fmt.Sprintf("%s | type=%s | email=%s | source=%s", event.CampaignName, event.Type, event.AudienceEmail, event.Source))
	}
	if len(lines) == 0 {
		lines = append(lines, "No engagement events recorded")
	}

	return strings.Join([]string{
		r.banner("Events", "Recorded engagement telemetry"),
		r.block("Telemetry", lines),
	}, "\n\n")
}

func (r Renderer) FindingImport(path, source, tool string, result store.FindingImportResult) string {
	return strings.Join([]string{
		r.banner("Findings", "External findings imported into the workspace"),
		r.block("Import", []string{
			kv("Path", path),
			kv("Source", source),
			kv("Tool", tool),
			kv("Created", fmt.Sprintf("%d", result.Created)),
			kv("Updated", fmt.Sprintf("%d", result.Updated)),
		}),
	}, "\n\n")
}

func (r Renderer) Findings(findings []store.Finding) string {
	lines := make([]string, 0, len(findings))
	for _, finding := range findings {
		lines = append(lines, fmt.Sprintf("%s | tool=%s | severity=%s | target=%s", finding.Rule, finding.Tool, finding.Severity, finding.Target))
		lines = append(lines, "  "+finding.Summary)
	}
	if len(lines) == 0 {
		lines = append(lines, "No findings recorded")
	}

	return strings.Join([]string{
		r.banner("Findings", "Imported scan and analysis signals"),
		r.block("Registry", lines),
	}, "\n\n")
}

func (r Renderer) MigrationReport(path string, report string) string {
	return strings.Join([]string{
		r.banner("Migration", "Legacy data import complete"),
		r.block("Result", []string{
			kv("Path", path),
			report,
		}),
	}, "\n\n")
}

func (r Renderer) banner(title, subtitle string) string {
	lines := []string{strings.ToUpper(title), subtitle}
	if r.plain {
		return strings.Join(lines, "\n")
	}

	return lipgloss.JoinVertical(lipgloss.Left, r.title.Render(lines[0]), r.subtitle.Render(lines[1]))
}

func (r Renderer) block(title string, lines []string) string {
	content := title + "\n" + strings.Join(lines, "\n")
	if r.plain {
		return content
	}

	return r.panel.Render(r.label.Render(title) + "\n" + strings.Join(lines, "\n"))
}

func (r Renderer) issueBlock(issues []app.Issue) string {
	if len(issues) == 0 {
		return r.block("Checks", []string{r.statusText("pass") + " no validation issues"})
	}

	lines := make([]string, 0, len(issues))
	for _, issue := range issues {
		lines = append(lines, fmt.Sprintf("%s %s: %s", r.statusText(string(issue.Severity)), issue.Field, issue.Message))
	}

	return r.block("Checks", lines)
}

func (r Renderer) statusText(status string) string {
	switch status {
	case "pass":
		if r.plain {
			return "[PASS]"
		}
		return r.pass.Render("[PASS]")
	case "warn", "warning":
		if r.plain {
			return "[WARN]"
		}
		return r.warn.Render("[WARN]")
	default:
		if r.plain {
			return "[FAIL]"
		}
		return r.fail.Render("[FAIL]")
	}
}

func kv(key, value string) string {
	return fmt.Sprintf("%s: %s", key, emptyDefault(value))
}

func emptyDefault(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}

	return value
}

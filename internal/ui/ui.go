package ui

import (
	"fmt"
	"strings"

	"github.com/Section9Labs/Cartero/internal/app"
	"github.com/Section9Labs/Cartero/internal/doctor"
	"github.com/Section9Labs/Cartero/internal/plugin"
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

func (r Renderer) Plugins(manifests []plugin.Manifest) string {
	lines := make([]string, 0, len(manifests))
	for _, manifest := range manifests {
		lines = append(lines, fmt.Sprintf("%s %s | kind=%s | mode=%s | safe=%t", manifest.Name, manifest.Version, manifest.Kind, manifest.Mode, manifest.Safe))
		if manifest.Description != "" {
			lines = append(lines, "  "+manifest.Description)
		}
	}
	if len(lines) == 0 {
		lines = append(lines, "No plugins discovered")
	}

	return strings.Join([]string{
		r.banner("Plugins", "Installed local manifests"),
		r.block("Registry", lines),
	}, "\n\n")
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

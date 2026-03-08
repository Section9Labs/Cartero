package app

import (
	"strconv"
	"strings"
	"time"
)

var allowedChannels = map[string]struct{}{
	"chat":     {},
	"email":    {},
	"intranet": {},
	"lms":      {},
	"webinar":  {},
}

func ValidateCampaign(c Campaign) []Issue {
	var issues []Issue

	if strings.TrimSpace(c.Metadata.Name) == "" {
		issues = append(issues, Issue{Severity: SeverityError, Field: "metadata.name", Message: "campaign name is required"})
	}
	if strings.TrimSpace(c.Metadata.Owner) == "" {
		issues = append(issues, Issue{Severity: SeverityError, Field: "metadata.owner", Message: "campaign owner is required"})
	}
	if strings.TrimSpace(c.Metadata.Audience) == "" {
		issues = append(issues, Issue{Severity: SeverityError, Field: "metadata.audience", Message: "target audience is required"})
	}
	if strings.TrimSpace(c.Metadata.RiskLevel) == "" {
		issues = append(issues, Issue{Severity: SeverityError, Field: "metadata.risk_level", Message: "risk level is required"})
	}
	if c.Metadata.Deadline != "" {
		if _, err := time.Parse("2006-01-02", c.Metadata.Deadline); err != nil {
			issues = append(issues, Issue{Severity: SeverityError, Field: "metadata.deadline", Message: "deadline must use YYYY-MM-DD"})
		}
	} else {
		issues = append(issues, Issue{Severity: SeverityWarning, Field: "metadata.deadline", Message: "deadline is missing"})
	}

	if len(c.Objectives) == 0 {
		issues = append(issues, Issue{Severity: SeverityError, Field: "objectives", Message: "at least one objective is required"})
	}

	if len(c.Channels) == 0 {
		issues = append(issues, Issue{Severity: SeverityError, Field: "channels", Message: "at least one delivery channel is required"})
	}

	for i, channel := range c.Channels {
		fieldPrefix := "channels[" + strconv.Itoa(i) + "]"
		if _, ok := allowedChannels[channel.Type]; !ok {
			issues = append(issues, Issue{Severity: SeverityError, Field: fieldPrefix + ".type", Message: "unsupported channel type"})
		}
		if strings.TrimSpace(channel.Template) == "" {
			issues = append(issues, Issue{Severity: SeverityWarning, Field: fieldPrefix + ".template", Message: "template name is missing"})
		}
		if strings.TrimSpace(channel.Goal) == "" {
			issues = append(issues, Issue{Severity: SeverityWarning, Field: fieldPrefix + ".goal", Message: "channel goal is missing"})
		}
	}

	if len(c.Assets) == 0 {
		issues = append(issues, Issue{Severity: SeverityWarning, Field: "assets", Message: "no supporting assets are attached"})
	}

	if !c.Safeguards.DryRun {
		issues = append(issues, Issue{Severity: SeverityWarning, Field: "safeguards.dry_run", Message: "dry_run is disabled"})
	}
	if c.Safeguards.CaptureCredentials {
		issues = append(issues, Issue{Severity: SeverityError, Field: "safeguards.capture_credentials", Message: "credential capture is not allowed"})
	}
	if c.Safeguards.AllowExternalLinks {
		issues = append(issues, Issue{Severity: SeverityError, Field: "safeguards.allow_external_links", Message: "external links are not allowed"})
	}
	if strings.EqualFold(c.Metadata.RiskLevel, "high") && !c.Safeguards.RequireManagerApproval {
		issues = append(issues, Issue{Severity: SeverityError, Field: "safeguards.require_manager_approval", Message: "high-risk campaigns require manager approval"})
	}
	if c.Safeguards.RequireManagerApproval && strings.TrimSpace(c.Safeguards.ApprovalTicket) == "" {
		issues = append(issues, Issue{Severity: SeverityWarning, Field: "safeguards.approval_ticket", Message: "approval ticket is missing"})
	}

	if len(c.Timeline) == 0 {
		issues = append(issues, Issue{Severity: SeverityWarning, Field: "timeline", Message: "no timeline stages defined"})
	}

	for i, stage := range c.Timeline {
		fieldPrefix := "timeline[" + strconv.Itoa(i) + "]"
		if strings.TrimSpace(stage.Name) == "" {
			issues = append(issues, Issue{Severity: SeverityError, Field: fieldPrefix + ".name", Message: "stage name is required"})
		}
		if strings.TrimSpace(stage.SuccessMetric) == "" {
			issues = append(issues, Issue{Severity: SeverityWarning, Field: fieldPrefix + ".success_metric", Message: "success metric is missing"})
		}
		if len(stage.Actions) == 0 {
			issues = append(issues, Issue{Severity: SeverityWarning, Field: fieldPrefix + ".actions", Message: "stage has no actions"})
		}
	}

	return issues
}

func HasErrors(issues []Issue) bool {
	for _, issue := range issues {
		if issue.Severity == SeverityError {
			return true
		}
	}

	return false
}

func ReadinessScore(issues []Issue) int {
	score := 100
	for _, issue := range issues {
		switch issue.Severity {
		case SeverityError:
			score -= 18
		case SeverityWarning:
			score -= 6
		}
	}
	if score < 0 {
		return 0
	}

	return score
}

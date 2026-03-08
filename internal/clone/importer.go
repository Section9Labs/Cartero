package clone

import (
	"bytes"
	"fmt"
	"io"
	"net/mail"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Section9Labs/Cartero/internal/app"
	"gopkg.in/yaml.v3"
)

type Message struct {
	SourcePath string
	Sender     string
	Subject    string
	Body       string
}

var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)

func LoadMessage(path string) (Message, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return Message{}, fmt.Errorf("read source message: %w", err)
	}

	msg := Message{SourcePath: path}
	parsed, err := mail.ReadMessage(bytes.NewReader(payload))
	if err == nil {
		msg.Sender = parsed.Header.Get("From")
		msg.Subject = parsed.Header.Get("Subject")
		body, readErr := io.ReadAll(parsed.Body)
		if readErr != nil {
			return Message{}, fmt.Errorf("read source message body: %w", readErr)
		}
		msg.Body = strings.TrimSpace(string(body))
	} else {
		msg.Body = strings.TrimSpace(string(payload))
		msg.Subject = humanizeSlug(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
		msg.Sender = "unknown"
	}

	if strings.TrimSpace(msg.Subject) == "" {
		msg.Subject = humanizeSlug(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	}
	if strings.TrimSpace(msg.Sender) == "" {
		msg.Sender = "unknown"
	}

	return msg, nil
}

func DraftCampaign(message Message) app.Campaign {
	templateSlug := slugify(message.Subject)
	deadline := time.Now().UTC().Add(30 * 24 * time.Hour).Format("2006-01-02")

	return app.Campaign{
		Metadata: app.Metadata{
			Name:      humanizeSlug(templateSlug),
			Owner:     "Security Operations",
			Audience:  "Imported Segment",
			Region:    "Global",
			RiskLevel: "medium",
			Deadline:  deadline,
		},
		Objectives: []string{
			"Turn a reviewed message into a safe rehearsal draft",
			"Exercise report-and-escalate behavior against a familiar lure",
		},
		Channels: []app.Channel{
			{
				Type:     "email",
				Template: templateSlug,
				Goal:     "Verify suspicious requests through approved contacts before acting",
			},
		},
		Assets: []app.Asset{
			{
				Name:   "source-message",
				Type:   "reviewed-email",
				Source: message.SourcePath,
			},
			{
				Name:   "source-sender",
				Type:   "metadata",
				Source: message.Sender,
			},
		},
		Safeguards: app.Safeguards{
			DryRun:                 true,
			CaptureCredentials:     false,
			AllowExternalLinks:     false,
			RequireManagerApproval: true,
			ApprovalTicket:         "IMPORTED-REVIEW",
		},
		Timeline: []app.Stage{
			{
				Name:          "triage",
				Duration:      "1d",
				SuccessMetric: "Analysts confirm the lure and define audience boundaries",
				Actions: []string{
					"Review the imported source content",
					"Remove unsafe or environment-specific details",
				},
			},
			{
				Name:          "rehearsal",
				Duration:      "2d",
				SuccessMetric: "Pilot group reports the message through the approved workflow",
				Actions: []string{
					"Run the draft against a controlled pilot segment",
					"Validate reporting instructions and landing content",
				},
			},
			{
				Name:          "debrief",
				Duration:      "1d",
				SuccessMetric: "Findings and next steps are documented",
				Actions: []string{
					"Capture lessons learned from the imported scenario",
				},
			},
		},
	}
}

func DraftYAML(campaign app.Campaign) (string, error) {
	payload, err := yaml.Marshal(campaign)
	if err != nil {
		return "", fmt.Errorf("encode imported campaign draft: %w", err)
	}

	return string(payload), nil
}

func OutputFilename(message Message) string {
	return slugify(message.Subject) + ".yaml"
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = slugPattern.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "imported-draft"
	}
	return value
}

func humanizeSlug(value string) string {
	parts := strings.Split(strings.ReplaceAll(value, "_", "-"), "-")
	words := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		words = append(words, capitalize(part))
	}
	if len(words) == 0 {
		return "Imported Draft"
	}
	return strings.Join(words, " ")
}

func capitalize(value string) string {
	if value == "" {
		return value
	}

	return strings.ToUpper(value[:1]) + value[1:]
}

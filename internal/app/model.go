package app

type Campaign struct {
	Metadata   Metadata   `yaml:"metadata"`
	Objectives []string   `yaml:"objectives"`
	Channels   []Channel  `yaml:"channels"`
	Assets     []Asset    `yaml:"assets"`
	Safeguards Safeguards `yaml:"safeguards"`
	Timeline   []Stage    `yaml:"timeline"`
}

type Metadata struct {
	Name      string `yaml:"name"`
	Owner     string `yaml:"owner"`
	Audience  string `yaml:"audience"`
	Region    string `yaml:"region"`
	RiskLevel string `yaml:"risk_level"`
	Deadline  string `yaml:"deadline"`
}

type Channel struct {
	Type     string `yaml:"type"`
	Template string `yaml:"template"`
	Goal     string `yaml:"goal"`
}

type Asset struct {
	Name   string `yaml:"name"`
	Type   string `yaml:"type"`
	Source string `yaml:"source"`
}

type Stage struct {
	Name          string   `yaml:"name"`
	Duration      string   `yaml:"duration"`
	SuccessMetric string   `yaml:"success_metric"`
	Actions       []string `yaml:"actions"`
}

type Safeguards struct {
	DryRun                 bool   `yaml:"dry_run"`
	CaptureCredentials     bool   `yaml:"capture_credentials"`
	AllowExternalLinks     bool   `yaml:"allow_external_links"`
	RequireManagerApproval bool   `yaml:"require_manager_approval"`
	ApprovalTicket         string `yaml:"approval_ticket"`
}

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

type Issue struct {
	Severity Severity
	Field    string
	Message  string
}

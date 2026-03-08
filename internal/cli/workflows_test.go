package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Section9Labs/Cartero/internal/cli"
	"github.com/Section9Labs/Cartero/internal/version"
)

func TestWorkspaceInitSeedsDatabaseAndBuiltins(t *testing.T) {
	var out bytes.Buffer
	root := t.TempDir()

	cmd := cli.NewRootCmd(cli.IOStreams{Out: &out, Err: &out}, version.BuildInfo())
	cmd.SetArgs([]string{"--plain", "--root", root, "workspace", "init"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v\noutput=%s", err, out.String())
	}

	if _, err := os.Stat(filepath.Join(root, ".cartero", "cartero.db")); err != nil {
		t.Fatalf("expected database to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "plugins", "template-library.yaml")); err != nil {
		t.Fatalf("expected builtin manifest to exist: %v", err)
	}
	if !strings.Contains(out.String(), "Seeded templates: 5") {
		t.Fatalf("expected template seed output, got %s", out.String())
	}
}

func TestAudienceImportAndList(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "audiences", "finance.csv"), ""+
		"email,display_name,department,title\n"+
		"analyst@example.com,Finance Analyst,Finance,Analyst\n"+
		"manager@example.com,Finance Manager,Finance,Manager\n")

	var out bytes.Buffer
	cmd := cli.NewRootCmd(cli.IOStreams{Out: &out, Err: &out}, version.BuildInfo())
	cmd.SetArgs([]string{"--plain", "--root", root, "audience", "import", "--segment", "finance-emea", "--csv", filepath.Join("audiences", "finance.csv")})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v\noutput=%s", err, out.String())
	}
	if !strings.Contains(out.String(), "Created: 2") {
		t.Fatalf("expected created count, got %s", out.String())
	}

	out.Reset()
	cmd = cli.NewRootCmd(cli.IOStreams{Out: &out, Err: &out}, version.BuildInfo())
	cmd.SetArgs([]string{"--plain", "--root", root, "audience", "list", "--segment", "finance-emea"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v\noutput=%s", err, out.String())
	}
	if !strings.Contains(out.String(), "analyst@example.com") || !strings.Contains(out.String(), "manager@example.com") {
		t.Fatalf("expected audience members in output, got %s", out.String())
	}
}

func TestCloneImportAndReportExport(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "samples", "reported.eml"), ""+
		"From: alerts@example.com\n"+
		"Subject: Review payroll account changes\n"+
		"\n"+
		"Please review the attached account changes.\n")

	var out bytes.Buffer
	cmd := cli.NewRootCmd(cli.IOStreams{Out: &out, Err: &out}, version.BuildInfo())
	cmd.SetArgs([]string{"--plain", "--root", root, "import", "clone", "-f", filepath.Join("samples", "reported.eml")})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v\noutput=%s", err, out.String())
	}
	if !strings.Contains(out.String(), "review-payroll-account-changes.yaml") {
		t.Fatalf("expected generated draft in output, got %s", out.String())
	}
	if _, err := os.Stat(filepath.Join(root, "drafts", "review-payroll-account-changes.yaml")); err != nil {
		t.Fatalf("expected generated draft file: %v", err)
	}

	out.Reset()
	cmd = cli.NewRootCmd(cli.IOStreams{Out: &out, Err: &out}, version.BuildInfo())
	cmd.SetArgs([]string{"--plain", "--root", root, "report", "export", "--format", "json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v\noutput=%s", err, out.String())
	}
	if !strings.Contains(out.String(), "workspace-report.json") {
		t.Fatalf("expected report path in output, got %s", out.String())
	}
	payload, err := os.ReadFile(filepath.Join(root, "exports", "workspace-report.json"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(payload), "\"campaigns\"") || !strings.Contains(string(payload), "\"imports\"") {
		t.Fatalf("expected campaigns and imports in report export, got %s", payload)
	}
}

func TestEventRecordAndList(t *testing.T) {
	root := t.TempDir()
	var out bytes.Buffer

	cmd := cli.NewRootCmd(cli.IOStreams{Out: &out, Err: &out}, version.BuildInfo())
	cmd.SetArgs([]string{"--plain", "--root", root, "event", "record", "--campaign", "Quarterly Drill", "--email", "analyst@example.com", "--type", "reported"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v\noutput=%s", err, out.String())
	}
	if !strings.Contains(out.String(), "Type: reported") {
		t.Fatalf("expected recorded event output, got %s", out.String())
	}

	out.Reset()
	cmd = cli.NewRootCmd(cli.IOStreams{Out: &out, Err: &out}, version.BuildInfo())
	cmd.SetArgs([]string{"--plain", "--root", root, "event", "list"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v\noutput=%s", err, out.String())
	}
	if !strings.Contains(out.String(), "Quarterly Drill") || !strings.Contains(out.String(), "reported") {
		t.Fatalf("expected event output, got %s", out.String())
	}
}

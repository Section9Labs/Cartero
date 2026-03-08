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

func TestValidateCommandPlainOutput(t *testing.T) {
	var out bytes.Buffer
	cmd := cli.NewRootCmd(cli.IOStreams{
		Out: &out,
		Err: &out,
	}, version.BuildInfo())
	cmd.SetArgs([]string{"--plain", "validate", "-f", filepath.Join("..", "app", "testdata", "valid.yaml")})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(out.String(), "campaign passed all checks") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestPreviewHelpIncludesFileFlag(t *testing.T) {
	var out bytes.Buffer
	cmd := cli.NewRootCmd(cli.IOStreams{
		Out: &out,
		Err: &out,
	}, version.BuildInfo())
	cmd.SetArgs([]string{"--plain", "preview", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	rendered := out.String()
	if !strings.Contains(rendered, "--file string") {
		t.Fatalf("expected file flag in help output, got %s", rendered)
	}
	if !strings.Contains(rendered, "--plain") {
		t.Fatalf("expected inherited flag in help output, got %s", rendered)
	}
}

func TestPreviewCommandFailsOnInvalidCampaign(t *testing.T) {
	var out bytes.Buffer
	cmd := cli.NewRootCmd(cli.IOStreams{
		Out: &out,
		Err: &out,
	}, version.BuildInfo())
	cmd.SetArgs([]string{"--plain", "preview", "-f", filepath.Join("..", "app", "testdata", "invalid.yaml")})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected preview command to fail")
	}

	if !strings.Contains(out.String(), "Credential capture") && !strings.Contains(strings.ToLower(out.String()), "credential capture") {
		t.Fatalf("expected safeguards output, got %s", out.String())
	}
}

func TestDoctorResolvesRootFromSubdirectory(t *testing.T) {
	var out bytes.Buffer
	cmd := cli.NewRootCmd(cli.IOStreams{
		Out: &out,
		Err: &out,
	}, version.BuildInfo())

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(cwd, "..", ".."))
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(filepath.Join(cwd, "..")); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalWD); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})

	cmd.SetArgs([]string{"--plain", "doctor"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v\noutput=%s", err, out.String())
	}

	if !strings.Contains(out.String(), "Root: "+repoRoot) {
		t.Fatalf("expected resolved project root in output, got %s", out.String())
	}
}

func TestPluginListResolvesExplicitRoot(t *testing.T) {
	var out bytes.Buffer
	cmd := cli.NewRootCmd(cli.IOStreams{
		Out: &out,
		Err: &out,
	}, version.BuildInfo())

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(cwd, "..", ".."))
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalWD); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})

	cmd.SetArgs([]string{"--plain", "--root", repoRoot, "plugin", "list"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v\noutput=%s", err, out.String())
	}

	if !strings.Contains(out.String(), "local-preview") {
		t.Fatalf("expected plugin manifest in output, got %s", out.String())
	}
}

func TestPluginListShowsWarningsWithoutFailing(t *testing.T) {
	var out bytes.Buffer
	cmd := cli.NewRootCmd(cli.IOStreams{
		Out: &out,
		Err: &out,
	}, version.BuildInfo())

	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "plugins", "nested", "valid.yml"), ""+
		"schema_version: v1\n"+
		"name: nested-preview\n"+
		"version: 1.0.0\n"+
		"kind: renderer\n"+
		"mode: local-only\n"+
		"safe: true\n"+
		"capabilities:\n"+
		"  - preview.render\n"+
		"trust:\n"+
		"  level: reviewed\n"+
		"  review_required: false\n"+
		"description: Nested manifest\n")
	mustWriteFile(t, filepath.Join(root, "plugins", "broken.yaml"), "name: [\n")

	cmd.SetArgs([]string{"--plain", "--root", root, "plugin", "list"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v\noutput=%s", err, out.String())
	}

	rendered := out.String()
	if !strings.Contains(rendered, "nested-preview") {
		t.Fatalf("expected valid plugin in output, got %s", rendered)
	}
	if !strings.Contains(rendered, "Warnings") || !strings.Contains(rendered, "broken.yaml") {
		t.Fatalf("expected warning output, got %s", rendered)
	}
}

func TestDoctorWarnsOnPluginDiscoveryIssues(t *testing.T) {
	var out bytes.Buffer
	cmd := cli.NewRootCmd(cli.IOStreams{
		Out: &out,
		Err: &out,
	}, version.BuildInfo())

	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "configs", "campaign.example.yaml"), "metadata:\n  name: sample\n")
	mustWriteFile(t, filepath.Join(root, ".goreleaser.yaml"), "project_name: cartero\n")
	mustWriteFile(t, filepath.Join(root, "scripts", "smoke.sh"), "#!/usr/bin/env bash\n")
	mustWriteFile(t, filepath.Join(root, "plugins", "valid.yaml"), ""+
		"schema_version: v1\n"+
		"name: valid\n"+
		"version: 1.0.0\n"+
		"kind: renderer\n"+
		"mode: local-only\n"+
		"safe: true\n"+
		"capabilities:\n"+
		"  - preview.render\n"+
		"trust:\n"+
		"  level: reviewed\n"+
		"  review_required: false\n"+
		"description: Valid plugin\n")
	mustWriteFile(t, filepath.Join(root, "plugins", "broken.yml"), "name: [\n")

	cmd.SetArgs([]string{"--plain", "--root", root, "doctor"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v\noutput=%s", err, out.String())
	}

	if !strings.Contains(out.String(), "[WARN] Plugin manifests: 1 manifest, 1 warning") {
		t.Fatalf("expected warning summary, got %s", out.String())
	}
}

func mustWriteFile(t *testing.T, path, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

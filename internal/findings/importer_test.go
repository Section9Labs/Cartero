package findings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadJSONLinesNuclei(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "nuclei.jsonl")
	if err := os.WriteFile(path, []byte(""+
		"{\"template-id\":\"open-redirect\",\"host\":\"https://example.com\",\"matcher-name\":\"redirect\",\"info\":{\"name\":\"Open Redirect\",\"severity\":\"medium\",\"description\":\"Redirect parameter can be abused\"}}\n"+
		"{\"template-id\":\"missing-headers\",\"host\":\"https://example.com/login\",\"info\":{\"name\":\"Missing Headers\",\"severity\":\"low\"}}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	findings, err := Load(path, "scan-import", "")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %+v", findings)
	}
	if findings[0].Tool != "nuclei" || findings[0].Rule != "open-redirect" {
		t.Fatalf("unexpected first finding: %+v", findings[0])
	}
	if findings[0].Severity != "medium" || findings[1].Severity != "low" {
		t.Fatalf("unexpected severities: %+v", findings)
	}
}

func TestLoadCSVGenericFindings(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "findings.csv")
	if err := os.WriteFile(path, []byte(""+
		"tool,rule,severity,target,summary,team\n"+
		"nessus,weak-cipher,high,https://vpn.example.com,Legacy cipher suite enabled,infra\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	findings, err := Load(path, "nightly-scan", "")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %+v", findings)
	}
	if findings[0].Source != "nightly-scan" || findings[0].Metadata["team"] != "infra" {
		t.Fatalf("unexpected finding: %+v", findings[0])
	}
}

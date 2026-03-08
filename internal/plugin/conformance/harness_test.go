package conformance_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Section9Labs/Cartero/internal/plugin/conformance"
)

func TestPluginListFixtureMatchesGolden(t *testing.T) {
	root := filepath.Join("testdata", "workspaces", "local-preview")

	result, err := conformance.PluginList(root)
	if err != nil {
		t.Fatalf("PluginList() error = %v", err)
	}

	if len(result.Discovery.Manifests) != 1 {
		t.Fatalf("expected 1 manifest, got %d", len(result.Discovery.Manifests))
	}
	if len(result.Discovery.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %+v", result.Discovery.Warnings)
	}

	golden := mustReadFile(t, filepath.Join(root, "plugin-list.plain.golden"))
	if result.Output != conformance.NormalizeOutput(golden) {
		t.Fatalf("unexpected plugin list output:\nwant:\n%s\n\ngot:\n%s", conformance.NormalizeOutput(golden), result.Output)
	}
}

func TestPluginListHandlesEmptyWorkspace(t *testing.T) {
	result, err := conformance.PluginList(t.TempDir())
	if err != nil {
		t.Fatalf("PluginList() error = %v", err)
	}

	if len(result.Discovery.Manifests) != 0 || len(result.Discovery.Warnings) != 0 {
		t.Fatalf("expected empty discovery, got %+v", result.Discovery)
	}
	if result.Output == "" {
		t.Fatal("expected rendered output for empty workspace")
	}
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()

	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	return string(payload)
}

package plugin_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Section9Labs/Cartero/internal/plugin"
)

func TestDiscoverSupportsNestedManifestsAndWarnings(t *testing.T) {
	root := t.TempDir()

	mustWriteFile(t, filepath.Join(root, "plugins", "alpha.yaml"), ""+
		"schema_version: v1\n"+
		"name: alpha\n"+
		"version: 1.0.0\n"+
		"kind: renderer\n"+
		"mode: local-only\n"+
		"safe: true\n"+
		"capabilities:\n"+
		"  - preview.render\n"+
		"trust:\n"+
		"  level: reviewed\n"+
		"  review_required: false\n"+
		"description: Alpha plugin\n")
	mustWriteFile(t, filepath.Join(root, "plugins", "nested", "beta.yml"), ""+
		"schema_version: v1\n"+
		"name: beta\n"+
		"version: 1.1.0\n"+
		"kind: exporter\n"+
		"mode: local-only\n"+
		"safe: true\n"+
		"capabilities:\n"+
		"  - results.export\n"+
		"trust:\n"+
		"  level: reviewed\n"+
		"  review_required: false\n"+
		"description: Beta plugin\n")
	mustWriteFile(t, filepath.Join(root, "plugins", "broken.yaml"), "name: [\n")

	discovery, err := plugin.Discover(filepath.Join(root, "plugins"))
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(discovery.Manifests) != 2 {
		t.Fatalf("expected 2 manifests, got %d", len(discovery.Manifests))
	}
	if discovery.Manifests[0].Name != "alpha" || discovery.Manifests[1].Name != "beta" {
		t.Fatalf("unexpected manifest order: %+v", discovery.Manifests)
	}
	if len(discovery.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(discovery.Warnings))
	}
	if filepath.Base(discovery.Warnings[0].Path) != "broken.yaml" {
		t.Fatalf("unexpected warning path: %+v", discovery.Warnings[0])
	}
}

func TestDiscoverMissingDirectory(t *testing.T) {
	discovery, err := plugin.Discover(filepath.Join(t.TempDir(), "plugins"))
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(discovery.Manifests) != 0 || len(discovery.Warnings) != 0 {
		t.Fatalf("expected empty discovery, got %+v", discovery)
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

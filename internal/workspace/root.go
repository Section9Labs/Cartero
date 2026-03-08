package workspace

import (
	"fmt"
	"os"
	"path/filepath"
)

var rootMarkers = []string{
	filepath.Join(".cartero", "cartero.sqlite"),
	filepath.Join(".cartero", "cartero.db"),
	"go.mod",
	".goreleaser.yaml",
	filepath.Join("configs", "campaign.example.yaml"),
	"plugins",
}

func ResolveRoot(cwd, explicit string) (string, error) {
	if explicit != "" {
		abs, err := filepath.Abs(explicit)
		if err != nil {
			return "", fmt.Errorf("resolve root path: %w", err)
		}
		if _, err := os.Stat(abs); err != nil {
			return "", fmt.Errorf("resolve root path: %w", err)
		}

		return abs, nil
	}

	absStart, err := filepath.Abs(cwd)
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}

	dir := absStart
	for {
		if looksLikeRoot(dir) {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return absStart, nil
		}
		dir = parent
	}
}

func ResolveInputPath(root, path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	if exists(path) {
		return path
	}

	candidate := filepath.Join(root, path)
	if exists(candidate) {
		return candidate
	}

	return path
}

func looksLikeRoot(dir string) bool {
	for _, marker := range rootMarkers {
		if exists(filepath.Join(dir, marker)) {
			return true
		}
	}

	return false
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

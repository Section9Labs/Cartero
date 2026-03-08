package plugin

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

type Manifest struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Kind        string `yaml:"kind"`
	Mode        string `yaml:"mode"`
	Safe        bool   `yaml:"safe"`
	Description string `yaml:"description"`
	Path        string `yaml:"-"`
}

func Discover(dir string) ([]Manifest, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("discover plugins: %w", err)
	}

	manifests := make([]Manifest, 0, len(matches))
	for _, match := range matches {
		payload, err := os.ReadFile(match)
		if err != nil {
			return nil, fmt.Errorf("read plugin manifest %s: %w", match, err)
		}

		var manifest Manifest
		decoder := yaml.NewDecoder(bytes.NewReader(payload))
		decoder.KnownFields(true)
		if err := decoder.Decode(&manifest); err != nil {
			return nil, fmt.Errorf("decode plugin manifest %s: %w", match, err)
		}

		manifest.Path = match
		manifests = append(manifests, manifest)
	}

	sort.Slice(manifests, func(i, j int) bool {
		return manifests[i].Name < manifests[j].Name
	})

	return manifests, nil
}

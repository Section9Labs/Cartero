package plugin

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

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

type Warning struct {
	Path    string
	Message string
}

type Discovery struct {
	Manifests []Manifest
	Warnings  []Warning
}

func Discover(dir string) (Discovery, error) {
	var discovery Discovery

	info, err := os.Stat(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return discovery, nil
		}
		return discovery, fmt.Errorf("discover plugins: %w", err)
	}
	if !info.IsDir() {
		return discovery, fmt.Errorf("discover plugins: %s is not a directory", dir)
	}

	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			discovery.Warnings = append(discovery.Warnings, Warning{
				Path:    path,
				Message: walkErr.Error(),
			})
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() || !isManifestPath(path) {
			return nil
		}

		manifest, err := loadManifest(path)
		if err != nil {
			discovery.Warnings = append(discovery.Warnings, Warning{
				Path:    path,
				Message: err.Error(),
			})
			return nil
		}

		discovery.Manifests = append(discovery.Manifests, manifest)
		return nil
	})
	if err != nil {
		return discovery, fmt.Errorf("discover plugins: %w", err)
	}

	sort.Slice(discovery.Manifests, func(i, j int) bool {
		if discovery.Manifests[i].Name == discovery.Manifests[j].Name {
			return discovery.Manifests[i].Path < discovery.Manifests[j].Path
		}
		return discovery.Manifests[i].Name < discovery.Manifests[j].Name
	})
	sort.Slice(discovery.Warnings, func(i, j int) bool {
		if discovery.Warnings[i].Path == discovery.Warnings[j].Path {
			return discovery.Warnings[i].Message < discovery.Warnings[j].Message
		}
		return discovery.Warnings[i].Path < discovery.Warnings[j].Path
	})

	return discovery, nil
}

func isManifestPath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func loadManifest(path string) (Manifest, error) {
	var manifest Manifest

	payload, err := os.ReadFile(path)
	if err != nil {
		return manifest, fmt.Errorf("read manifest: %w", err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(payload))
	decoder.KnownFields(true)
	if err := decoder.Decode(&manifest); err != nil {
		return manifest, fmt.Errorf("decode manifest: %w", err)
	}

	manifest.Path = path
	return manifest, nil
}

package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Section9Labs/Cartero/internal/plugin"
)

type Status string

const (
	StatusPass Status = "pass"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
)

type Check struct {
	Name   string
	Status Status
	Detail string
	Hint   string
}

type Report struct {
	Root      string
	GoVersion string
	Platform  string
	Checks    []Check
}

func Run(root string) Report {
	report := Report{
		Root:      root,
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}

	goPath, err := exec.LookPath("go")
	if err != nil {
		report.Checks = append(report.Checks, Check{
			Name:   "Go toolchain",
			Status: StatusFail,
			Detail: "go is not available on PATH",
			Hint:   "install Go 1.22+ before building Cartero",
		})
	} else {
		report.Checks = append(report.Checks, Check{
			Name:   "Go toolchain",
			Status: StatusPass,
			Detail: goPath,
		})
	}

	report.Checks = append(report.Checks, fileCheck(
		"Sample config",
		filepath.Join(root, "configs", "campaign.example.yaml"),
		"run `cartero init` to generate a local campaign file",
	))
	report.Checks = append(report.Checks, fileCheck(
		"Release config",
		filepath.Join(root, ".goreleaser.yaml"),
		"add GoReleaser config for packaged releases",
	))
	report.Checks = append(report.Checks, fileCheck(
		"Smoke test script",
		filepath.Join(root, "scripts", "smoke.sh"),
		"add a reproducible smoke test to verify the CLI end-to-end",
	))

	discovery, pluginErr := plugin.Discover(filepath.Join(root, "plugins"))
	if pluginErr != nil {
		report.Checks = append(report.Checks, Check{
			Name:   "Plugin manifests",
			Status: StatusFail,
			Detail: pluginErr.Error(),
			Hint:   "fix malformed files in plugins/",
		})
	} else if len(discovery.Manifests) == 0 && len(discovery.Warnings) == 0 {
		report.Checks = append(report.Checks, Check{
			Name:   "Plugin manifests",
			Status: StatusWarn,
			Detail: "no plugins discovered",
			Hint:   "drop one or more YAML manifests into plugins/",
		})
	} else if len(discovery.Warnings) > 0 {
		report.Checks = append(report.Checks, Check{
			Name:   "Plugin manifests",
			Status: StatusWarn,
			Detail: formatDiscoveryDetail(discovery),
			Hint:   "fix malformed files in plugins/ without blocking healthy manifests",
		})
	} else {
		report.Checks = append(report.Checks, Check{
			Name:   "Plugin manifests",
			Status: StatusPass,
			Detail: formatDiscoveryDetail(discovery),
		})
	}

	return report
}

func HasFailures(report Report) bool {
	for _, check := range report.Checks {
		if check.Status == StatusFail {
			return true
		}
	}

	return false
}

func fileCheck(name, path, hint string) Check {
	if _, err := os.Stat(path); err != nil {
		return Check{
			Name:   name,
			Status: StatusFail,
			Detail: path + " is missing",
			Hint:   hint,
		}
	}

	return Check{
		Name:   name,
		Status: StatusPass,
		Detail: path,
	}
}

func pluralize(count int, noun string) string {
	if count == 1 {
		return "1 " + noun
	}

	return fmt.Sprintf("%d %ss", count, noun)
}

func formatDiscoveryDetail(discovery plugin.Discovery) string {
	parts := []string{pluralize(len(discovery.Manifests), "manifest")}
	if len(discovery.Warnings) > 0 {
		parts = append(parts, pluralize(len(discovery.Warnings), "warning"))
	}

	return strings.Join(parts, ", ")
}

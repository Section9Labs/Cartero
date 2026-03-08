package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Section9Labs/Cartero/internal/app"
	"github.com/Section9Labs/Cartero/internal/catalog"
	"github.com/Section9Labs/Cartero/internal/store"
)

func prepareWorkspaceStore(root string) (*store.Store, error) {
	s, err := store.Open(root)
	if err != nil {
		return nil, err
	}

	if _, err := catalog.SyncManifests(root); err != nil {
		_ = s.Close()
		return nil, err
	}

	stats, err := s.Stats()
	if err != nil {
		_ = s.Close()
		return nil, err
	}
	if stats.TemplateCount == 0 {
		if _, err := catalog.SeedTemplateLibrary(s); err != nil {
			_ = s.Close()
			return nil, err
		}
	}

	return s, nil
}

func persistCampaignSnapshot(root, sourcePath string, campaign app.Campaign, score int, issues []app.Issue, source string) error {
	s, err := prepareWorkspaceStore(root)
	if err != nil {
		return fmt.Errorf("prepare workspace store: %w", err)
	}
	defer s.Close()

	absPath := sourcePath
	if !filepath.IsAbs(absPath) {
		if resolved, resolveErr := filepath.Abs(sourcePath); resolveErr == nil {
			absPath = resolved
		}
	}

	_, err = s.SaveCampaignSnapshot(store.CampaignSnapshot{
		Name:       campaign.Metadata.Name,
		SourcePath: absPath,
		Owner:      campaign.Metadata.Owner,
		Audience:   campaign.Metadata.Audience,
		Region:     campaign.Metadata.Region,
		RiskLevel:  campaign.Metadata.RiskLevel,
		Readiness:  score,
		IssueCount: len(issues),
		Issues:     issueStrings(issues),
		Source:     source,
	})
	if err != nil {
		return fmt.Errorf("persist campaign snapshot: %w", err)
	}

	return nil
}

func resolveOutputPath(root, requested, defaultDir, defaultFile string) string {
	target := requested
	if target == "" {
		target = filepath.Join(defaultDir, defaultFile)
	}
	if filepath.IsAbs(target) {
		return target
	}
	if filepath.Dir(target) == "." {
		return filepath.Join(root, target)
	}
	return filepath.Join(root, target)
}

func ensureParentDir(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent directory for %s: %w", path, err)
	}

	return nil
}

func issueStrings(issues []app.Issue) []string {
	lines := make([]string, 0, len(issues))
	for _, issue := range issues {
		lines = append(lines, string(issue.Severity)+": "+issue.Field+" "+issue.Message)
	}

	return lines
}

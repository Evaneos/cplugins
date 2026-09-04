package cache

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Evaneos/cplugins/internal/claude"
)

// FindOrphans walks cacheDir and returns paths that are not referenced.
// Entries starting with "temp_git_" are always considered orphans.
// For the marketplace/plugin/version directory structure, any version directory
// not present in referencedPaths is an orphan.
// The claude-plugins-official marketplace directory is always skipped.
func FindOrphans(cacheDir string, referencedPaths map[string]bool) ([]string, error) {
	var orphans []string

	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		entryPath := filepath.Join(cacheDir, entry.Name())

		if strings.HasPrefix(entry.Name(), "temp_git_") {
			orphans = append(orphans, entryPath)
			continue
		}

		if !entry.IsDir() || entry.Name() == claude.OfficialMarketplace {
			continue
		}

		// Walk marketplace/plugin/version
		versionOrphans, err := findVersionOrphans(entryPath, referencedPaths)
		if err != nil {
			return nil, err
		}
		orphans = append(orphans, versionOrphans...)
	}

	return orphans, nil
}

func findVersionOrphans(marketplaceDir string, referencedPaths map[string]bool) ([]string, error) {
	var orphans []string

	plugins, err := os.ReadDir(marketplaceDir)
	if err != nil {
		return nil, err
	}

	for _, plugin := range plugins {
		if !plugin.IsDir() {
			continue
		}

		pluginDir := filepath.Join(marketplaceDir, plugin.Name())
		versions, err := os.ReadDir(pluginDir)
		if err != nil {
			return nil, err
		}

		for _, version := range versions {
			if !version.IsDir() {
				continue
			}

			versionPath := filepath.Join(pluginDir, version.Name())
			if !referencedPaths[versionPath] {
				orphans = append(orphans, versionPath)
			}
		}
	}

	return orphans, nil
}

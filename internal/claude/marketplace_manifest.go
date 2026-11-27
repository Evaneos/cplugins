package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type marketplaceManifest struct {
	Metadata struct {
		PluginRoot string `json:"pluginRoot"`
	} `json:"metadata"`
	Plugins []struct {
		Name   string          `json:"name"`
		Source json.RawMessage `json:"source"`
	} `json:"plugins"`
}

// PluginSourceDir returns the directory holding plugin name inside the
// marketplace tree at root, as declared by the "source" of its entry in
// <root>/.claude-plugin/marketplace.json. Only relative-path sources resolve
// locally; an object source (github, git-subdir, url…) is an error.
func PluginSourceDir(root, name string) (string, error) {
	path := filepath.Join(root, ".claude-plugin", "marketplace.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading marketplace manifest: %w", err)
	}
	var m marketplaceManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return "", fmt.Errorf("parsing marketplace manifest: %w", err)
	}

	for _, p := range m.Plugins {
		if p.Name != name {
			continue
		}
		var source string
		if err := json.Unmarshal(p.Source, &source); err != nil {
			return "", fmt.Errorf("plugin %q is sourced outside the marketplace tree; run `claude plugin install` or `claude plugin update` instead", name)
		}
		rel := source
		if !strings.HasPrefix(source, "./") {
			// Claude Code resolves a source without "./" only as a bare name under metadata.pluginRoot.
			if source == "" || strings.Contains(source, "/") || m.Metadata.PluginRoot == "" {
				return "", fmt.Errorf("plugin %q has an invalid source %q in %s", name, source, path)
			}
			rel = filepath.Join(m.Metadata.PluginRoot, source)
		}
		if !filepath.IsLocal(filepath.Clean(rel)) {
			return "", fmt.Errorf("plugin %q source %q escapes the marketplace root", name, source)
		}
		return filepath.Join(root, rel), nil
	}
	return "", fmt.Errorf("plugin %q not listed in %s", name, path)
}

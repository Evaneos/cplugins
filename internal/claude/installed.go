package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Scope values used by Claude Code in installed_plugins.json.
const (
	ScopeUser    = "user"
	ScopeProject = "project"
)

// OfficialMarketplace is the marketplace name Claude Code uses for the
// plugins it manages itself.
const OfficialMarketplace = "claude-plugins-official"

// Install represents a single install entry for a plugin (one per scope).
type Install struct {
	Scope       string `json:"scope"`
	ProjectPath string `json:"projectPath"`
	InstallPath string `json:"installPath"`
	Version     string `json:"version"`
}

// Plugin represents a Claude Code plugin with its install entries.
type Plugin struct {
	Name        string
	Marketplace string
	Key         string
	Installs    []Install
}

type installedFile struct {
	Version int                  `json:"version"`
	Plugins map[string][]Install `json:"plugins"`
}

// ParseInstalledPlugins reads and parses Claude Code's installed_plugins.json.
func ParseInstalledPlugins(path string) (map[string]*Plugin, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading installed plugins: %w", err)
	}

	var file installedFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parsing installed plugins: %w", err)
	}

	plugins := make(map[string]*Plugin, len(file.Plugins))
	for key, installs := range file.Plugins {
		name, marketplace := splitPluginKey(key)
		plugins[key] = &Plugin{
			Name:        name,
			Marketplace: marketplace,
			Key:         key,
			Installs:    installs,
		}
	}

	return plugins, nil
}

// splitPluginKey splits "plugin-name@marketplace" into (name, marketplace).
func splitPluginKey(key string) (string, string) {
	i := strings.LastIndex(key, "@")
	if i < 0 {
		return key, ""
	}
	return key[:i], key[i+1:]
}

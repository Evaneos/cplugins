package claude

import (
	"encoding/json"
	"os"
)

type rawSettings struct {
	EnabledPlugins map[string]bool `json:"enabledPlugins"`
}

// ReadEnabledPlugins reads the "enabledPlugins" map from a settings.json
// file. A missing file, an unreadable file, or invalid JSON all yield a nil
// map — the caller treats an absent key as enabled, so this never fails the
// caller's command.
func ReadEnabledPlugins(path string) map[string]bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var s rawSettings
	if err := json.Unmarshal(data, &s); err != nil {
		return nil
	}
	return s.EnabledPlugins
}

// MergeEnabledPlugins merges the "enabledPlugins" maps read from paths, in
// order: for a given key, the last path that mentions it wins. This mirrors
// Claude Code's settings chain (user, user local, project, project local).
func MergeEnabledPlugins(paths ...string) map[string]bool {
	merged := make(map[string]bool)
	for _, path := range paths {
		for key, enabled := range ReadEnabledPlugins(path) {
			merged[key] = enabled
		}
	}
	return merged
}

// IsPluginEnabled reports whether key is enabled per a merged enabledPlugins
// map, defaulting to enabled when key is absent — Claude Code's behavior for
// plugins installed without going through /plugin.
func IsPluginEnabled(key string, merged map[string]bool) bool {
	enabled, ok := merged[key]
	if !ok {
		return true
	}
	return enabled
}

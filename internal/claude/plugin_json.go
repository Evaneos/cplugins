package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type pluginJSON struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ReadPluginName reads the plugin name from <dir>/.claude-plugin/plugin.json.
func ReadPluginName(dir string) (string, error) {
	path := filepath.Join(dir, ".claude-plugin", "plugin.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading plugin.json: %w", err)
	}
	var pj pluginJSON
	if err := json.Unmarshal(data, &pj); err != nil {
		return "", fmt.Errorf("parsing plugin.json: %w", err)
	}
	if pj.Name == "" {
		return "", fmt.Errorf("plugin.json missing \"name\" field")
	}
	return pj.Name, nil
}

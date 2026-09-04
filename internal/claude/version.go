package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func ResolveSourceVersion(pluginDir string) (string, error) {
	path := filepath.Join(pluginDir, ".claude-plugin", "plugin.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading plugin.json: %w", err)
	}
	var p pluginJSON
	if err := json.Unmarshal(data, &p); err != nil {
		return "", fmt.Errorf("parsing plugin.json: %w", err)
	}
	return p.Version, nil
}

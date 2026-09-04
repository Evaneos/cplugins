package e2e_test

// claude_config_dir_test.go covers CLAUDE_CONFIG_DIR: claudeDir() must read
// from that directory when set, and fall back to ~/.claude when it is unset
// or set to an empty value.

import (
	"path/filepath"
	"testing"
)

func TestClaudeConfigDir_overridesRegistry(t *testing.T) {
	home := setupHome(t) // ordinary, empty $HOME/.claude registry
	altConfig := t.TempDir()
	if err := copyDir("claude-home", altConfig, nil); err != nil {
		t.Fatalf("copyDir: %v", err)
	}

	mpDir := t.TempDir()
	installDir := filepath.Join(mpDir, "plugins", "hello")
	writePluginJSON(t, installDir, "hello", "0.1.0")
	writeJSON(t, filepath.Join(altConfig, "plugins", "known_marketplaces.json"), map[string]any{
		"alt-mp": map[string]any{
			"source":          map[string]any{"source": "directory", "path": mpDir},
			"installLocation": mpDir,
			"autoUpdate":      false,
		},
	})
	writeJSON(t, filepath.Join(altConfig, "plugins", "installed_plugins.json"), map[string]any{
		"version": 2,
		"plugins": map[string]any{
			"hello@alt-mp": []map[string]any{
				{"scope": "user", "projectPath": "", "installPath": installDir, "version": "0.1.0"},
			},
		},
	})

	env := envWithHomeAndConfigDir(home, altConfig)
	out := mustCpluginsWithEnv(t, env, "status")
	assertContains(t, out, "hello")
	assertContains(t, out, "alt-mp")

	// Sanity: the plugin only exists under altConfig, not under $HOME/.claude.
	out2 := mustCplugins(t, home, "status")
	assertNotContains(t, out2, "hello")
}

func TestClaudeConfigDir_emptyBehavesAsAbsent(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	env := envWithHomeAndConfigDir(home, "")
	out := mustCpluginsWithEnv(t, env, "status")
	assertContains(t, out, "hello")
}

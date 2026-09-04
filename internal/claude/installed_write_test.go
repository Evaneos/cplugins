package claude_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Evaneos/cplugins/internal/claude"
)

func writeInstalledPlugins(t *testing.T, dir string, content map[string]any) string {
	t.Helper()
	path := filepath.Join(dir, "installed_plugins.json")
	data, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readInstalledPlugins(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestPatchInstalls(t *testing.T) {
	dir := t.TempDir()
	path := writeInstalledPlugins(t, dir, map[string]any{
		"version": 1,
		"plugins": map[string]any{
			"hello@test-mp": []any{
				map[string]any{"scope": "user", "projectPath": "", "installPath": "/old/path", "version": "0.1.0"},
			},
		},
	})

	err := claude.PatchInstalls(path, "hello@test-mp", func(scope, currentPath, currentVersion string) (string, string) {
		return "/new/cache/path", "0.2.0"
	})
	if err != nil {
		t.Fatalf("PatchInstalls failed: %v", err)
	}

	data := readInstalledPlugins(t, path)
	plugins, _ := data["plugins"].(map[string]any)
	entries, _ := plugins["hello@test-mp"].([]any)
	inst, _ := entries[0].(map[string]any)
	if inst["installPath"] != "/new/cache/path" {
		t.Errorf("installPath = %q, want /new/cache/path", inst["installPath"])
	}
	if inst["version"] != "0.2.0" {
		t.Errorf("version = %q, want 0.2.0", inst["version"])
	}
}

func TestPatchInstalls_preservesExistingFileMode(t *testing.T) {
	dir := t.TempDir()
	path := writeInstalledPlugins(t, dir, map[string]any{
		"version": 1,
		"plugins": map[string]any{
			"hello@test-mp": []any{
				map[string]any{"scope": "user", "projectPath": "", "installPath": "/old/path", "version": "0.1.0"},
			},
		},
	})
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}

	err := claude.PatchInstalls(path, "hello@test-mp", func(scope, currentPath, currentVersion string) (string, string) {
		return "/new/cache/path", "0.2.0"
	})
	if err != nil {
		t.Fatalf("PatchInstalls failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("file mode = %o, want 0600 (hardened mode preserved)", got)
	}
}

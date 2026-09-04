package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writePluginJSON(t *testing.T, dir, name, version string) {
	t.Helper()
	pjDir := filepath.Join(dir, ".claude-plugin")
	if err := os.MkdirAll(pjDir, 0o755); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(map[string]string{"name": name, "version": version})
	if err := os.WriteFile(filepath.Join(pjDir, "plugin.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRecachePlugin(t *testing.T) {
	mpDir := t.TempDir()
	pluginSrc := filepath.Join(mpDir, "plugins", "hello")
	writePluginJSON(t, pluginSrc, "hello", "0.2.0")
	os.WriteFile(filepath.Join(pluginSrc, "README.md"), []byte("hello"), 0o644)

	cacheDir := t.TempDir()

	cachePath, version, err := RecachePlugin(filepath.Join(mpDir, "plugins"), "hello", cacheDir, "test-mp")
	if err != nil {
		t.Fatalf("RecachePlugin failed: %v", err)
	}
	if version != "0.2.0" {
		t.Errorf("version = %q, want %q", version, "0.2.0")
	}
	expectedPath := filepath.Join(cacheDir, "test-mp", "hello", "0.2.0")
	if cachePath != expectedPath {
		t.Errorf("cachePath = %q, want %q", cachePath, expectedPath)
	}
	if _, err := os.Stat(filepath.Join(cachePath, ".claude-plugin", "plugin.json")); err != nil {
		t.Errorf("plugin.json not copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cachePath, "README.md")); err != nil {
		t.Errorf("README.md not copied: %v", err)
	}
}

func TestRecachePlugin_overwritesExisting(t *testing.T) {
	mpDir := t.TempDir()
	pluginSrc := filepath.Join(mpDir, "plugins", "hello")
	writePluginJSON(t, pluginSrc, "hello", "0.1.0")

	cacheDir := t.TempDir()

	RecachePlugin(filepath.Join(mpDir, "plugins"), "hello", cacheDir, "test-mp")
	writePluginJSON(t, pluginSrc, "hello", "0.2.0")
	cachePath, version, err := RecachePlugin(filepath.Join(mpDir, "plugins"), "hello", cacheDir, "test-mp")
	if err != nil {
		t.Fatalf("RecachePlugin failed on overwrite: %v", err)
	}
	if version != "0.2.0" {
		t.Errorf("version = %q, want %q", version, "0.2.0")
	}
	expectedPath := filepath.Join(cacheDir, "test-mp", "hello", "0.2.0")
	if cachePath != expectedPath {
		t.Errorf("cachePath = %q, want %q", cachePath, expectedPath)
	}
}

func TestRecachePlugin_missingSource(t *testing.T) {
	cacheDir := t.TempDir()
	_, _, err := RecachePlugin("/nonexistent/plugins", "hello", cacheDir, "test-mp")
	if err == nil {
		t.Error("expected error for missing source")
	}
}

// TestRecachePlugin_symlinkedSource covers the case where a directory-style
// marketplace contains symlinks instead of real plugin directories (a common
// pattern for aggregating per-repo plugins under one marketplace).
func TestRecachePlugin_symlinkedSource(t *testing.T) {
	realParent := t.TempDir()
	pluginReal := filepath.Join(realParent, "hello-real")
	writePluginJSON(t, pluginReal, "hello", "0.3.0")
	if err := os.WriteFile(filepath.Join(pluginReal, "README.md"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(pluginReal, "skills", "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginReal, "skills", "demo", "SKILL.md"), []byte("# demo"), 0o644); err != nil {
		t.Fatal(err)
	}

	mpDir := t.TempDir()
	pluginsDir := filepath.Join(mpDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(pluginReal, filepath.Join(pluginsDir, "hello")); err != nil {
		t.Fatal(err)
	}

	cacheDir := t.TempDir()
	cachePath, version, err := RecachePlugin(pluginsDir, "hello", cacheDir, "test-mp")
	if err != nil {
		t.Fatalf("RecachePlugin failed on symlinked source: %v", err)
	}
	if version != "0.3.0" {
		t.Errorf("version = %q, want %q", version, "0.3.0")
	}
	for _, rel := range []string{".claude-plugin/plugin.json", "README.md", "skills/demo/SKILL.md"} {
		if _, err := os.Stat(filepath.Join(cachePath, rel)); err != nil {
			t.Errorf("%s not copied: %v", rel, err)
		}
	}
}

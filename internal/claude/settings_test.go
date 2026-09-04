package claude

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestSettings(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestReadEnabledPlugins(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(dir, "settings.json")
	writeTestSettings(t, path, `{"enabledPlugins": {"hello@mp": false, "other@mp": true}}`)
	got := ReadEnabledPlugins(path)
	if got["hello@mp"] != false || got["other@mp"] != true {
		t.Errorf("ReadEnabledPlugins() = %v", got)
	}
}

func TestReadEnabledPlugins_missingFile(t *testing.T) {
	got := ReadEnabledPlugins(filepath.Join(t.TempDir(), "nope.json"))
	if got != nil {
		t.Errorf("ReadEnabledPlugins() on missing file = %v, want nil", got)
	}
}

func TestReadEnabledPlugins_invalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	writeTestSettings(t, path, `{not json`)
	got := ReadEnabledPlugins(path)
	if got != nil {
		t.Errorf("ReadEnabledPlugins() on invalid JSON = %v, want nil", got)
	}
}

func TestReadEnabledPlugins_noEnabledPluginsKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	writeTestSettings(t, path, `{}`)
	got := ReadEnabledPlugins(path)
	if got != nil {
		t.Errorf("ReadEnabledPlugins() = %v, want nil", got)
	}
}

func TestMergeEnabledPlugins_laterFileWins(t *testing.T) {
	dir := t.TempDir()
	user := filepath.Join(dir, "settings.json")
	project := filepath.Join(dir, "project", "settings.json")
	writeTestSettings(t, user, `{"enabledPlugins": {"hello@mp": false}}`)
	writeTestSettings(t, project, `{"enabledPlugins": {"hello@mp": true}}`)

	merged := MergeEnabledPlugins(user, project)
	if !merged["hello@mp"] {
		t.Errorf("merged[hello@mp] = false, want true (project settings win)")
	}
}

func TestMergeEnabledPlugins_missingFilesIgnored(t *testing.T) {
	dir := t.TempDir()
	merged := MergeEnabledPlugins(filepath.Join(dir, "nope1.json"), filepath.Join(dir, "nope2.json"))
	if len(merged) != 0 {
		t.Errorf("merged = %v, want empty", merged)
	}
}

func TestIsPluginEnabled(t *testing.T) {
	merged := map[string]bool{"hello@mp": false}

	if IsPluginEnabled("hello@mp", merged) {
		t.Error("IsPluginEnabled(hello@mp) = true, want false")
	}
	if !IsPluginEnabled("absent@mp", merged) {
		t.Error("IsPluginEnabled(absent@mp) = false, want true (default enabled)")
	}
	if !IsPluginEnabled("anything", nil) {
		t.Error("IsPluginEnabled with nil map = false, want true")
	}
}

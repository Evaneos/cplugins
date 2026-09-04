package claude

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveSourceVersion(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, ".claude-plugin")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(`{"version":"2.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	version, err := ResolveSourceVersion(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version != "2.0.0" {
		t.Errorf("got %q, want %q", version, "2.0.0")
	}
}

func TestResolveSourceVersion_NoFile(t *testing.T) {
	_, err := ResolveSourceVersion("/nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent path, got nil")
	}
}

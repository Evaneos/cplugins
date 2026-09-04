package claude_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Evaneos/cplugins/internal/claude"
)

func TestReadPluginName(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, ".claude-plugin", "plugin.json")
	if err := os.MkdirAll(filepath.Dir(jsonPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jsonPath, []byte(`{"name":"my-plugin","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	name, err := claude.ReadPluginName(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "my-plugin" {
		t.Errorf("got %q, want %q", name, "my-plugin")
	}
}

func TestReadPluginName_Missing(t *testing.T) {
	_, err := claude.ReadPluginName(t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing plugin.json")
	}
}

func TestReadPluginName_NoName(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, ".claude-plugin", "plugin.json")
	if err := os.MkdirAll(filepath.Dir(jsonPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jsonPath, []byte(`{"version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := claude.ReadPluginName(dir)
	if err == nil {
		t.Fatal("expected error for missing name field")
	}
}

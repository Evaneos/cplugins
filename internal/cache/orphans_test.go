package cache

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestFindOrphans(t *testing.T) {
	dir := t.TempDir()

	// marketplace/plugin/version structure
	referenced := filepath.Join(dir, "my-market", "good-plugin", "1.0.0")
	orphanVersion := filepath.Join(dir, "my-market", "good-plugin", "0.9.0")
	tempGit := filepath.Join(dir, "temp_git_abc123")

	for _, d := range []string{referenced, orphanVersion, tempGit} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	referencedPaths := map[string]bool{
		referenced: true,
	}

	orphans, err := FindOrphans(dir, referencedPaths)
	if err != nil {
		t.Fatalf("FindOrphans: %v", err)
	}

	sort.Strings(orphans)
	if len(orphans) != 2 {
		t.Fatalf("expected 2 orphans, got %d: %v", len(orphans), orphans)
	}

	// Sort to have deterministic order
	expected := []string{orphanVersion, tempGit}
	sort.Strings(expected)

	for i, want := range expected {
		if orphans[i] != want {
			t.Errorf("orphans[%d] = %q, want %q", i, orphans[i], want)
		}
	}
}

func TestFindOrphans_EmptyCache(t *testing.T) {
	dir := t.TempDir()
	orphans, err := FindOrphans(dir, map[string]bool{})
	if err != nil {
		t.Fatalf("FindOrphans: %v", err)
	}
	if len(orphans) != 0 {
		t.Errorf("expected 0 orphans, got %d: %v", len(orphans), orphans)
	}
}

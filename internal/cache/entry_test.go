package cache

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveEntry(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "toremove")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "file.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := RemoveEntry(target); err != nil {
		t.Fatalf("RemoveEntry: %v", err)
	}

	if _, err := os.Stat(target); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected target to be gone, got err: %v", err)
	}
}

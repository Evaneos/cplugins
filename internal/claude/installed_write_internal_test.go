package claude

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteRegistry_newFileGetsDefaultMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "installed_plugins.json")
	root, err := decodeOrderedObject([]byte(`{"version":1}`))
	if err != nil {
		t.Fatal(err)
	}

	if err := writeRegistry(path, root); err != nil {
		t.Fatalf("writeRegistry: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Errorf("file mode = %o, want 0644 for a newly created file", got)
	}
}

func TestWriteRegistry_preservesExistingFileMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "installed_plugins.json")
	if err := os.WriteFile(path, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	root, err := decodeOrderedObject([]byte(`{"version":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := writeRegistry(path, root); err != nil {
		t.Fatalf("writeRegistry: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("file mode = %o, want 0600 (hardened mode preserved)", got)
	}
}

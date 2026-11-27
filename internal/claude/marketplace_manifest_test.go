package claude

import (
	"os"
	"path/filepath"
	"testing"
)

func writeManifest(t *testing.T, root, content string) {
	t.Helper()
	dir := filepath.Join(root, ".claude-plugin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "marketplace.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPluginSourceDir(t *testing.T) {
	cases := []struct {
		name     string
		manifest string
		want     string // relative to root; "" means an error is expected
	}{
		{
			name:     "repository root",
			manifest: `{"plugins":[{"name":"p","source":"./"}]}`,
			want:     ".",
		},
		{
			name:     "plugins subdirectory",
			manifest: `{"plugins":[{"name":"p","source":"./plugins/p"}]}`,
			want:     "plugins/p",
		},
		{
			name:     "directory at the root",
			manifest: `{"plugins":[{"name":"other","source":"./other"},{"name":"p","source":"./p-dir"}]}`,
			want:     "p-dir",
		},
		{
			name:     "bare name under pluginRoot",
			manifest: `{"metadata":{"pluginRoot":"./plugins"},"plugins":[{"name":"p","source":"p"}]}`,
			want:     "plugins/p",
		},
		{
			name:     "pluginRoot ignored for a ./ source",
			manifest: `{"metadata":{"pluginRoot":"./plugins"},"plugins":[{"name":"p","source":"./p"}]}`,
			want:     "p",
		},
		{
			name:     "path escaping the root",
			manifest: `{"plugins":[{"name":"p","source":"./../elsewhere"}]}`,
		},
		{
			name:     "object source",
			manifest: `{"plugins":[{"name":"p","source":{"source":"github","repo":"org/p"}}]}`,
		},
		{
			name:     "empty source",
			manifest: `{"plugins":[{"name":"p","source":""}]}`,
		},
		{
			name:     "bare name without pluginRoot",
			manifest: `{"plugins":[{"name":"p","source":"p"}]}`,
		},
		{
			name:     "nested path without ./",
			manifest: `{"metadata":{"pluginRoot":"./plugins"},"plugins":[{"name":"p","source":"team/p"}]}`,
		},
		{
			name:     "pluginRoot escaping the root",
			manifest: `{"metadata":{"pluginRoot":"../outside"},"plugins":[{"name":"p","source":"p"}]}`,
		},
		{
			name:     "plugin not listed",
			manifest: `{"plugins":[{"name":"other","source":"./"}]}`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			writeManifest(t, root, c.manifest)

			got, err := PluginSourceDir(root, "p")
			if c.want == "" {
				if err == nil {
					t.Fatalf("PluginSourceDir() = %q, want an error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("PluginSourceDir() error = %v", err)
			}
			if want := filepath.Join(root, c.want); got != want {
				t.Errorf("PluginSourceDir() = %q, want %q", got, want)
			}
		})
	}
}

func TestPluginSourceDir_missingManifest(t *testing.T) {
	if got, err := PluginSourceDir(t.TempDir(), "p"); err == nil {
		t.Fatalf("PluginSourceDir() = %q, want an error", got)
	}
}

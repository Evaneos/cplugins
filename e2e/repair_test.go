package e2e_test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRepair_brokenPlugin(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	ip := installPath(t, home, "hello@test-mp")
	os.RemoveAll(ip)

	out := mustCplugins(t, home, "status")
	assertContains(t, out, "broken")

	out = mustCplugins(t, home, "repair")
	assertContains(t, out, "hello@test-mp")
	assertContains(t, out, "→")

	out = mustCplugins(t, home, "status")
	assertContains(t, out, "ok")
	assertNotContains(t, out, "broken")
}

func TestRepair_specificPlugin(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	ip := installPath(t, home, "hello@test-mp")
	os.RemoveAll(ip)

	out := mustCplugins(t, home, "repair", "hello@test-mp")
	assertContains(t, out, "→")
}

func TestRepair_nothingToRepair(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	out := mustCplugins(t, home, "repair")
	assertContains(t, out, "Nothing to repair")
}

func TestRepair_unknownPlugin(t *testing.T) {
	home := setupHome(t)
	_, _, err := runCplugins(t, home, "repair", "ghost@nowhere")
	if err == nil {
		t.Error("expected error for unknown plugin")
	}
}

func TestRepair_updatesVersion(t *testing.T) {
	home := setupHome(t)
	mpDir := newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	ip := installPath(t, home, "hello@test-mp")
	os.RemoveAll(ip)

	bumpPluginSourceVersion(t, filepath.Join(mpDir, "plugins", "hello"), "0.2.0")

	mustCplugins(t, home, "repair")

	entry := installedPluginEntry(t, home, "hello@test-mp")
	if v, _ := entry["version"].(string); v != "0.2.0" {
		t.Errorf("version = %q, want 0.2.0", v)
	}
}

// TestRepair_symlinkedMarketplaceSource simulates the "local marketplace
// aggregating via symlinks to the real plugin repos" pattern: the plugin under
// plugins/<name> in the marketplace is a symlink to a directory elsewhere.
// repair must follow the symlink when copying into the cache.
func TestRepair_symlinkedMarketplaceSource(t *testing.T) {
	home := setupHome(t)

	// Real plugin source directory, outside the marketplace.
	pluginReal := filepath.Join(t.TempDir(), "hello-source")
	writePluginJSON(t, pluginReal, "hello", "0.1.0")
	if err := os.WriteFile(filepath.Join(pluginReal, "README.md"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Marketplace whose plugins/hello entry is a symlink to pluginReal.
	mpDir := t.TempDir()
	pluginsSubdir := filepath.Join(mpDir, "plugins")
	if err := os.MkdirAll(pluginsSubdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(pluginReal, filepath.Join(pluginsSubdir, "hello")); err != nil {
		t.Fatal(err)
	}
	writeJSON(t, filepath.Join(mpDir, ".claude-plugin", "marketplace.json"), map[string]any{
		"name":  "test-mp",
		"owner": map[string]any{"name": "test"},
		"plugins": []map[string]any{{
			"name":        "hello",
			"source":      "./plugins/hello",
			"description": "test plugin hello",
			"version":     "0.1.0",
			"author":      map[string]any{"name": "test"},
			"tags":        []string{},
		}},
	})

	mustClaudeOK(t, home, "plugins", "marketplace", "add", mpDir)
	installPlugin(t, home, "hello@test-mp")

	ip := installPath(t, home, "hello@test-mp")
	os.RemoveAll(ip)

	out := mustCplugins(t, home, "repair")
	assertContains(t, out, "hello@test-mp")
	assertContains(t, out, "→")
	assertNotContains(t, out, "is a directory")
}

// TestRepair_skipsBuildArtifacts verifies that repair does not copy over
// rebuildable artifact directories (.git, .venv, __pycache__, node_modules) while
// preserving the plugin's actual content. The artifacts are added after the install
// to isolate the test to repair's copy step.
func TestRepair_skipsBuildArtifacts(t *testing.T) {
	home := setupHome(t)
	mpDir := newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	pluginSrc := filepath.Join(mpDir, "plugins", "hello")
	for _, dir := range []string{".git", ".venv", "__pycache__", "node_modules"} {
		d := filepath.Join(pluginSrc, dir)
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "junk"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(pluginSrc, "keep.md"), []byte("real"), 0o644); err != nil {
		t.Fatal(err)
	}

	ip := installPath(t, home, "hello@test-mp")
	os.RemoveAll(ip)

	mustCplugins(t, home, "repair")

	cache := installPath(t, home, "hello@test-mp")
	assertFileExists(t, filepath.Join(cache, "keep.md"))
	for _, dir := range []string{".git", ".venv", "__pycache__", "node_modules"} {
		assertFileNotExists(t, filepath.Join(cache, dir))
	}
}

// TestRepair_internalSymlink ensures that repair recreates as-is a symlink
// internal to the plugin source (e.g. .venv/lib64 -> lib in a virtualenv) and that
// the copied link stays a working link, to a directory as well as to a file.
// The links are added to the source after the install to isolate the test to
// repair's copy step (claude plugin install never sees them).
func TestRepair_internalSymlink(t *testing.T) {
	home := setupHome(t)
	mpDir := newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	// Source: a real directory, a link-to-directory (like lib64 -> lib in a
	// venv) and a link-to-file, all relative and internal to the source.
	pluginSrc := filepath.Join(mpDir, "plugins", "hello")
	realDir := filepath.Join(pluginSrc, "vendor", "real")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realDir, "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real", filepath.Join(pluginSrc, "vendor", "dirlink")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real/f.txt", filepath.Join(pluginSrc, "vendor", "filelink")); err != nil {
		t.Fatal(err)
	}

	ip := installPath(t, home, "hello@test-mp")
	os.RemoveAll(ip)

	out := mustCplugins(t, home, "repair")
	assertContains(t, out, "→")
	assertNotContains(t, out, "is a directory")

	// The links must be recreated as links, and resolve to their
	// target: reading content through each of them.
	vendor := filepath.Join(installPath(t, home, "hello@test-mp"), "vendor")
	for _, link := range []string{"dirlink", "filelink"} {
		p := filepath.Join(vendor, link)
		fi, err := os.Lstat(p)
		if err != nil {
			t.Fatalf("%s missing from cache: %v", link, err)
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			t.Errorf("%s copied as non-symlink (mode %v)", link, fi.Mode())
		}
	}
	got, err := os.ReadFile(filepath.Join(vendor, "dirlink", "f.txt"))
	if err != nil || string(got) != "x" {
		t.Errorf("target unreadable via dirlink: got %q err %v", got, err)
	}
	got, err = os.ReadFile(filepath.Join(vendor, "filelink"))
	if err != nil || string(got) != "x" {
		t.Errorf("target unreadable via filelink: got %q err %v", got, err)
	}
}

func TestRepair_gitMarketplace(t *testing.T) {
	home := setupHome(t)
	injectGitMarketplace(t, home, "github-mp", "github", "org/plugin-repo", []pluginSpec{{"hello", "1.0.0"}})

	cachePath := cacheVersionDir(home, "github-mp", "hello", "1.0.0")
	addInstallEntry(t, home, "hello@github-mp", map[string]any{
		"scope": "user", "projectPath": "",
		"installPath": cachePath, "version": "1.0.0",
	})

	out := mustCplugins(t, home, "repair", "hello@github-mp")
	assertContains(t, out, "hello@github-mp")
	assertContains(t, out, "→")

	pj := filepath.Join(cachePath, ".claude-plugin", "plugin.json")
	if _, err := os.Stat(pj); err != nil {
		t.Errorf("plugin.json missing at %s after repair: %v", pj, err)
	}
}

func TestRepair_afterUndev(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	ip := installPath(t, home, "hello@test-mp")
	os.RemoveAll(ip)

	out := mustCplugins(t, home, "repair")
	assertContains(t, out, "→")

	newIP := installPath(t, home, "hello@test-mp")
	assertUnderCache(t, home, newIP)
}

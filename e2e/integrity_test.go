package e2e_test

// Integrity tests: cplugins must never break Claude Code.
//
// Invariants verified here:
//   1. claude-plugins-official cache dirs are never removed
//   2. known_marketplaces.json is never written by cplugins directly
//   3. installed_plugins.json is only patched by dev/undev (installPath only); clean never writes it
//   4. Marketplace source directories are never modified
//   5. installed_plugins.json stays valid JSON after every operation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// --- 1. claude-plugins-official is never touched ------------------------------

func TestIntegrity_cleanPreservesOfficialMarketplaceCache(t *testing.T) {
	home := setupHome(t)
	officialDir := filepath.Join(home, ".claude", "plugins", "cache", "claude-plugins-official", "hookify", "unknown")
	if err := os.MkdirAll(officialDir, 0o755); err != nil {
		t.Fatal(err)
	}

	mustCplugins(t, home, "clean")

	assertFileExists(t, officialDir)
}

func TestIntegrity_cleanPreservesAllOfficialPlugins(t *testing.T) {
	home := setupHome(t)
	plugins := []string{"hookify", "code-review", "context7", "commit-commands", "pr-review-toolkit"}
	for _, p := range plugins {
		dir := filepath.Join(home, ".claude", "plugins", "cache", "claude-plugins-official", p, "unknown")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	mustCplugins(t, home, "clean")

	for _, p := range plugins {
		assertFileExists(t, filepath.Join(home, ".claude", "plugins", "cache", "claude-plugins-official", p, "unknown"))
	}
}

// --- 2. known_marketplaces.json is never written by cplugins -----------------

func marketplacesFilePath(home string) string {
	return filepath.Join(home, ".claude", "plugins", "known_marketplaces.json")
}

func TestIntegrity_cleanDoesNotWriteKnownMarketplaces(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	stat := statFile(t, marketplacesFilePath(home))
	mustCplugins(t, home, "clean")
	assertFileUnchanged(t, marketplacesFilePath(home), stat, "clean wrote known_marketplaces.json")
}

func TestIntegrity_devDoesNotWriteKnownMarketplaces(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	src := newPluginSource(t, "hello", "0.1.0")

	stat := statFile(t, marketplacesFilePath(home))
	mustCplugins(t, home, "dev", "hello@test-mp", src)
	assertFileUnchanged(t, marketplacesFilePath(home), stat, "dev wrote known_marketplaces.json")
}

func TestIntegrity_undevDoesNotWriteKnownMarketplaces(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	src := newPluginSource(t, "hello", "0.1.0")
	mustCplugins(t, home, "dev", "hello@test-mp", src)

	stat := statFile(t, marketplacesFilePath(home))
	mustCplugins(t, home, "undev", "hello@test-mp")
	assertFileUnchanged(t, marketplacesFilePath(home), stat, "undev wrote known_marketplaces.json")
}

// --- 3. installed_plugins.json is never written by clean ----------------------

func TestIntegrity_cleanDoesNotWriteInstalledPlugins(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	stat := statFile(t, installedPluginsPath(home))
	mustCplugins(t, home, "clean")
	assertFileUnchanged(t, installedPluginsPath(home), stat, "clean wrote installed_plugins.json")
}

func TestIntegrity_devPatchesInstallPath(t *testing.T) {
	// dev patches installPath in installed_plugins.json to point to the source dir
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	origIP := installPath(t, home, "hello@test-mp")
	src := newPluginSource(t, "hello", "0.1.0")

	mustCplugins(t, home, "dev", "hello@test-mp", src)

	ip := installPath(t, home, "hello@test-mp")
	if ip == origIP {
		t.Error("dev should have patched installPath in installed_plugins.json")
	}
}

// --- 4. Marketplace source directories are never modified --------------------

func TestIntegrity_cleanDoesNotModifyMarketplaceSource(t *testing.T) {
	home := setupHome(t)
	mpDir := newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	snapBefore := dirSnapshot(t, mpDir)
	mustCplugins(t, home, "clean")
	snapAfter := dirSnapshot(t, mpDir)

	if snapBefore != snapAfter {
		t.Error("clean modified marketplace source directory")
	}
}

func TestIntegrity_devDoesNotModifyMarketplaceSource(t *testing.T) {
	home := setupHome(t)
	mpDir := newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	src := newPluginSource(t, "hello", "0.1.0")

	snapBefore := dirSnapshot(t, mpDir)
	mustCplugins(t, home, "dev", "hello@test-mp", src)
	snapAfter := dirSnapshot(t, mpDir)

	if snapBefore != snapAfter {
		t.Error("dev modified marketplace source directory")
	}
}

// --- 6. installed_plugins.json stays valid JSON after every operation --------

func TestIntegrity_installedPluginsRemainsValidAfterDev(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	src := newPluginSource(t, "hello", "0.1.0")

	mustCplugins(t, home, "dev", "hello@test-mp", src)
	assertValidJSON(t, installedPluginsPath(home))
}

func TestIntegrity_installedPluginsRemainsValidAfterUndev(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	src := newPluginSource(t, "hello", "0.1.0")
	mustCplugins(t, home, "dev", "hello@test-mp", src)

	mustCplugins(t, home, "undev", "hello@test-mp")
	assertValidJSON(t, installedPluginsPath(home))
}

func TestIntegrity_installedPluginsRemainsValidAfterClean(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	mustCplugins(t, home, "clean")
	assertValidJSON(t, installedPluginsPath(home))
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func statFile(t *testing.T, path string) os.FileInfo {
	t.Helper()
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("statFile %s: %v", path, err)
	}
	return fi
}

func assertFileUnchanged(t *testing.T, path string, before os.FileInfo, msg string) {
	t.Helper()
	after, err := os.Stat(path)
	if err != nil {
		t.Fatalf("assertFileUnchanged stat %s: %v", path, err)
	}
	if before.ModTime() != after.ModTime() {
		t.Errorf("%s: %s", msg, path)
	}
}

// dirSnapshot returns a string summarising file names and modtimes under dir.
func dirSnapshot(t *testing.T, dir string) string {
	t.Helper()
	var snap string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		snap += rel + ":" + info.ModTime().String() + "\n"
		return nil
	})
	if err != nil {
		t.Fatalf("dirSnapshot %s: %v", dir, err)
	}
	return snap
}

func assertValidJSON(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("assertValidJSON: cannot read %s: %v", path, err)
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Errorf("assertValidJSON: %s is not valid JSON: %v\ncontent:\n%s", path, err, data)
	}
}

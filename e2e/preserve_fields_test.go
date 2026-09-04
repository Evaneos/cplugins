package e2e_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// extraFields mimics the fields Claude Code writes into an install entry that
// cplugins does not know about (lastUpdated, installedAt, gitCommitSha).
var extraFields = map[string]any{
	"lastUpdated":  "2026-01-01T00:00:00.000Z",
	"installedAt":  "2025-12-01T00:00:00.000Z",
	"gitCommitSha": "abc123def456",
}

// setEntryFields merges fields into the first install entry for key.
func setEntryFields(t *testing.T, home, key string, fields map[string]any) {
	t.Helper()
	var data map[string]any
	readJSON(t, installedPluginsPath(home), &data)
	plugins, _ := data["plugins"].(map[string]any)
	entries, _ := plugins[key].([]any)
	if len(entries) == 0 {
		t.Fatalf("no entries for %s", key)
	}
	entry, _ := entries[0].(map[string]any)
	for k, v := range fields {
		entry[k] = v
	}
	entries[0] = entry
	plugins[key] = entries
	data["plugins"] = plugins
	writeJSON(t, installedPluginsPath(home), data)
}

// setRootField adds a top-level key to installed_plugins.json, alongside "version" and "plugins".
func setRootField(t *testing.T, home, key string, value any) {
	t.Helper()
	var data map[string]any
	readJSON(t, installedPluginsPath(home), &data)
	data[key] = value
	writeJSON(t, installedPluginsPath(home), data)
}

// assertExtraFieldsIntact checks that entry still carries extraFields unchanged.
func assertExtraFieldsIntact(t *testing.T, entry map[string]any) {
	t.Helper()
	for k, want := range extraFields {
		if got := entry[k]; got != want {
			t.Errorf("%s = %v, want %v (preserved)", k, got, want)
		}
	}
}

// assertNoTempFiles checks that dir holds no residual installed_plugins.json temp file.
func assertNoTempFiles(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading dir %s: %v", dir, err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".installed_plugins-") {
			t.Errorf("leftover temp file: %s", e.Name())
		}
	}
}

// --- 1. Preservation on the modified entry ----------------------------------

func TestPreserveFields_ModifiedEntry(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	setEntryFields(t, home, "hello@test-mp", extraFields)
	src := newPluginSource(t, "hello", "0.1.0")

	mustCplugins(t, home, "dev", "hello@test-mp", src)

	entry := installedPluginEntry(t, home, "hello@test-mp")
	assertExtraFieldsIntact(t, entry)
	resolvedSrc, _ := filepath.EvalSymlinks(src)
	if entry["installPath"] != resolvedSrc {
		t.Errorf("installPath = %v, want %v", entry["installPath"], resolvedSrc)
	}
}

// --- 2. Collateral preservation, per writer ---------------------------------

func TestPreserveFields_CollateralPrune(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}, {"other", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	installPlugin(t, home, "other@test-mp")
	setEntryFields(t, home, "other@test-mp", extraFields)
	addInstallEntry(t, home, "hello@test-mp", deadProjectEntry(t, "/nonexistent/install/path", "0.1.0"))

	mustCplugins(t, home, "prune")

	entry := installedPluginEntry(t, home, "other@test-mp")
	assertExtraFieldsIntact(t, entry)
}

func TestPreserveFields_CollateralDev(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}, {"other", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	installPlugin(t, home, "other@test-mp")
	setEntryFields(t, home, "other@test-mp", extraFields)
	src := newPluginSource(t, "hello", "0.1.0")

	mustCplugins(t, home, "dev", "hello@test-mp", src)

	entry := installedPluginEntry(t, home, "other@test-mp")
	assertExtraFieldsIntact(t, entry)
}

func TestPreserveFields_CollateralUndev(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}, {"other", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	installPlugin(t, home, "other@test-mp")
	src := newPluginSource(t, "hello", "0.1.0")
	mustCplugins(t, home, "dev", "hello@test-mp", src)
	setEntryFields(t, home, "other@test-mp", extraFields)

	mustCplugins(t, home, "undev", "hello@test-mp")

	entry := installedPluginEntry(t, home, "other@test-mp")
	assertExtraFieldsIntact(t, entry)
}

func TestPreserveFields_CollateralRepair(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}, {"other", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	installPlugin(t, home, "other@test-mp")
	setEntryFields(t, home, "other@test-mp", extraFields)

	ip := installPath(t, home, "hello@test-mp")
	os.RemoveAll(ip)

	mustCplugins(t, home, "repair", "hello@test-mp")

	entry := installedPluginEntry(t, home, "other@test-mp")
	assertExtraFieldsIntact(t, entry)
}

// --- 3. Unknown root key -----------------------------------------------------

func TestPreserveFields_UnknownRootKey(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	setRootField(t, home, "foo", float64(1))
	src := newPluginSource(t, "hello", "0.1.0")

	mustCplugins(t, home, "dev", "hello@test-mp", src)

	var data map[string]any
	readJSON(t, installedPluginsPath(home), &data)
	if data["foo"] != float64(1) {
		t.Errorf(`root key "foo" = %v, want 1 (preserved)`, data["foo"])
	}
}

// --- 4. Atomicity: no residual temp file -------------------------------------

func TestPreserveFields_NoLeftoverTempFile(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	src := newPluginSource(t, "hello", "1.0.0")

	mustCplugins(t, home, "dev", "hello@test-mp", src)
	mustCplugins(t, home, "undev", "hello@test-mp")
	addInstallEntry(t, home, "hello@test-mp", deadProjectEntry(t, "/nonexistent/install/path", "0.1.0"))
	mustCplugins(t, home, "prune")

	dir := filepath.Dir(installedPluginsPath(home))
	assertNoTempFiles(t, dir)
}

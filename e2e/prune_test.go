package e2e_test

import (
	"os"
	"path/filepath"
	"testing"
)

// deadProjectPath allocates a fresh temp directory and removes it, yielding a
// path guaranteed not to exist on disk.
func deadProjectPath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Remove(dir); err != nil {
		t.Fatalf("removing temp project dir: %v", err)
	}
	return dir
}

// deadProjectEntry returns a project-scope install entry pointing at a
// projectPath that does not exist on disk.
func deadProjectEntry(t *testing.T, installPath, version string) map[string]any {
	t.Helper()
	return map[string]any{
		"scope":       "project",
		"projectPath": deadProjectPath(t),
		"installPath": installPath,
		"version":     version,
	}
}

func TestPrune_removesDeadProjectInstall_keepsLive(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp") // user-scope install

	liveProject := t.TempDir()
	addInstallEntry(t, home, "hello@test-mp", map[string]any{
		"scope":       "project",
		"projectPath": liveProject,
		"installPath": installPath(t, home, "hello@test-mp"),
		"version":     "0.1.0",
	})
	deadInstallPath := cacheVersionDir(home, "test-mp", "hello", "0.0.1")
	deadEntry := deadProjectEntry(t, deadInstallPath, "0.0.1")
	addInstallEntry(t, home, "hello@test-mp", deadEntry)
	deadPath := deadEntry["projectPath"].(string)

	out := mustCplugins(t, home, "prune")
	assertContains(t, out, "→ pruning: hello@test-mp (project "+deadPath+")")

	var data map[string]any
	readJSON(t, installedPluginsPath(home), &data)
	plugins, _ := data["plugins"].(map[string]any)
	entries, _ := plugins["hello@test-mp"].([]any)
	if len(entries) != 2 {
		t.Fatalf("expected 2 remaining installs (user + live project), got %d: %v", len(entries), entries)
	}
	for _, e := range entries {
		entry, _ := e.(map[string]any)
		if entry["projectPath"] == deadPath {
			t.Errorf("dead project install was not removed: %v", entry)
		}
	}
}

func TestPrune_allInstallsDead_removesKey(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	removeInstalledPlugin(t, home, "hello@test-mp")
	addInstallEntry(t, home, "hello@test-mp", deadProjectEntry(t, "/nonexistent/install/path", "0.1.0"))

	mustCplugins(t, home, "prune")

	var data map[string]any
	readJSON(t, installedPluginsPath(home), &data)
	plugins, _ := data["plugins"].(map[string]any)
	if _, exists := plugins["hello@test-mp"]; exists {
		t.Error("expected hello@test-mp key to be removed once all its installs are dead")
	}
}

func TestPrune_dryRun_noChanges(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	removeInstalledPlugin(t, home, "hello@test-mp")
	addInstallEntry(t, home, "hello@test-mp", deadProjectEntry(t, "/nonexistent/install/path", "0.1.0"))

	before, err := os.ReadFile(installedPluginsPath(home))
	if err != nil {
		t.Fatal(err)
	}

	out := mustCplugins(t, home, "prune", "--dry-run")
	assertContains(t, out, "→ would prune: hello@test-mp")
	assertNotContains(t, out, "→ pruning:")
	assertNotContains(t, out, "cplugins clean")

	after, err := os.ReadFile(installedPluginsPath(home))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Errorf("registry changed under --dry-run\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestPrune_quiet_nothingToDo(t *testing.T) {
	home := setupHome(t)
	out := mustCplugins(t, home, "prune", "--quiet")
	if out != "" {
		t.Errorf("expected empty output with --quiet and nothing to prune, got: %s", out)
	}
}

func TestPrune_nothingToDo(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	out := mustCplugins(t, home, "prune")
	assertContains(t, out, "Nothing to prune.")
}

func TestPrune_projectPathStatError_notRemoved_warns(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	removeInstalledPlugin(t, home, "hello@test-mp")

	// A regular file where a directory is expected turns any os.Stat of a
	// path underneath it into ENOTDIR, not "not exist".
	blocker := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("writing blocker file: %v", err)
	}
	badProjectPath := filepath.Join(blocker, "sub")
	addInstallEntry(t, home, "hello@test-mp", map[string]any{
		"scope":       "project",
		"projectPath": badProjectPath,
		"installPath": "/nonexistent/install/path",
		"version":     "0.1.0",
	})

	stdout, stderr, err := runCplugins(t, home, "prune")
	if err != nil {
		t.Fatalf("prune failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	assertContains(t, stdout, "Nothing to prune.")
	assertContains(t, stderr, badProjectPath)

	var data map[string]any
	readJSON(t, installedPluginsPath(home), &data)
	plugins, _ := data["plugins"].(map[string]any)
	if _, exists := plugins["hello@test-mp"]; !exists {
		t.Error("expected the install with an unreadable projectPath to survive prune")
	}
}

func TestPrune_userScopeBrokenInstallPath_notRemoved(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	removeInstalledPlugin(t, home, "hello@test-mp")
	addInstallEntry(t, home, "hello@test-mp", map[string]any{
		"scope":       "user",
		"projectPath": "",
		"installPath": "/nonexistent/user/install/path",
		"version":     "0.1.0",
	})

	out := mustCplugins(t, home, "prune")
	assertContains(t, out, "Nothing to prune.")

	var data map[string]any
	readJSON(t, installedPluginsPath(home), &data)
	plugins, _ := data["plugins"].(map[string]any)
	if _, exists := plugins["hello@test-mp"]; !exists {
		t.Error("expected the user-scope install to survive prune (repair's domain, not prune's)")
	}
}

func TestPrune_preservesUnknownFields(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	setEntryFields(t, home, "hello@test-mp", extraFields)

	addInstallEntry(t, home, "hello@test-mp", deadProjectEntry(t, "/nonexistent/install/path", "0.1.0"))

	mustCplugins(t, home, "prune")

	entry := installedPluginEntry(t, home, "hello@test-mp")
	assertExtraFieldsIntact(t, entry)
}

func TestPrune_thenClean_reclaimsCache(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	removeInstalledPlugin(t, home, "hello@test-mp")

	deadCacheDir := cacheVersionDir(home, "test-mp", "hello", "0.0.1")
	os.MkdirAll(deadCacheDir, 0o755)
	writePluginJSON(t, deadCacheDir, "hello", "0.0.1")
	addInstallEntry(t, home, "hello@test-mp", deadProjectEntry(t, deadCacheDir, "0.0.1"))

	mustCplugins(t, home, "prune")

	out := mustCplugins(t, home, "clean", "--dry-run")
	assertContains(t, out, "orphan")
	assertFileExists(t, deadCacheDir)

	mustCplugins(t, home, "clean")
	assertFileNotExists(t, deadCacheDir)
}

// TestPrune_registryWriteFails_othersStillPruned forces the registry write for
// one plugin to fail while a second plugin's dead install is genuinely
// pruned, to verify a write failure on one plugin doesn't abort the command
// before it gets to report — or prune — the others.
//
// The failure is provoked by giving hello@test-mp a JSON null alongside its
// real dead-project install: ParseInstalledPlugins tolerates a null array
// element as a zero-value (and thus harmless) install, but the registry
// writer's stricter per-entry decoder rejects it, so the write for that one
// plugin fails deterministically without touching the file.
func TestPrune_registryWriteFails_othersStillPruned(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}, {"world", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	installPlugin(t, home, "world@test-mp")
	removeInstalledPlugin(t, home, "hello@test-mp")
	removeInstalledPlugin(t, home, "world@test-mp")

	addInstallEntry(t, home, "hello@test-mp", deadProjectEntry(t, "/nonexistent/install/path", "0.1.0"))
	addInstallEntry(t, home, "hello@test-mp", nil)

	worldDead := deadProjectEntry(t, "/nonexistent/install/path", "0.1.0")
	addInstallEntry(t, home, "world@test-mp", worldDead)
	worldDeadPath := worldDead["projectPath"].(string)

	stdout, stderr, err := runCplugins(t, home, "prune")
	if err != nil {
		t.Fatalf("expected prune to still succeed overall, got error: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	assertContains(t, stderr, "⚠ hello@test-mp:")
	assertContains(t, stdout, "→ pruning: world@test-mp (project "+worldDeadPath+")")

	var data map[string]any
	readJSON(t, installedPluginsPath(home), &data)
	plugins, _ := data["plugins"].(map[string]any)
	if _, exists := plugins["world@test-mp"]; exists {
		t.Error("expected world@test-mp's key to be removed once its only install was pruned")
	}
	if _, exists := plugins["hello@test-mp"]; !exists {
		t.Error("expected hello@test-mp to survive untouched after its registry write failed")
	}
}

// TestPrune_registryWriteFails_noneSucceed_exitsError verifies that when
// every candidate plugin's registry write fails, prune reports an error
// instead of a false "Nothing to prune."
func TestPrune_registryWriteFails_noneSucceed_exitsError(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	removeInstalledPlugin(t, home, "hello@test-mp")

	addInstallEntry(t, home, "hello@test-mp", deadProjectEntry(t, "/nonexistent/install/path", "0.1.0"))
	addInstallEntry(t, home, "hello@test-mp", nil)

	stdout, stderr, err := runCplugins(t, home, "prune")
	if err == nil {
		t.Fatalf("expected prune to exit with an error, got success\nstdout: %s\nstderr: %s", stdout, stderr)
	}
	assertContains(t, stderr, "⚠ hello@test-mp:")
	assertNotContains(t, stdout, "Nothing to prune.")
}

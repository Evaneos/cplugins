package e2e_test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClean_nothingToDo(t *testing.T) {
	home := setupHome(t)
	out := mustCplugins(t, home, "clean")
	assertContains(t, out, "Everything is clean")
}

func TestClean_nothingToDo_quiet(t *testing.T) {
	home := setupHome(t)
	out := mustCplugins(t, home, "clean", "--quiet")
	if out != "" {
		t.Errorf("expected empty output with --quiet and nothing to do, got: %s", out)
	}
}

func TestClean_dryRun_noChanges(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	orphan := filepath.Join(home, ".claude", "plugins", "cache", "test-mp", "hello", "0.0.1")
	os.MkdirAll(orphan, 0o755)

	mustCplugins(t, home, "clean", "--dry-run")

	assertFileExists(t, orphan)
}

func TestClean_dryRun_showsActions(t *testing.T) {
	home := setupHome(t)
	orphan := filepath.Join(home, ".claude", "plugins", "cache", "temp_git_abc")
	os.MkdirAll(orphan, 0o755)

	out := mustCplugins(t, home, "clean", "--dry-run")
	assertContains(t, out, "→ would remove orphan: "+orphan)
}

func TestClean_cleansOrphanTempGitDir(t *testing.T) {
	home := setupHome(t)
	cacheDir := filepath.Join(home, ".claude", "plugins", "cache")
	os.MkdirAll(cacheDir, 0o755)

	orphan := filepath.Join(cacheDir, "temp_git_abc123")
	os.MkdirAll(orphan, 0o755)

	out := mustCplugins(t, home, "clean")
	assertContains(t, out, "→ removing orphan: "+orphan)
	assertFileNotExists(t, orphan)
}

func TestClean_cleansOrphanVersionDir(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	orphanDir := cacheVersionDir(home, "test-mp", "hello", "0.0.1")
	os.MkdirAll(orphanDir, 0o755)
	writePluginJSON(t, orphanDir, "hello", "0.0.1")

	out := mustCplugins(t, home, "clean")
	assertContains(t, out, "orphan")
	assertFileNotExists(t, orphanDir)
}

func TestClean_multipleIssues(t *testing.T) {
	home := setupHome(t)

	orphan1 := filepath.Join(home, ".claude", "plugins", "cache", "temp_git_xyz")
	os.MkdirAll(orphan1, 0o755)

	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	orphan2 := cacheVersionDir(home, "test-mp", "hello", "0.0.1")
	os.MkdirAll(orphan2, 0o755)
	writePluginJSON(t, orphan2, "hello", "0.0.1")

	out := mustCplugins(t, home, "clean", "--dry-run")
	assertContains(t, out, "orphan")
}

func TestClean_quiet_withChanges(t *testing.T) {
	home := setupHome(t)
	cacheDir := filepath.Join(home, ".claude", "plugins", "cache")
	orphan := filepath.Join(cacheDir, "temp_git_quiet")
	os.MkdirAll(orphan, 0o755)

	out := mustCplugins(t, home, "clean", "--quiet")
	assertContains(t, out, "orphan")
}

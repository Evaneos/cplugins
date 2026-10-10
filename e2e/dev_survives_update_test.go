package e2e_test

// dev_survives_update_test.go covers repair restoring dev mode after Claude
// Code's plugin auto-update repointed a dev-mode plugin at the cache.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// simulateAutoUpdate points every install of key at cachePath, as Claude
// Code's auto-update does when the marketplace publishes a new version.
func simulateAutoUpdate(t *testing.T, home, key, cachePath string) {
	t.Helper()
	var data map[string]any
	readJSON(t, installedPluginsPath(home), &data)
	plugins, _ := data["plugins"].(map[string]any)
	entries, _ := plugins[key].([]any)
	for _, e := range entries {
		e.(map[string]any)["installPath"] = cachePath
	}
	writeJSON(t, installedPluginsPath(home), data)
}

func devStateFile(home string) string {
	return filepath.Join(home, ".claude", "cplugins", "dev.json")
}

// devState returns the recorded dev path of each plugin key.
func devState(t *testing.T, home string) map[string]string {
	t.Helper()
	state := map[string]string{}
	if _, err := os.Stat(devStateFile(home)); err != nil {
		return state
	}
	var doc struct {
		Plugins map[string]struct {
			Path string `json:"path"`
		} `json:"plugins"`
	}
	readJSON(t, devStateFile(home), &doc)
	for k, v := range doc.Plugins {
		state[k] = v.Path
	}
	return state
}

func setupDevPlugin(t *testing.T) (home, key, cachePath, src string) {
	t.Helper()
	home = setupHome(t)
	key = "hello@test-mp"
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, key)
	cachePath = installPath(t, home, key)
	src, _ = filepath.EvalSymlinks(newPluginSource(t, "hello", "0.1.0"))
	mustCplugins(t, home, "dev", key, src)
	return home, key, cachePath, src
}

func TestRepair_restoresDevAfterUpdate(t *testing.T) {
	home, key, cachePath, src := setupDevPlugin(t)
	simulateAutoUpdate(t, home, key, cachePath)

	out := mustCplugins(t, home, "repair")

	assertContains(t, out, key)
	assertContains(t, out, "dev mode restored")
	if ip := installPath(t, home, key); ip != src {
		t.Errorf("installPath = %q, want %q", ip, src)
	}
}

func TestRepair_devStillActive_leftAlone(t *testing.T) {
	home, key, _, src := setupDevPlugin(t)

	out := mustCplugins(t, home, "repair")

	assertContains(t, out, "Nothing to repair.")
	if ip := installPath(t, home, key); ip != src {
		t.Errorf("installPath = %q, want %q", ip, src)
	}
}

func TestRepair_devPathGone_forgetsDev(t *testing.T) {
	home, key, cachePath, src := setupDevPlugin(t)
	simulateAutoUpdate(t, home, key, cachePath)
	if err := os.RemoveAll(src); err != nil {
		t.Fatal(err)
	}

	out := mustCplugins(t, home, "repair")

	if strings.Contains(out, "dev mode restored") {
		t.Errorf("restored a dev path that no longer exists: %s", out)
	}
	if ip := installPath(t, home, key); ip != cachePath {
		t.Errorf("installPath = %q, want %q", ip, cachePath)
	}
	if _, ok := devState(t, home)[key]; ok {
		t.Errorf("dev record kept for a dev path that no longer exists")
	}
}

func TestUndev_forgetsDev(t *testing.T) {
	home, key, cachePath, _ := setupDevPlugin(t)

	mustCplugins(t, home, "undev", key)
	if _, ok := devState(t, home)[key]; ok {
		t.Fatalf("dev record kept after undev")
	}

	mustCplugins(t, home, "repair")
	if ip := installPath(t, home, key); ip != cachePath {
		t.Errorf("repair re-applied dev after undev: installPath = %q", ip)
	}
}

func TestUndev_afterUpdate_forgetsDev(t *testing.T) {
	home, key, cachePath, _ := setupDevPlugin(t)
	simulateAutoUpdate(t, home, key, cachePath)

	if _, _, err := runCplugins(t, home, "undev", key); err == nil {
		t.Fatalf("expected undev to report the plugin is not in dev mode")
	}
	if _, ok := devState(t, home)[key]; ok {
		t.Errorf("dev record kept after undev")
	}
}

func TestRepair_devWithoutRecord_recordedThenRestored(t *testing.T) {
	home, key, cachePath, src := setupDevPlugin(t)
	if err := os.RemoveAll(filepath.Dir(devStateFile(home))); err != nil {
		t.Fatal(err)
	}

	mustCplugins(t, home, "repair")
	if got := devState(t, home)[key]; got != src {
		t.Fatalf("dev record = %q, want %q", got, src)
	}

	simulateAutoUpdate(t, home, key, cachePath)
	mustCplugins(t, home, "repair")
	if ip := installPath(t, home, key); ip != src {
		t.Errorf("installPath = %q, want %q", ip, src)
	}
}

func TestRepair_unreadableDevState_setAsideAndKeepsWorking(t *testing.T) {
	home, key, cachePath, _ := setupDevPlugin(t)
	simulateAutoUpdate(t, home, key, cachePath)
	if err := os.WriteFile(devStateFile(home), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := runCplugins(t, home, "repair")
	mustSucceed(t, "cplugins repair", stdout, stderr, err)

	assertContains(t, stderr, "set aside")
	matches, _ := filepath.Glob(devStateFile(home) + ".unreadable-*")
	if len(matches) != 1 {
		t.Errorf("expected the unreadable file set aside, found %v", matches)
	}
}

func TestDev_keepsUnknownFieldsInDevState(t *testing.T) {
	home, key, _, _ := setupDevPlugin(t)
	writeJSON(t, devStateFile(home), map[string]any{
		"schema":  2,
		"plugins": map[string]any{key: map[string]any{"path": "/x", "branch": "feat"}},
	})

	src, _ := filepath.EvalSymlinks(newPluginSource(t, "hello", "0.1.0"))
	mustCplugins(t, home, "dev", key, src)

	var doc map[string]any
	readJSON(t, devStateFile(home), &doc)
	if doc["schema"] != float64(2) {
		t.Errorf("top-level field dropped: %v", doc)
	}
	entry := doc["plugins"].(map[string]any)[key].(map[string]any)
	if entry["branch"] != "feat" || entry["path"] != src {
		t.Errorf("entry = %v, want branch kept and path %q", entry, src)
	}
}

func TestRepair_uninstalledPlugin_forgetsDev(t *testing.T) {
	home, key, _, _ := setupDevPlugin(t)
	removeInstalledPlugin(t, home, key)

	mustCplugins(t, home, "repair")

	if _, ok := devState(t, home)[key]; ok {
		t.Errorf("dev record kept for an uninstalled plugin")
	}
}

func TestRepair_recordFollowsActiveDevPath(t *testing.T) {
	home, key, cachePath, _ := setupDevPlugin(t)
	other, _ := filepath.EvalSymlinks(newPluginSource(t, "hello", "0.1.0"))
	simulateAutoUpdate(t, home, key, other)

	mustCplugins(t, home, "repair")
	simulateAutoUpdate(t, home, key, cachePath)
	mustCplugins(t, home, "repair")

	if ip := installPath(t, home, key); ip != other {
		t.Errorf("installPath = %q, want the last active dev path %q", ip, other)
	}
}

func TestRepair_partialUpdate_restoresAllInstalls(t *testing.T) {
	home, key, cachePath, src := setupDevPlugin(t)
	addInstallEntry(t, home, key, map[string]any{
		"scope": "project", "projectPath": t.TempDir(),
		"installPath": cachePath, "version": "0.1.0",
	})

	mustCplugins(t, home, "repair")

	var data map[string]any
	readJSON(t, installedPluginsPath(home), &data)
	for _, e := range data["plugins"].(map[string]any)[key].([]any) {
		if ip := e.(map[string]any)["installPath"]; ip != src {
			t.Errorf("install left at %v, want %q", ip, src)
		}
	}
}

func TestRepair_unreadableDevStateFile_leftUntouched(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads files regardless of permissions")
	}
	home, _, _, _ := setupDevPlugin(t)
	if err := os.Chmod(devStateFile(home), 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(devStateFile(home), 0o644) })

	stdout, stderr, err := runCplugins(t, home, "repair")
	mustSucceed(t, "cplugins repair", stdout, stderr, err)

	assertContains(t, stderr, "left untouched")
	if _, err := os.Stat(devStateFile(home)); err != nil {
		t.Errorf("dev state moved or removed: %v", err)
	}
}

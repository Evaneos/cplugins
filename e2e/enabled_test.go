package e2e_test

// enabled_test.go covers the ENABLED column: cplugins reads enabledPlugins
// from the settings.json chain instead of shelling out to
// 'claude plugin list --json'.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeSettings writes a settings.json with the given enabledPlugins map.
func writeSettings(t *testing.T, path string, enabledPlugins map[string]bool) {
	t.Helper()
	writeJSON(t, path, map[string]any{"enabledPlugins": enabledPlugins})
}

func TestStatus_enabled_absentKeyDefaultsYes(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	out := mustCplugins(t, home, "status")
	assertContains(t, out, "ENABLED")
	assertContains(t, out, "yes")
}

func TestStatus_enabled_falseInUserSettings(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	writeSettings(t, filepath.Join(home, ".claude", "settings.json"), map[string]bool{
		"hello@test-mp": false,
	})

	out := mustCplugins(t, home, "status")
	assertContains(t, out, "no")
}

func TestStatus_enabled_projectOverridesUser(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	writeSettings(t, filepath.Join(home, ".claude", "settings.json"), map[string]bool{
		"hello@test-mp": false,
	})

	projectDir := t.TempDir()
	writeSettings(t, filepath.Join(projectDir, ".claude", "settings.json"), map[string]bool{
		"hello@test-mp": true,
	})

	out := mustCpluginsInDir(t, projectDir, home, "status")
	assertContains(t, out, "yes")
	assertNotContains(t, out, "no")
}

func TestStatus_enabled_localSettingsOverridesSettings(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	writeSettings(t, filepath.Join(home, ".claude", "settings.json"), map[string]bool{
		"hello@test-mp": true,
	})
	writeSettings(t, filepath.Join(home, ".claude", "settings.local.json"), map[string]bool{
		"hello@test-mp": false,
	})

	out := mustCplugins(t, home, "status")
	assertContains(t, out, "no")
}

// TestStatus_enabled_projectSettingsFoundFromSubdirectory covers walking up
// from the CWD to find the project's .claude directory, mirroring how Claude
// Code itself locates project settings.
func TestStatus_enabled_projectSettingsFoundFromSubdirectory(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	projectRoot := t.TempDir()
	writeSettings(t, filepath.Join(projectRoot, ".claude", "settings.json"), map[string]bool{
		"hello@test-mp": false,
	})

	subDir := filepath.Join(projectRoot, "internal", "api")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("creating subdirectory: %v", err)
	}

	out := mustCpluginsInDir(t, subDir, home, "status")
	assertContains(t, out, "no")
}

func TestStatus_enabled_missingSettingsFile(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	if err := os.Remove(filepath.Join(home, ".claude", "settings.json")); err != nil {
		t.Fatalf("removing settings.json: %v", err)
	}

	out := mustCplugins(t, home, "status")
	assertContains(t, out, "yes")
}

// TestStatus_allProjects_enabledPerInstallProject covers the ENABLED column
// with --all-projects: each project-scope install must reflect its own
// project's settings chain, not the CWD's.
func TestStatus_allProjects_enabledPerInstallProject(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	removeInstalledPlugin(t, home, "hello@test-mp")

	projectA := t.TempDir()
	projectB := t.TempDir()
	cwdDir := t.TempDir()

	for _, dir := range []string{projectA, projectB} {
		addInstallEntry(t, home, "hello@test-mp", map[string]any{
			"scope":       "project",
			"projectPath": dir,
			"installPath": cacheVersionDir(home, "test-mp", "hello", "0.1.0"),
			"version":     "0.1.0",
		})
	}

	writeSettings(t, filepath.Join(projectA, ".claude", "settings.json"), map[string]bool{
		"hello@test-mp": false,
	})
	writeSettings(t, filepath.Join(projectB, ".claude", "settings.json"), map[string]bool{
		"hello@test-mp": true,
	})
	// The CWD's own settings disable the plugin too, to prove neither row
	// inherits the CWD's state instead of its own project's.
	writeSettings(t, filepath.Join(cwdDir, ".claude", "settings.json"), map[string]bool{
		"hello@test-mp": false,
	})

	out := mustCpluginsInDir(t, cwdDir, home, "status", "--all-projects")

	var lineA, lineB string
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, filepath.Base(projectA)) {
			lineA = l
		}
		if strings.Contains(l, filepath.Base(projectB)) {
			lineB = l
		}
	}
	if lineA == "" || lineB == "" {
		t.Fatalf("expected rows for both project-scope installs, got:\n%s", out)
	}
	if !strings.Contains(lineA, "no") {
		t.Errorf("projectA row should show ENABLED=no per its own settings: %q", lineA)
	}
	if !strings.Contains(lineB, "yes") {
		t.Errorf("projectB row should show ENABLED=yes per its own settings, not the CWD's: %q", lineB)
	}
}

func TestStatus_enabled_invalidSettingsJSON(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	if err := os.WriteFile(filepath.Join(home, ".claude", "settings.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatalf("writing invalid settings.json: %v", err)
	}

	out := mustCplugins(t, home, "status")
	assertContains(t, out, "yes")
}

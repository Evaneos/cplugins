package e2e_test

import (
	"strings"
	"testing"
)

func TestUninstall_afterDevUndev(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	src := newPluginSource(t, "hello", "0.1.0")

	mustCplugins(t, home, "dev", "hello@test-mp", src)
	mustCplugins(t, home, "undev", "hello@test-mp")

	if _, has := installedPluginEntry(t, home, "hello@test-mp")["projectPath"]; has {
		t.Error("user install carries a projectPath key after dev/undev")
	}
	mustClaudeOK(t, home, "plugin", "uninstall", "hello@test-mp")
}

func TestRepair_dropsEmptyUserProjectPath(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	entry := installedPluginEntry(t, home, "hello@test-mp")
	entry["projectPath"] = ""
	removeInstalledPlugin(t, home, "hello@test-mp")
	addInstallEntry(t, home, "hello@test-mp", entry)

	stdout, stderr, err := runClaude(t, home, "plugin", "uninstall", "hello@test-mp")
	if err == nil || !strings.Contains(stdout+stderr, "is not installed in user scope") {
		t.Fatalf("precondition: expected uninstall to fail on the empty user projectPath, got err=%v\n%s%s", err, stdout, stderr)
	}

	out := mustCplugins(t, home, "repair")
	assertContains(t, out, "hello@test-mp")
	if _, has := installedPluginEntry(t, home, "hello@test-mp")["projectPath"]; has {
		t.Error("repair left the empty projectPath on the user install")
	}
	mustClaudeOK(t, home, "plugin", "uninstall", "hello@test-mp")
}

func TestRepair_keepsProjectInstallProjectPath(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	entry := installedPluginEntry(t, home, "hello@test-mp")
	entry["scope"] = "project"
	entry["projectPath"] = home
	removeInstalledPlugin(t, home, "hello@test-mp")
	addInstallEntry(t, home, "hello@test-mp", entry)

	mustCplugins(t, home, "repair")
	if got := installedPluginEntry(t, home, "hello@test-mp")["projectPath"]; got != home {
		t.Errorf("projectPath = %v, want %q", got, home)
	}
}

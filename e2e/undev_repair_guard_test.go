package e2e_test

// undev_repair_guard_test.go covers undev and repair refusing to recache a
// plugin whose marketplace source type has no fixed plugins/ layout (npm,
// archive, command, ...) — Marketplace.SupportsSourceVersion() reports false
// for those, and cache.RecachePlugin must never be handed such a path.

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestUndev_unsupportedSourceType_refuses(t *testing.T) {
	home := setupHome(t)
	mpName := "npm-mp"
	key := "hello@" + mpName

	installLocation := filepath.Join(home, ".claude", "plugins", "cache", mpName)
	injectMarketplace(t, home, mpName, map[string]any{"source": "npm", "package": "@org/hello"}, installLocation, false)

	devSource := newPluginSource(t, "hello", "0.1.0")
	addInstallEntry(t, home, key, map[string]any{
		"scope": "user", "projectPath": "",
		"installPath": devSource, "version": "0.1.0",
	})

	stdout, stderr, err := runCplugins(t, home, "undev", key)
	if err == nil {
		t.Fatalf("expected undev to fail, got stdout: %s", stdout)
	}
	combined := stdout + stderr
	if !strings.Contains(combined, "npm") || !strings.Contains(combined, key) {
		t.Errorf("expected error to name the plugin and its source type, got: %s", combined)
	}

	entry := installedPluginEntry(t, home, key)
	if entry["installPath"] != devSource {
		t.Errorf("installPath changed despite refusal: %v", entry["installPath"])
	}
}

func TestRepair_unsupportedSourceType_refuses(t *testing.T) {
	home := setupHome(t)
	mpName := "npm-mp"
	key := "hello@" + mpName

	installLocation := filepath.Join(home, ".claude", "plugins", "cache", mpName)
	injectMarketplace(t, home, mpName, map[string]any{"source": "npm", "package": "@org/hello"}, installLocation, false)

	brokenPath := filepath.Join(home, ".claude", "plugins", "cache", mpName, "hello", "0.1.0")
	addInstallEntry(t, home, key, map[string]any{
		"scope": "user", "projectPath": "",
		"installPath": brokenPath, "version": "0.1.0",
	})

	stdout, stderr, err := runCplugins(t, home, "repair")
	if err != nil {
		t.Fatalf("repair failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	if !strings.Contains(stderr, "npm") || !strings.Contains(stderr, key) {
		t.Errorf("expected warning to name the plugin and its source type, got: %s", stderr)
	}

	entry := installedPluginEntry(t, home, key)
	if entry["installPath"] != brokenPath {
		t.Errorf("installPath changed despite refusal: %v", entry["installPath"])
	}
}

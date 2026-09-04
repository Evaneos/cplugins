package e2e_test

import (
	"os"
	"path/filepath"
	"testing"
)

// --- dev command -----------------------------------------------------------

func TestDev_basic(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	src := newPluginSource(t, "hello", "0.1.0")

	out := mustCplugins(t, home, "dev", "hello@test-mp", src)

	assertContains(t, out, "hello@test-mp")
	assertContains(t, out, src)
	// installPath must point directly to the source dir, not a symlink in cache
	ip := installPath(t, home, "hello@test-mp")
	resolvedSrc, _ := filepath.EvalSymlinks(src)
	if ip != resolvedSrc {
		t.Errorf("installPath = %q, want %q", ip, resolvedSrc)
	}
	fi, err := os.Lstat(ip)
	if err != nil {
		t.Fatalf("installPath not accessible: %v", err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Error("expected real directory at installPath, got symlink")
	}
}

func TestDev_installPathPointsToSource(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	src := newPluginSource(t, "hello", "0.1.0")

	mustCplugins(t, home, "dev", "hello@test-mp", src)

	ip := installPath(t, home, "hello@test-mp")
	resolvedSrc, _ := filepath.EvalSymlinks(src)
	if ip != resolvedSrc {
		t.Errorf("installPath = %q, want %q", ip, resolvedSrc)
	}
}

func TestDev_sourceVersionIndependent(t *testing.T) {
	// Source version doesn't affect installPath — it's the source dir itself.
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	src := newPluginSource(t, "hello", "9.9.9") // source version differs from registry

	mustCplugins(t, home, "dev", "hello@test-mp", src)

	ip := installPath(t, home, "hello@test-mp")
	resolvedSrc, _ := filepath.EvalSymlinks(src)
	if ip != resolvedSrc {
		t.Errorf("installPath = %q, want %q", ip, resolvedSrc)
	}
	if !filepath.IsAbs(ip) {
		t.Errorf("installPath should be absolute: %s", ip)
	}
}

func TestDev_idempotent(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	src := newPluginSource(t, "hello", "0.1.0")

	mustCplugins(t, home, "dev", "hello@test-mp", src)
	mustCplugins(t, home, "dev", "hello@test-mp", src) // second call must not error

	ip := installPath(t, home, "hello@test-mp")
	resolvedSrc, _ := filepath.EvalSymlinks(src)
	if ip != resolvedSrc {
		t.Errorf("installPath = %q, want %q after idempotent dev", ip, resolvedSrc)
	}
}

func TestDev_replacesWithNewSource(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	src1 := newPluginSource(t, "hello", "0.1.0")
	src2 := newPluginSource(t, "hello", "0.2.0")

	mustCplugins(t, home, "dev", "hello@test-mp", src1)
	mustCplugins(t, home, "dev", "hello@test-mp", src2)

	ip := installPath(t, home, "hello@test-mp")
	resolvedSrc2, _ := filepath.EvalSymlinks(src2)
	if ip != resolvedSrc2 {
		t.Errorf("installPath = %q, want %q after switching source", ip, resolvedSrc2)
	}
}

func TestDev_unknownPlugin(t *testing.T) {
	home := setupHome(t)
	_, _, err := runCplugins(t, home, "dev", "ghost@nowhere", "/some/path")
	if err == nil {
		t.Error("expected error for unknown plugin, got nil")
	}
}

func TestDev_invalidSourcePath(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	_, _, err := runCplugins(t, home, "dev", "hello@test-mp", "/nonexistent/path")
	if err == nil {
		t.Error("expected error for invalid source path, got nil")
	}
}

// --- undev command ---------------------------------------------------------

func TestUndev_basic(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	src := newPluginSource(t, "hello", "0.1.0")

	mustCplugins(t, home, "dev", "hello@test-mp", src)
	out := mustCplugins(t, home, "undev", "hello@test-mp")

	assertContains(t, out, "hello@test-mp")
	ip := installPath(t, home, "hello@test-mp")
	assertUnderCache(t, home, ip)
	pj := filepath.Join(ip, ".claude-plugin", "plugin.json")
	if _, err := os.Stat(pj); err != nil {
		t.Errorf("plugin.json missing at %s: %v", pj, err)
	}
}

func TestUndev_notInDevMode(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	_, _, err := runCplugins(t, home, "undev", "hello@test-mp")
	if err == nil {
		t.Error("expected error when plugin is not in dev mode")
	}
}

func TestUndev_gitMarketplace(t *testing.T) {
	home := setupHome(t)
	injectGitMarketplace(t, home, "github-mp", "github", "org/plugin-repo", []pluginSpec{{"hello", "1.0.0"}})

	cachePath := cacheVersionDir(home, "github-mp", "hello", "1.0.0")
	writePluginJSON(t, cachePath, "hello", "1.0.0")
	addInstallEntry(t, home, "hello@github-mp", map[string]any{
		"scope": "user", "projectPath": "",
		"installPath": cachePath, "version": "1.0.0",
	})

	src := newPluginSource(t, "hello", "1.0.0")
	mustCplugins(t, home, "dev", "hello@github-mp", src)

	out := mustCplugins(t, home, "undev", "hello@github-mp")
	assertContains(t, out, "hello@github-mp")

	ip := installPath(t, home, "hello@github-mp")
	assertUnderCache(t, home, ip)
	pj := filepath.Join(ip, ".claude-plugin", "plugin.json")
	if _, err := os.Stat(pj); err != nil {
		t.Errorf("plugin.json missing at %s: %v", pj, err)
	}
}

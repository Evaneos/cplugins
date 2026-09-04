package e2e_test

// status_test.go covers the plugin development use cases:
//
// Case 1: directory marketplace — plugins as subdirectories in a local git repo
// Case 2: git marketplace (github/url source) — plugins in remote git repositories

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Case 1: directory marketplace

func TestStatus_empty(t *testing.T) {
	home := setupHome(t)
	out := mustCplugins(t, home, "status")
	assertContains(t, out, "PLUGIN")
	assertNotContains(t, out, "hello")
}

func TestStatus_singleOk(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	out := mustCplugins(t, home, "status")
	assertContains(t, out, "hello")
	assertContains(t, out, "test-mp")
	assertContains(t, out, "ok")
}

func TestStatus_listAlias(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	for _, alias := range []string{"list", "ls"} {
		out := mustCplugins(t, home, alias)
		assertContains(t, out, "PLUGIN")
		assertContains(t, out, "hello")
	}
}

func TestStatus_stale(t *testing.T) {
	home := setupHome(t)
	mpDir := newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	// Bump source version: plugin.json now says 0.2.0
	bumpPluginSourceVersion(t, filepath.Join(mpDir, "plugins", "hello"), "0.2.0")

	out := mustCplugins(t, home, "status")
	assertContains(t, out, "stale")
}

func TestStatus_devOk(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	src := newPluginSource(t, "hello", "0.1.0")

	mustCplugins(t, home, "dev", "hello@test-mp", src)

	out := mustCplugins(t, home, "status")
	assertContains(t, out, "DEV")
	// Symlink is valid so health should be ok (not broken)
	assertNotContains(t, out, "broken")
}

func TestStatus_devBrokenSource(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	src := newPluginSource(t, "hello", "0.1.0")

	mustCplugins(t, home, "dev", "hello@test-mp", src)

	// Delete the source directory — installPath now points to a missing dir
	os.RemoveAll(src)

	out := mustCplugins(t, home, "status")
	assertContains(t, out, "broken")
}

func TestStatus_detailedView(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	out := mustCplugins(t, home, "status", "hello@test-mp")
	assertContains(t, out, "Plugin:")
	assertContains(t, out, "Marketplace:")
	assertContains(t, out, "Scope:")
	assertContains(t, out, "Version:")
}

func TestStatus_detailedView_devMode(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	src := newPluginSource(t, "hello", "0.1.0")

	mustCplugins(t, home, "dev", "hello@test-mp", src)

	out := mustCplugins(t, home, "status", "hello@test-mp")
	assertContains(t, out, "Dev path:")
	assertContains(t, out, src)
}

func TestStatus_duplicateInstall(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	// Inject a second entry with the SAME "user" scope → real dup
	addInstallEntry(t, home, "hello@test-mp", map[string]any{
		"scope":       "user",
		"projectPath": "",
		"installPath": cacheVersionDir(home, "test-mp", "hello", "0.1.0"),
		"version":     "0.1.0",
	})

	out := mustCplugins(t, home, "status")
	assertContains(t, out, "dup")
	assertContains(t, out, "Duplicate install entries detected")
	assertContains(t, out, "hello@test-mp")
	assertContains(t, out, "2 entries")
}

func TestStatus_scopeColumn(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	out := mustCplugins(t, home, "status")
	assertContains(t, out, "SCOPE")
	assertContains(t, out, "user")
}

func TestStatus_userAndProjectScope(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	// Add a project-scope entry
	addInstallEntry(t, home, "hello@test-mp", map[string]any{
		"scope":       "project",
		"projectPath": home,
		"installPath": cacheVersionDir(home, "test-mp", "hello", "0.1.0"),
		"version":     "0.1.0",
	})

	// Run from home so the project-scope entry (projectPath=home) is visible
	out := mustCpluginsInDir(t, home, home, "status")
	// Two rows for hello (user + project), not a dup
	assertNotContains(t, out, "dup")
	assertNotContains(t, out, "Duplicate")
	// Both scopes are displayed
	assertContains(t, out, "user")
	assertContains(t, out, "project")
}

func TestStatus_projectScopeVisibleFromProjectDir(t *testing.T) {
	home := setupHome(t)
	projectDir := t.TempDir()
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	// Replace the user entry with a project-scope entry with projectPath = projectDir
	removeInstalledPlugin(t, home, "hello@test-mp")
	addInstallEntry(t, home, "hello@test-mp", map[string]any{
		"scope":       "project",
		"projectPath": projectDir,
		"installPath": cacheVersionDir(home, "test-mp", "hello", "0.1.0"),
		"version":     "0.1.0",
	})

	// From projectDir → visible
	out := mustCpluginsInDir(t, projectDir, home, "status")
	assertContains(t, out, "hello")
	assertContains(t, out, "project")
}

func TestStatus_projectScopeHiddenFromOtherDir(t *testing.T) {
	home := setupHome(t)
	projectDir := t.TempDir()
	otherDir := t.TempDir()
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	// Switch to project-scope with projectPath = projectDir
	removeInstalledPlugin(t, home, "hello@test-mp")
	addInstallEntry(t, home, "hello@test-mp", map[string]any{
		"scope":       "project",
		"projectPath": projectDir,
		"installPath": cacheVersionDir(home, "test-mp", "hello", "0.1.0"),
		"version":     "0.1.0",
	})

	// From another directory → invisible
	out := mustCpluginsInDir(t, otherDir, home, "status")
	assertNotContains(t, out, "hello")
}

func TestStatus_projectScopeVisibleFromSubdir(t *testing.T) {
	home := setupHome(t)
	projectDir := t.TempDir()
	subDir := filepath.Join(projectDir, "src", "pkg")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	removeInstalledPlugin(t, home, "hello@test-mp")
	addInstallEntry(t, home, "hello@test-mp", map[string]any{
		"scope":       "project",
		"projectPath": projectDir,
		"installPath": cacheVersionDir(home, "test-mp", "hello", "0.1.0"),
		"version":     "0.1.0",
	})

	// From a subdirectory of the project → visible
	out := mustCpluginsInDir(t, subDir, home, "status")
	assertContains(t, out, "hello")
}

func TestStatus_projectColumn(t *testing.T) {
	home := setupHome(t)
	projectDir := filepath.Join(t.TempDir(), "my-project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	addInstallEntry(t, home, "hello@test-mp", map[string]any{
		"scope":       "project",
		"projectPath": projectDir,
		"installPath": cacheVersionDir(home, "test-mp", "hello", "0.1.0"),
		"version":     "0.1.0",
	})

	out := mustCpluginsInDir(t, projectDir, home, "status")
	assertContains(t, out, "PROJECT")
	assertContains(t, out, "my-project")

	// The user-scope row keeps the PROJECT column empty: no token between
	// the scope and the marketplace name, just padding.
	userRow := regexp.MustCompile(`hello\s+user\s+test-mp`)
	if !userRow.MatchString(out) {
		t.Errorf("expected the user-scope row to have an empty PROJECT cell, got:\n%s", out)
	}
}

func TestStatus_multipleProjectScopes_notDup(t *testing.T) {
	home := setupHome(t)
	base := t.TempDir()
	projectA := filepath.Join(base, "project-a")
	projectB := filepath.Join(base, "project-b")
	for _, dir := range []string{projectA, projectB} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	removeInstalledPlugin(t, home, "hello@test-mp")

	for _, projectDir := range []string{projectA, projectB} {
		addInstallEntry(t, home, "hello@test-mp", map[string]any{
			"scope":       "project",
			"projectPath": projectDir,
			"installPath": cacheVersionDir(home, "test-mp", "hello", "0.1.0"),
			"version":     "0.1.0",
		})
	}

	// Two distinct projects in project scope: not a duplicate, even though the
	// "project" scope appears twice.
	out := mustCpluginsInDir(t, projectA, home, "status", "--all-projects")
	assertNotContains(t, out, "dup")
	assertNotContains(t, out, "Duplicate")
	assertContains(t, out, "project-a")
	assertContains(t, out, "project-b")
}

func TestStatus_sameProjectDuplicate_isDup(t *testing.T) {
	home := setupHome(t)
	projectDir := t.TempDir()
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	removeInstalledPlugin(t, home, "hello@test-mp")

	entry := map[string]any{
		"scope":       "project",
		"projectPath": projectDir,
		"installPath": cacheVersionDir(home, "test-mp", "hello", "0.1.0"),
		"version":     "0.1.0",
	}
	// Same scope AND same projectPath, injected twice: real duplicate.
	addInstallEntry(t, home, "hello@test-mp", entry)
	addInstallEntry(t, home, "hello@test-mp", entry)

	out := mustCpluginsInDir(t, projectDir, home, "status")
	assertContains(t, out, "dup")
	assertContains(t, out, "Duplicate install entries detected")
}

func TestStatus_unknownPlugin(t *testing.T) {
	home := setupHome(t)
	_, _, err := runCplugins(t, home, "status", "ghost@nowhere")
	if err == nil {
		t.Error("expected error for unknown plugin, got nil")
	}
}

func TestStatus_multiplePlugins(t *testing.T) {
	home := setupHome(t)
	mpDir := newMarketplace(t, home, "test-mp", []pluginSpec{
		{"alpha", "0.1.0"},
		{"beta", "0.2.0"},
	})
	installPlugin(t, home, "alpha@test-mp")
	installPlugin(t, home, "beta@test-mp")

	// Make alpha stale
	bumpPluginSourceVersion(t, filepath.Join(mpDir, "plugins", "alpha"), "0.3.0")

	out := mustCplugins(t, home, "status")
	assertContains(t, out, "alpha")
	assertContains(t, out, "beta")
	assertContains(t, out, "stale")
	assertContains(t, out, "ok")
}

func TestStatus_allProjects_showsOutOfCWD(t *testing.T) {
	home := setupHome(t)
	projectDir := t.TempDir()
	otherDir := t.TempDir()
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	// Switch to project-scope with projectPath = projectDir
	removeInstalledPlugin(t, home, "hello@test-mp")
	addInstallEntry(t, home, "hello@test-mp", map[string]any{
		"scope":       "project",
		"projectPath": projectDir,
		"installPath": cacheVersionDir(home, "test-mp", "hello", "0.1.0"),
		"version":     "0.1.0",
	})

	// Without --all-projects, from otherDir: invisible
	out := mustCpluginsInDir(t, otherDir, home, "status")
	assertNotContains(t, out, "hello")

	// With --all-projects, from otherDir: visible
	out = mustCpluginsInDir(t, otherDir, home, "status", "--all-projects")
	assertContains(t, out, "hello")
	assertContains(t, out, "project")
}

// Case 2: git marketplace (github / url source)

func TestStatus_gitMarketplace_ok(t *testing.T) {
	home := setupHome(t)
	injectGitMarketplace(t, home, "github-mp", "github", "org/plugin-repo", nil)

	cachePath := cacheVersionDir(home, "github-mp", "hello", "1.0.0")
	writePluginJSON(t, cachePath, "hello", "1.0.0")
	addInstallEntry(t, home, "hello@github-mp", map[string]any{
		"scope": "user", "projectPath": "",
		"installPath": cachePath, "version": "1.0.0",
	})

	out := mustCplugins(t, home, "status")
	assertContains(t, out, "hello")
	assertContains(t, out, "MARKETPLACE")
	assertContains(t, out, "ok")
	assertNotContains(t, out, "stale")
	assertNotContains(t, out, "broken")
}

func TestStatus_gitMarketplace_broken(t *testing.T) {
	home := setupHome(t)
	injectGitMarketplace(t, home, "github-mp", "github", "org/plugin-repo", nil)

	// No cache: writePluginJSON is not called
	cachePath := cacheVersionDir(home, "github-mp", "hello", "1.0.0")
	addInstallEntry(t, home, "hello@github-mp", map[string]any{
		"scope": "user", "projectPath": "",
		"installPath": cachePath, "version": "1.0.0",
	})

	out := mustCplugins(t, home, "status")
	assertContains(t, out, "broken")
}

func TestStatus_devGitBranch(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	src := newGitPluginSource(t, "hello", "0.1.0", "feat/my-feature")

	mustCplugins(t, home, "dev", "hello@test-mp", src)

	out := mustCplugins(t, home, "status")
	assertContains(t, out, "feat/my-feature")
}

func TestStatus_devGitBranch_worktreeSubdirectory(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	// Main repo with a worktree; plugin in plugins/hello of the worktree
	mainRepo := t.TempDir()
	runGit(t, mainRepo, "init", "-b", "main")
	runGit(t, mainRepo, "commit", "--allow-empty", "-m", "init")

	wtDir := t.TempDir()
	os.RemoveAll(wtDir) // git worktree add needs a non-existing path
	runGit(t, mainRepo, "worktree", "add", "-b", "feat/wt-branch", wtDir)

	pluginDir := filepath.Join(wtDir, "plugins", "hello")
	writePluginJSON(t, pluginDir, "hello", "0.1.0")

	mustCplugins(t, home, "dev", "hello@test-mp", pluginDir)

	out := mustCplugins(t, home, "status")
	assertContains(t, out, "feat/wt-branch")
}

func TestStatus_devGitBranch_subdirectory(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	// Git repo at parent, plugin in plugins/hello subdirectory
	repoDir := t.TempDir()
	pluginDir := filepath.Join(repoDir, "plugins", "hello")
	writePluginJSON(t, pluginDir, "hello", "0.1.0")
	runGit(t, repoDir, "init", "-b", "feat/subdir-branch")

	mustCplugins(t, home, "dev", "hello@test-mp", pluginDir)

	out := mustCplugins(t, home, "status")
	assertContains(t, out, "feat/subdir-branch")
}

func TestStatus_devNoGit_sourceEmpty(t *testing.T) {
	home := setupHome(t)
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")
	src := newPluginSource(t, "hello", "0.1.0") // not a git repository

	mustCplugins(t, home, "dev", "hello@test-mp", src)

	out := mustCplugins(t, home, "status")
	assertContains(t, out, "DEV")
	// No git branch: the SOURCE column stays empty (-)
	assertContains(t, out, "-")
}

func TestStatus_gitMarketplace_dev(t *testing.T) {
	home := setupHome(t)
	injectGitMarketplace(t, home, "github-mp", "github", "org/plugin-repo", nil)

	cachePath := cacheVersionDir(home, "github-mp", "hello", "1.0.0")
	writePluginJSON(t, cachePath, "hello", "1.0.0")
	addInstallEntry(t, home, "hello@github-mp", map[string]any{
		"scope": "user", "projectPath": "",
		"installPath": cachePath, "version": "1.0.0",
	})

	src := newPluginSource(t, "hello", "1.0.0")
	mustCplugins(t, home, "dev", "hello@github-mp", src)

	out := mustCplugins(t, home, "status")
	assertContains(t, out, "DEV")
	assertNotContains(t, out, "broken")
}

// Case 3: PROJECT column truncation

func TestStatus_projectColumn_truncatesLongBasename(t *testing.T) {
	home := setupHome(t)
	longName := strings.Repeat("x", 40)
	projectDir := filepath.Join(t.TempDir(), longName)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	addInstallEntry(t, home, "hello@test-mp", map[string]any{
		"scope":       "project",
		"projectPath": projectDir,
		"installPath": cacheVersionDir(home, "test-mp", "hello", "0.1.0"),
		"version":     "0.1.0",
	})

	out := mustCpluginsInDir(t, projectDir, home, "status")
	assertNotContains(t, out, longName)
	assertContains(t, out, "..."+strings.Repeat("x", 25))
}

func TestStatus_projectColumn_shortBasenameUnchanged(t *testing.T) {
	home := setupHome(t)
	projectDir := filepath.Join(t.TempDir(), "short-name")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	newMarketplace(t, home, "test-mp", []pluginSpec{{"hello", "0.1.0"}})
	installPlugin(t, home, "hello@test-mp")

	addInstallEntry(t, home, "hello@test-mp", map[string]any{
		"scope":       "project",
		"projectPath": projectDir,
		"installPath": cacheVersionDir(home, "test-mp", "hello", "0.1.0"),
		"version":     "0.1.0",
	})

	out := mustCpluginsInDir(t, projectDir, home, "status")
	assertContains(t, out, "short-name")
	assertNotContains(t, out, "...")
}

func TestStatus_urlMarketplace_ok(t *testing.T) {
	home := setupHome(t)
	injectGitMarketplace(t, home, "url-mp", "url", "https://example.com/plugins.git", nil)

	cachePath := cacheVersionDir(home, "url-mp", "hello", "2.0.0")
	writePluginJSON(t, cachePath, "hello", "2.0.0")
	addInstallEntry(t, home, "hello@url-mp", map[string]any{
		"scope": "user", "projectPath": "",
		"installPath": cachePath, "version": "2.0.0",
	})

	out := mustCplugins(t, home, "status")
	assertContains(t, out, "hello")
	assertContains(t, out, "ok")
	assertNotContains(t, out, "stale")
}

package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// HOME setup
// ---------------------------------------------------------------------------

// setupHome copies the e2e/claude-home/ template to a fresh t.TempDir().
// The template is the .claude directory itself: it lands at <home>/.claude.
func setupHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	// Go tests run with CWD = the package directory (e2e/).
	src := "claude-home"
	dst := filepath.Join(home, ".claude")
	if err := copyDir(src, dst, nil); err != nil {
		t.Fatalf("setupHome: %v", err)
	}
	return home
}

// ---------------------------------------------------------------------------
// Marketplace & plugin fixtures
// ---------------------------------------------------------------------------

type pluginSpec struct {
	name    string
	version string
}

// newMarketplace creates a temp marketplace directory, writes manifests,
// and registers it with 'claude plugin marketplace add'. Returns the mp dir.
func newMarketplace(t *testing.T, home, mpName string, plugins []pluginSpec) string {
	t.Helper()
	mpDir := t.TempDir()

	pluginEntries := make([]map[string]any, len(plugins))
	for i, p := range plugins {
		pluginEntries[i] = map[string]any{
			"name":        p.name,
			"source":      "./plugins/" + p.name,
			"description": "test plugin " + p.name,
			"version":     p.version,
			"author":      map[string]any{"name": "test"},
			"tags":        []string{},
		}
	}
	writeJSON(t, filepath.Join(mpDir, ".claude-plugin", "marketplace.json"), map[string]any{
		"name":    mpName,
		"owner":   map[string]any{"name": "test"},
		"plugins": pluginEntries,
	})
	for _, p := range plugins {
		writePluginJSON(t, filepath.Join(mpDir, "plugins", p.name), p.name, p.version)
	}
	mustClaudeOK(t, home, "plugin", "marketplace", "add", mpDir)
	return mpDir
}

// injectGitMarketplace injects a git marketplace into known_marketplaces.json
// and materializes the local clone under installLocation. sourceType is "github"
// or "url". repoOrURL is the repo slug or the full URL. If plugins is non-empty,
// it writes a plugin.json for each entry under <installLocation>/plugins/<name>/
// (mimicking the clone that Claude Code maintains for this marketplace type).
// Returns the local clone path.
func injectGitMarketplace(t *testing.T, home, mpName, sourceType, repoOrURL string, plugins []pluginSpec) string {
	t.Helper()
	path := filepath.Join(home, ".claude", "plugins", "known_marketplaces.json")
	var mps map[string]any
	readJSON(t, path, &mps)

	cloneDir := filepath.Join(home, ".claude", "plugins", "cache", mpName)
	source := map[string]any{"source": sourceType}
	if sourceType == "github" {
		source["repo"] = repoOrURL
	} else {
		source["url"] = repoOrURL
	}
	mps[mpName] = map[string]any{
		"source":          source,
		"installLocation": cloneDir,
		"autoUpdate":      true,
	}
	writeJSON(t, path, mps)

	for _, p := range plugins {
		writePluginJSON(t, filepath.Join(cloneDir, "plugins", p.name), p.name, p.version)
	}
	return cloneDir
}

// newPluginSource creates a standalone plugin source directory for dev mode.
func newPluginSource(t *testing.T, name, version string) string {
	t.Helper()
	dir := t.TempDir()
	writePluginJSON(t, dir, name, version)
	return dir
}

// newGitPluginSource creates a plugin source directory backed by a git repo on the given branch.
func newGitPluginSource(t *testing.T, name, version, branch string) string {
	t.Helper()
	dir := t.TempDir()
	writePluginJSON(t, dir, name, version)
	runGit(t, dir, "init", "-b", branch)
	return dir
}

// runGit runs a git command in dir and fails the test on error.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// writePluginJSON writes a minimal .claude-plugin/plugin.json into dir.
func writePluginJSON(t *testing.T, dir, name, version string) {
	t.Helper()
	writeJSON(t, filepath.Join(dir, ".claude-plugin", "plugin.json"), map[string]any{
		"name":        name,
		"version":     version,
		"description": "test plugin " + name,
	})
}

// bumpPluginSourceVersion updates only the plugin.json version in a source dir
// (simulates a local code change without updating the marketplace manifest).
func bumpPluginSourceVersion(t *testing.T, sourceDir, newVersion string) {
	t.Helper()
	path := filepath.Join(sourceDir, ".claude-plugin", "plugin.json")
	var pj map[string]any
	readJSON(t, path, &pj)
	pj["version"] = newVersion
	writeJSON(t, path, pj)
}

// ---------------------------------------------------------------------------
// CLI runners
// ---------------------------------------------------------------------------

// runBin executes bin with args, using env as its environment and dir (if
// non-empty) as its working directory. Returns stdout, stderr, error.
func runBin(bin string, env []string, dir string, args ...string) (stdout, stderr string, err error) {
	cmd := exec.Command(bin, args...)
	cmd.Env = env
	cmd.Dir = dir
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return out.String(), errBuf.String(), err
}

// mustSucceed fatals with a labeled message if err is non-nil, otherwise
// returns stdout.
func mustSucceed(t *testing.T, label string, stdout, stderr string, err error) string {
	t.Helper()
	if err != nil {
		t.Fatalf("%s failed: %v\nstdout: %s\nstderr: %s", label, err, stdout, stderr)
	}
	return stdout
}

// runCplugins executes cplugins with the given HOME and returns stdout, stderr, error.
func runCplugins(t *testing.T, home string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	return runBin(cpluginsBin, envWithHome(home), "", args...)
}

// runClaude executes the claude CLI with the given HOME.
func runClaude(t *testing.T, home string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	return runBin(claudeBin, envWithHome(home), "", args...)
}

// mustCplugins runs cplugins and fatals on error. Returns stdout.
func mustCplugins(t *testing.T, home string, args ...string) string {
	t.Helper()
	stdout, stderr, err := runCplugins(t, home, args...)
	return mustSucceed(t, fmt.Sprintf("cplugins %v", args), stdout, stderr, err)
}

// runCpluginsInDir executes cplugins from a specific directory.
func runCpluginsInDir(t *testing.T, dir, home string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	return runBin(cpluginsBin, envWithHome(home), dir, args...)
}

// mustCpluginsInDir runs cplugins from a specific directory and fatals on error. Returns stdout.
func mustCpluginsInDir(t *testing.T, dir, home string, args ...string) string {
	t.Helper()
	stdout, stderr, err := runCpluginsInDir(t, dir, home, args...)
	return mustSucceed(t, fmt.Sprintf("cplugins %v (dir=%s)", args, dir), stdout, stderr, err)
}

// mustClaudeOK runs claude and fatals on error. Returns stdout.
func mustClaudeOK(t *testing.T, home string, args ...string) string {
	t.Helper()
	stdout, stderr, err := runClaude(t, home, args...)
	return mustSucceed(t, fmt.Sprintf("claude %v", args), stdout, stderr, err)
}

// installPlugin calls 'claude plugin install key'.
func installPlugin(t *testing.T, home, key string) {
	t.Helper()
	mustClaudeOK(t, home, "plugin", "install", key)
}

// ---------------------------------------------------------------------------
// Assertions
// ---------------------------------------------------------------------------

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected output to contain %q\ngot:\n%s", substr, s)
	}
}

func assertNotContains(t *testing.T, s, substr string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Errorf("expected output NOT to contain %q\ngot:\n%s", substr, s)
	}
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", path)
	}
}

func assertFileNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Errorf("expected file NOT to exist: %s", path)
	}
}

// assertUnderCache checks that path sits under home's plugin cache directory.
func assertUnderCache(t *testing.T, home, path string) {
	t.Helper()
	cacheDir := filepath.Join(home, ".claude", "plugins", "cache")
	if !strings.HasPrefix(path, cacheDir) {
		t.Errorf("expected %q to be under cache %q", path, cacheDir)
	}
}

// ---------------------------------------------------------------------------
// Filesystem helpers
// ---------------------------------------------------------------------------

func installedPluginsPath(home string) string {
	return filepath.Join(home, ".claude", "plugins", "installed_plugins.json")
}

func cacheVersionDir(home, marketplace, plugin, version string) string {
	return filepath.Join(home, ".claude", "plugins", "cache", marketplace, plugin, version)
}

// installedPluginEntry returns the first install entry for a plugin key, or nil.
func installedPluginEntry(t *testing.T, home, key string) map[string]any {
	t.Helper()
	var data map[string]any
	readJSON(t, installedPluginsPath(home), &data)
	plugins, _ := data["plugins"].(map[string]any)
	entries, _ := plugins[key].([]any)
	if len(entries) == 0 {
		return nil
	}
	entry, _ := entries[0].(map[string]any)
	return entry
}

func installPath(t *testing.T, home, key string) string {
	t.Helper()
	entry := installedPluginEntry(t, home, key)
	if entry == nil {
		return ""
	}
	p, _ := entry["installPath"].(string)
	return p
}

// removeInstalledPlugin removes a plugin's entire entry from installed_plugins.json.
// Used in tests to simulate 'claude plugin uninstall' without going through the CLI.
func removeInstalledPlugin(t *testing.T, home, key string) {
	t.Helper()
	var data map[string]any
	readJSON(t, installedPluginsPath(home), &data)
	plugins, _ := data["plugins"].(map[string]any)
	delete(plugins, key)
	data["plugins"] = plugins
	writeJSON(t, installedPluginsPath(home), data)
}

// addInstallEntry appends a raw install entry to installed_plugins.json for the given key.
func addInstallEntry(t *testing.T, home, key string, entry map[string]any) {
	t.Helper()
	var data map[string]any
	readJSON(t, installedPluginsPath(home), &data)
	plugins, _ := data["plugins"].(map[string]any)
	entries, _ := plugins[key].([]any)
	plugins[key] = append(entries, entry)
	data["plugins"] = plugins
	writeJSON(t, installedPluginsPath(home), data)
}

func copyDir(src, dst string, skip func(rel string) bool) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		if rel == "." {
			return os.MkdirAll(dst, 0o755)
		}
		if skip != nil && skip(rel) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		dstPath := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(dstPath, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, 0o644)
	})
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("writeJSON mkdir: %v", err)
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("writeJSON marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("writeJSON write %s: %v", path, err)
	}
}

func readJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("readJSON %s: %v", path, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("readJSON unmarshal %s: %v", path, err)
	}
}

// envWithHome returns os.Environ() with HOME replaced by the given value.
func envWithHome(home string) []string {
	return append(filterEnv(os.Environ(), "HOME"), "HOME="+home)
}

// envWithHomeAndConfigDir returns envWithHome(home) with CLAUDE_CONFIG_DIR
// also set to configDir.
func envWithHomeAndConfigDir(home, configDir string) []string {
	env := filterEnv(envWithHome(home), "CLAUDE_CONFIG_DIR")
	return append(env, "CLAUDE_CONFIG_DIR="+configDir)
}

// filterEnv returns env with any variable named in keys removed.
func filterEnv(env []string, keys ...string) []string {
	out := make([]string, 0, len(env))
	for _, e := range env {
		drop := false
		for _, k := range keys {
			if strings.HasPrefix(e, k+"=") {
				drop = true
				break
			}
		}
		if !drop {
			out = append(out, e)
		}
	}
	return out
}

// runCpluginsWithEnv executes cplugins with a fully custom environment.
func runCpluginsWithEnv(t *testing.T, env []string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	return runBin(cpluginsBin, env, "", args...)
}

// mustCpluginsWithEnv runs cplugins with a custom environment and fatals on error. Returns stdout.
func mustCpluginsWithEnv(t *testing.T, env []string, args ...string) string {
	t.Helper()
	stdout, stderr, err := runCpluginsWithEnv(t, env, args...)
	return mustSucceed(t, fmt.Sprintf("cplugins %v", args), stdout, stderr, err)
}

// injectMarketplace writes a marketplace entry straight into
// known_marketplaces.json, for any source type — including the ones
// 'claude plugin marketplace add' cannot produce (git-subdir, npm, archive,
// command, or an invented one).
func injectMarketplace(t *testing.T, home, mpName string, source map[string]any, installLocation string, autoUpdate bool) {
	t.Helper()
	path := filepath.Join(home, ".claude", "plugins", "known_marketplaces.json")
	var mps map[string]any
	readJSON(t, path, &mps)
	mps[mpName] = map[string]any{
		"source":          source,
		"installLocation": installLocation,
		"autoUpdate":      autoUpdate,
	}
	writeJSON(t, path, mps)
}

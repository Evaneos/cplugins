package cmd

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Evaneos/cplugins/internal/cache"
	"github.com/Evaneos/cplugins/internal/claude"
	"github.com/spf13/cobra"
)

// claudeDir returns the path to the Claude Code configuration directory:
// $CLAUDE_CONFIG_DIR when set to a non-empty value, ~/.claude otherwise.
func claudeDir() string {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.Getenv("HOME"), ".claude")
	}
	return filepath.Join(home, ".claude")
}

// settingsPaths returns the settings.json chain that determines whether a
// plugin is enabled, in the order Claude Code applies them: user settings,
// user local settings, then project settings and project local settings for
// the project root found by walking up from startDir. No project root found
// yields just the user chain.
func settingsPaths(startDir string) []string {
	paths := []string{
		filepath.Join(claudeDir(), "settings.json"),
		filepath.Join(claudeDir(), "settings.local.json"),
	}
	if root := findProjectRoot(startDir); root != "" {
		paths = append(paths,
			filepath.Join(root, ".claude", "settings.json"),
			filepath.Join(root, ".claude", "settings.local.json"),
		)
	}
	return paths
}

// findProjectRoot walks up from startDir to the nearest ancestor containing a
// .claude directory, mirroring how Claude Code locates a project's settings —
// stopping at the filesystem root if none is found. A .claude directory that
// is Claude Code's own configuration directory never counts as a project
// root, so the walk continues past it.
func findProjectRoot(startDir string) string {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return ""
	}
	cd := claudeDir()
	for {
		candidate := filepath.Join(dir, ".claude")
		if candidate != cd {
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func installedPluginsPath() string {
	return filepath.Join(claudeDir(), "plugins", "installed_plugins.json")
}

func marketplacesPath() string {
	return filepath.Join(claudeDir(), "plugins", "known_marketplaces.json")
}

func cacheBaseDir() string {
	return filepath.Join(claudeDir(), "plugins", "cache")
}

// loadResources loads the shared data sources used by most commands.
func loadResources() (map[string]*claude.Plugin, map[string]*claude.Marketplace, error) {
	plugins, err := claude.ParseInstalledPlugins(installedPluginsPath())
	if err != nil {
		return nil, nil, fmt.Errorf("reading installed plugins: %w", err)
	}
	marketplaces, err := claude.ParseMarketplaces(marketplacesPath())
	if err != nil {
		return nil, nil, fmt.Errorf("reading marketplaces: %w", err)
	}
	return plugins, marketplaces, nil
}

// abbrevPath replaces the home directory prefix with ~ and left-truncates with
// "..." if the result exceeds maxLen characters.
func abbrevPath(path string, maxLen int) string {
	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(path, home) {
		path = "~" + path[len(home):]
	}
	return truncateLeft(path, maxLen)
}

// truncateLeft left-truncates s to maxLen characters, keeping its tail and
// prefixing "..." when truncation occurs — the tail is more discriminating
// than the head for worktree-derived names.
func truncateLeft(s string, maxLen int) string {
	if len(s) > maxLen {
		return "..." + s[len(s)-maxLen+3:]
	}
	return s
}

// sortedKeys returns the keys of a map[string]*claude.Plugin sorted alphabetically.
func sortedKeys(m map[string]*claude.Plugin) []string {
	return slices.Sorted(maps.Keys(m))
}

// devModePath returns the install path of the first install among installs
// whose installPath falls outside cbd, or "" if every install resolves into
// the cache — the plugin is not in dev mode.
func devModePath(cbd string, installs []claude.Install) string {
	for _, inst := range installs {
		if !cache.IsUnderCache(cbd, inst.InstallPath) {
			return inst.InstallPath
		}
	}
	return ""
}

// modeLabel renders the MODE value for a set of installs: "MARKETPLACE", or
// "DEV@" followed by the abbreviated dev-mode install path.
func modeLabel(cbd string, installs []claude.Install) string {
	if path := devModePath(cbd, installs); path != "" {
		return "DEV@" + abbrevPath(path, 30)
	}
	return "MARKETPLACE"
}

// recacheFromMarketplace resolves p's marketplace, verifies it supports
// source-version resolution, recaches p from its marketplace source, and
// points installed_plugins.json's installPath/version at the recached
// location. Returns the new cache path and version.
func recacheFromMarketplace(p *claude.Plugin, marketplaces map[string]*claude.Marketplace) (cachePath, version string, err error) {
	mp, ok := marketplaces[p.Marketplace]
	if !ok {
		return "", "", fmt.Errorf("marketplace %q not found", p.Marketplace)
	}
	if !mp.SupportsSourceVersion() {
		return "", "", fmt.Errorf("plugin %q comes from a %q marketplace; cplugins can't recache it — run `claude plugin install %s` or `claude plugin update %s` instead", p.Key, mp.SourceType, p.Key, p.Key)
	}

	pluginsDir := filepath.Join(mp.LocalRoot(), "plugins")
	cachePath, version, err = cache.RecachePlugin(pluginsDir, p.Name, cacheBaseDir(), p.Marketplace)
	if err != nil {
		return "", "", fmt.Errorf("recaching %s: %w", p.Key, err)
	}

	if err := claude.PatchInstalls(installedPluginsPath(), p.Key, func(_, _, _ string) (string, string) {
		return cachePath, version
	}); err != nil {
		return "", "", fmt.Errorf("updating installPath: %w", err)
	}

	return cachePath, version, nil
}

// pluginKeysWhere returns the keys of the non-official plugins in
// installed_plugins.json for which predicate reports true. A nil predicate
// matches every key.
func pluginKeysWhere(predicate func(*claude.Plugin) bool) ([]string, error) {
	plugins, err := claude.ParseInstalledPlugins(installedPluginsPath())
	if err != nil {
		return nil, err
	}
	var keys []string
	for key, p := range plugins {
		if p.Marketplace == claude.OfficialMarketplace {
			continue
		}
		if predicate == nil || predicate(p) {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

// completeInstalledPlugins provides dynamic shell completion for installed plugin keys.
func completeInstalledPlugins(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	keys, err := pluginKeysWhere(nil)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return keys, cobra.ShellCompDirectiveNoFileComp
}

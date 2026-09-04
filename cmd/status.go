package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/Evaneos/cplugins/internal/cache"
	"github.com/Evaneos/cplugins/internal/claude"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:     "status [plugin@marketplace]",
	Aliases: []string{"ls", "list"},
	Short:   "List installed plugins and their cache health",
	Long: `List installed plugins (excluding claude-plugins-official) with their
marketplace, mode (MARKETPLACE or DEV), cached version, source version and
health. With a plugin key, shows the detailed view for that plugin.

User-scope plugins are always shown; project-scope plugins only when the
current directory is inside their project (or with --all-projects).`,
	Example: `  cplugins status        # list everything (aliases: ls, list)
  cplugins list          # same thing
  cplugins status my-plugin@my-marketplace                   # detailed view
  cplugins status --all-projects                             # include other projects`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeInstalledPlugins,
	RunE:              runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().Bool("all-projects", false, "Show project-scope plugins from all projects, not just the current directory")
}

func runStatus(cmd *cobra.Command, args []string) error {
	plugins, marketplaces, err := loadResources()
	if err != nil {
		return err
	}

	allProjects, _ := cmd.Flags().GetBool("all-projects")
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	if len(args) > 0 {
		return runStatusDetailed(cmd.OutOrStdout(), args[0], plugins, marketplaces, cwd, allProjects)
	}

	cbd := cacheBaseDir()
	enabledPluginsFor := memoizedEnabledPlugins()
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PLUGIN\tSCOPE\tPROJECT\tMARKETPLACE\tMODE\tENABLED\tCACHED\tSOURCE\tHEALTH")

	var dups []string
	dupCounts := map[string]int{}
	keys := sortedKeys(plugins)
	for _, key := range keys {
		p := plugins[key]
		if p.Marketplace == claude.OfficialMarketplace {
			continue
		}

		visible := visibleInstalls(p.Installs, cwd, allProjects)
		if len(visible) == 0 {
			continue
		}

		mode := modeLabel(cbd, visible)

		identityCounts := dupIdentityCounts(visible)
		dupCount := 0
		for _, c := range identityCounts {
			if c > 1 {
				dupCount += c
			}
		}
		if dupCount > 0 {
			dups = append(dups, key)
			dupCounts[key] = dupCount
		}

		for i := range visible {
			inst := &visible[i]
			health, sourceVersion := resolveHealth(p, inst, cbd, marketplaces)
			if identityCounts[dupIdentity(*inst)] > 1 {
				health = "dup"
			}
			enabled := "yes"
			if !claude.IsPluginEnabled(key, enabledPluginsFor(installContextDir(*inst, cwd))) {
				enabled = "no"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				p.Name, inst.Scope, projectName(*inst), p.Marketplace, mode, enabled, inst.Version, sourceVersion, health)
		}
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("writing the plugin table: %w", err)
	}

	if len(dups) > 0 {
		out := cmd.OutOrStdout()
		fmt.Fprintln(out)
		fmt.Fprintln(out, "⚠ Duplicate install entries detected — reinstall to fix:")
		for _, key := range dups {
			fmt.Fprintf(out, "  %s (%d entries): claude plugin install %s\n", key, dupCounts[key], key)
		}
	}

	return nil
}

func runStatusDetailed(w io.Writer, key string, plugins map[string]*claude.Plugin, marketplaces map[string]*claude.Marketplace, cwd string, allProjects bool) error {
	p, ok := plugins[key]
	if !ok {
		return fmt.Errorf("plugin %s not found", key)
	}

	cbd := cacheBaseDir()
	devPath := devModePath(cbd, p.Installs)

	fmt.Fprintf(w, "Plugin:      %s\n", p.Name)
	fmt.Fprintf(w, "Marketplace: %s\n", p.Marketplace)
	fmt.Fprintf(w, "Mode:        %s\n", modeLabel(cbd, p.Installs))
	if devPath != "" {
		fmt.Fprintf(w, "Dev path:    %s\n", devPath)
	}
	fmt.Fprintln(w)

	visible := visibleInstalls(p.Installs, cwd, allProjects)
	for _, inst := range visible {
		health, sourceVersion := resolveHealth(p, &inst, cbd, marketplaces)
		fmt.Fprintf(w, "  Scope:        %s\n", inst.Scope)
		if inst.Scope == claude.ScopeProject {
			fmt.Fprintf(w, "  Project:      %s\n", inst.ProjectPath)
		}
		fmt.Fprintf(w, "  Install path: %s\n", inst.InstallPath)
		fmt.Fprintf(w, "  Version:      %s\n", inst.Version)
		if sourceVersion != "-" {
			fmt.Fprintf(w, "  Source:       %s\n", sourceVersion)
		}
		fmt.Fprintf(w, "  Health:       %s\n", health)
		fmt.Fprintln(w)
	}
	return nil
}

// installContextDir returns the directory whose settings chain governs an
// install's enabled state: the install's own project for a project-scope
// install, cwd for a user-scope one.
func installContextDir(inst claude.Install, cwd string) string {
	if inst.Scope == claude.ScopeProject {
		return inst.ProjectPath
	}
	return cwd
}

// memoizedEnabledPlugins returns a function resolving the merged
// enabledPlugins map for a given context directory, caching the result per
// directory so installs sharing a context reuse the same settings chain.
func memoizedEnabledPlugins() func(dir string) map[string]bool {
	cache := make(map[string]map[string]bool)
	return func(dir string) map[string]bool {
		if merged, ok := cache[dir]; ok {
			return merged
		}
		merged := claude.MergeEnabledPlugins(settingsPaths(dir)...)
		cache[dir] = merged
		return merged
	}
}

// visibleInstalls returns the installs to display based on the CWD.
// - user-scope: always visible
// - project-scope: visible if cwd is inside the project (isAncestorOf(projectPath, cwd)) or if allProjects
func visibleInstalls(installs []claude.Install, cwd string, allProjects bool) []claude.Install {
	out := make([]claude.Install, 0, len(installs))
	for _, inst := range installs {
		if inst.Scope != claude.ScopeProject {
			out = append(out, inst)
			continue
		}
		if allProjects || isAncestorOf(inst.ProjectPath, cwd) {
			out = append(out, inst)
		}
	}
	return out
}

// isAncestorOf returns true if ancestor is an ancestor of path (or equal to path).
func isAncestorOf(ancestor, path string) bool {
	if ancestor == "" {
		return false
	}
	rel, err := filepath.Rel(ancestor, path)
	return err == nil && !strings.HasPrefix(rel, "..")
}

// projectColumnMaxLen bounds the PROJECT column: a worktree name derived
// from a branch name can exceed 100 characters and blow up the table.
const projectColumnMaxLen = 28

// projectName returns the project name (basename of projectPath, truncated to
// projectColumnMaxLen) for a project-scope install, empty for user scope
// — this is the value of the PROJECT column.
func projectName(inst claude.Install) string {
	if inst.Scope != claude.ScopeProject || inst.ProjectPath == "" {
		return ""
	}
	return truncateLeft(filepath.Base(inst.ProjectPath), projectColumnMaxLen)
}

// dupIdentity identifies an install for duplicate detection: two
// installs are only duplicates if they share scope AND projectPath —
// two distinct projects in project scope are not duplicates of each other.
func dupIdentity(inst claude.Install) string {
	return inst.Scope + "\x00" + filepath.Clean(inst.ProjectPath)
}

// dupIdentityCounts counts installs by identity (scope + projectPath);
// an identity present more than once designates a duplicate group.
func dupIdentityCounts(installs []claude.Install) map[string]int {
	counts := make(map[string]int, len(installs))
	for _, inst := range installs {
		counts[dupIdentity(inst)]++
	}
	return counts
}

func resolveHealth(p *claude.Plugin, inst *claude.Install, cacheDir string, marketplaces map[string]*claude.Marketplace) (health, sourceVersion string) {
	health = "ok"
	sourceVersion = "-"

	if _, err := os.Stat(inst.InstallPath); err != nil {
		health = "broken"
	} else if !cache.IsUnderCache(cacheDir, inst.InstallPath) {
		// Dev mode: installPath is outside the cache
		if branch := gitBranch(inst.InstallPath); branch != "" {
			sourceVersion = branch
		}
	} else {
		if mp, ok := marketplaces[p.Marketplace]; ok && mp.SupportsSourceVersion() {
			pluginSourceDir := filepath.Join(mp.LocalRoot(), "plugins", p.Name)
			if sv, err := claude.ResolveSourceVersion(pluginSourceDir); err == nil {
				sourceVersion = sv
				if sv != inst.Version {
					health = "stale"
				}
			}
		}
	}
	return
}

// gitBranch returns the current branch of the git repository containing dir, or "" if
// dir is not in a git repository or if HEAD is in detached mode.
// Walks up the tree to find .git (directory or worktree file).
func gitBranch(dir string) string {
	headPath := findGitHEAD(dir)
	if headPath == "" {
		return ""
	}
	data, err := os.ReadFile(headPath)
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(string(data))
	const prefix = "ref: refs/heads/"
	if !strings.HasPrefix(line, prefix) {
		return ""
	}
	return line[len(prefix):]
}

// findGitHEAD walks up the tree from dir to find the git repository's HEAD
// file. Handles normal repos (.git/HEAD) and worktrees (.git file
// pointing to the gitdir).
func findGitHEAD(dir string) string {
	dir, _ = filepath.Abs(dir)
	for {
		dotgit := filepath.Join(dir, ".git")
		fi, err := os.Lstat(dotgit)
		if err == nil {
			if fi.IsDir() {
				return filepath.Join(dotgit, "HEAD")
			}
			// worktree: .git is a "gitdir: <path>" file
			data, err := os.ReadFile(dotgit)
			if err == nil {
				line := strings.TrimSpace(string(data))
				if after, ok := strings.CutPrefix(line, "gitdir: "); ok {
					if !filepath.IsAbs(after) {
						after = filepath.Join(dir, after)
					}
					return filepath.Join(after, "HEAD")
				}
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

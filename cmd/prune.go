package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/Evaneos/cplugins/internal/claude"
	"github.com/spf13/cobra"
)

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove project-scope installs whose project directory is gone",
	Long: `Prune removes installed_plugins.json entries for scope: project installs
whose projectPath no longer exists on disk — the leftovers of a worktree
workflow, where a plugin gets one install per working directory and nothing
removes the entry once the directory is gone.

Run 'cplugins clean' afterwards to reclaim the cache versions these installs
were pinning.`,
	RunE: runPrune,
}

func init() {
	pruneCmd.Flags().Bool("quiet", false, "suppress output if nothing to prune")
	pruneCmd.Flags().Bool("dry-run", false, "report what would be pruned without doing it")
	rootCmd.AddCommand(pruneCmd)
}

// pruneCandidate identifies one install about to be, or already, pruned.
type pruneCandidate struct {
	Key     string
	Project string
}

func runPrune(cmd *cobra.Command, args []string) error {
	quiet, _ := cmd.Flags().GetBool("quiet")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	plugins, err := claude.ParseInstalledPlugins(installedPluginsPath())
	if err != nil {
		return fmt.Errorf("reading installed plugins: %w", err)
	}

	var pruned []pruneCandidate
	anyFailed := false
	for _, key := range sortedKeys(plugins) {
		p := plugins[key]
		if p.Marketplace == claude.OfficialMarketplace {
			continue
		}

		var candidates []pruneCandidate
		for _, inst := range p.Installs {
			if inst.Scope != claude.ScopeProject {
				continue
			}
			if err := projectStatError(inst); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "⚠ %s: cannot check project %s: %v\n", key, inst.ProjectPath, err)
				continue
			}
			if isDeadProjectInstall(inst) {
				candidates = append(candidates, pruneCandidate{Key: key, Project: projectLabel(inst)})
			}
		}
		if len(candidates) == 0 {
			continue
		}

		if !dryRun {
			if _, err := claude.RemoveInstalls(installedPluginsPath(), key, isDeadProjectInstall); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "⚠ %s: %v\n", key, err)
				anyFailed = true
				continue
			}
		}
		pruned = append(pruned, candidates...)
	}

	if len(pruned) == 0 {
		if anyFailed {
			return fmt.Errorf("no plugin could be pruned")
		}
		if !quiet {
			fmt.Fprintln(cmd.OutOrStdout(), "Nothing to prune.")
		}
		return nil
	}

	for _, c := range pruned {
		if dryRun {
			fmt.Fprintf(cmd.OutOrStdout(), "→ would prune: %s (project %s)\n", c.Key, c.Project)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "→ pruning: %s (project %s)\n", c.Key, c.Project)
		}
	}
	if !dryRun {
		fmt.Fprintln(cmd.OutOrStdout(), "Run `cplugins clean` to reclaim the cache directories these were pinning.")
	}

	return nil
}

// isDeadProjectInstall reports whether inst is a scope: project install whose
// projectPath is empty or confirmed absent (fs.ErrNotExist). A stat error of
// any other kind — a stale mount, a directory it cannot traverse — is
// reported separately by projectStatError.
func isDeadProjectInstall(inst claude.Install) bool {
	if inst.Scope != claude.ScopeProject {
		return false
	}
	if inst.ProjectPath == "" {
		return true
	}
	info, err := os.Stat(inst.ProjectPath)
	if err != nil {
		return errors.Is(err, fs.ErrNotExist)
	}
	return !info.IsDir()
}

// projectStatError returns the error from stat-ing a scope: project install's
// projectPath when it is neither confirmed present nor confirmed absent — a
// stale mount or an unreadable directory, say. Returns nil for a user-scope
// install, an empty projectPath, or a stat that succeeds or reports
// fs.ErrNotExist.
func projectStatError(inst claude.Install) error {
	if inst.Scope != claude.ScopeProject || inst.ProjectPath == "" {
		return nil
	}
	_, err := os.Stat(inst.ProjectPath)
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// projectLabel renders an install's projectPath for the prune output,
// flagging an empty one explicitly.
func projectLabel(inst claude.Install) string {
	if inst.ProjectPath == "" {
		return "<empty projectPath>"
	}
	return inst.ProjectPath
}

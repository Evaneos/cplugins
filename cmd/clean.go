package cmd

import (
	"fmt"

	"github.com/Evaneos/cplugins/internal/cache"
	"github.com/Evaneos/cplugins/internal/claude"
	"github.com/spf13/cobra"
)

type cleanAction struct {
	Plugin      string
	Description string
	IsError     bool
}

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove orphaned cache entries",
	Long:  `Clean removes unreferenced entries from the plugin cache.`,
	RunE:  runClean,
}

func init() {
	cleanCmd.Flags().Bool("quiet", false, "suppress output if nothing to clean")
	cleanCmd.Flags().Bool("dry-run", false, "report what would be done without doing it")
	rootCmd.AddCommand(cleanCmd)
}

func runClean(cmd *cobra.Command, args []string) error {
	quiet, _ := cmd.Flags().GetBool("quiet")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	plugins, err := claude.ParseInstalledPlugins(installedPluginsPath())
	if err != nil {
		return fmt.Errorf("reading installed plugins: %w", err)
	}

	referencedPaths := buildReferencedPaths(plugins)
	actions := cleanOrphans(cacheBaseDir(), referencedPaths, dryRun)

	if len(actions) == 0 {
		if !quiet {
			fmt.Fprintln(cmd.OutOrStdout(), "Everything is clean.")
		}
		return nil
	}

	for _, a := range actions {
		prefix := "→"
		if a.IsError {
			prefix = "⚠"
		}
		name := a.Plugin
		if name != "" {
			name += ": "
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s %s%s\n", prefix, name, a.Description)
	}

	return nil
}

func buildReferencedPaths(plugins map[string]*claude.Plugin) map[string]bool {
	refs := make(map[string]bool)
	for _, plugin := range plugins {
		for _, inst := range plugin.Installs {
			refs[inst.InstallPath] = true
		}
	}
	return refs
}

func cleanOrphans(cacheDir string, referencedPaths map[string]bool, dryRun bool) []cleanAction {
	var actions []cleanAction

	orphans, err := cache.FindOrphans(cacheDir, referencedPaths)
	if err != nil {
		actions = append(actions, cleanAction{
			Description: fmt.Sprintf("failed to scan for orphans: %v", err),
			IsError:     true,
		})
		return actions
	}

	verb := "removing"
	if dryRun {
		verb = "would remove"
	}
	for _, orphan := range orphans {
		desc := fmt.Sprintf("%s orphan: %s", verb, orphan)
		actions = append(actions, cleanAction{Description: desc})
		if !dryRun {
			if err := cache.RemoveEntry(orphan); err != nil {
				actions = append(actions, cleanAction{
					Description: fmt.Sprintf("failed to remove orphan %s: %v", orphan, err),
					IsError:     true,
				})
			}
		}
	}

	return actions
}

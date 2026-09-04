package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Evaneos/cplugins/internal/claude"
	"github.com/spf13/cobra"
)

var devCmd = &cobra.Command{
	Use:   "dev <plugin@marketplace> <path>",
	Short: "Point a plugin's installPath directly to a local source directory",
	Long: `Patch installed_plugins.json so the plugin loads directly from a local
directory instead of the marketplace cache. Changes to the source are
immediately reflected on the next /reload-plugins without any session restart.`,
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: completeDevArgs,
	RunE:              runDev,
}

func init() {
	rootCmd.AddCommand(devCmd)
}

func runDev(cmd *cobra.Command, args []string) error {
	plugins, err := claude.ParseInstalledPlugins(installedPluginsPath())
	if err != nil {
		return fmt.Errorf("reading installed plugins: %w", err)
	}

	key := args[0]
	if _, ok := plugins[key]; !ok {
		return fmt.Errorf("plugin %q not found in installed_plugins.json", key)
	}

	devPath, err := filepath.EvalSymlinks(args[1])
	if err != nil {
		return fmt.Errorf("resolving path %q: %w", args[1], err)
	}

	pluginJSON := filepath.Join(devPath, ".claude-plugin", "plugin.json")
	if _, err := os.Stat(pluginJSON); err != nil {
		return fmt.Errorf("no plugin.json at %s — is this a valid plugin directory?", pluginJSON)
	}

	if err := claude.PatchInstallPaths(installedPluginsPath(), key, func(_, _ string) string {
		return devPath
	}); err != nil {
		return fmt.Errorf("patching installPath: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "dev: %s → %s\n", key, devPath)
	fmt.Fprintln(cmd.OutOrStdout(), "Run /reload-plugins in Claude Code to pick up changes.")
	return nil
}

func completeDevArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	switch len(args) {
	case 0:
		return completeInstalledPlugins(cmd, args, toComplete)
	default:
		return nil, cobra.ShellCompDirectiveDefault
	}
}

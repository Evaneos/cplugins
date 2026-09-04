package cmd

import (
	"fmt"

	"github.com/Evaneos/cplugins/internal/claude"
	"github.com/spf13/cobra"
)

var undevCmd = &cobra.Command{
	Use:               "undev <plugin@marketplace>",
	Short:             "Restore the marketplace version of a dev-mode plugin",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeDevPlugins,
	RunE:              runUndev,
}

func init() {
	rootCmd.AddCommand(undevCmd)
}

func runUndev(cmd *cobra.Command, args []string) error {
	key := args[0]

	plugins, marketplaces, err := loadResources()
	if err != nil {
		return err
	}
	plugin, ok := plugins[key]
	if !ok {
		return fmt.Errorf("plugin %q not found in installed_plugins.json", key)
	}

	if devModePath(cacheBaseDir(), plugin.Installs) == "" {
		return fmt.Errorf("plugin %q is not in dev mode", key)
	}

	cachePath, version, err := recacheFromMarketplace(plugin, marketplaces)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "undev: %s → %s (%s)\n", key, cachePath, version)
	fmt.Fprintln(cmd.OutOrStdout(), "Run /reload-plugins in Claude Code to pick up changes.")
	return nil
}

func completeDevPlugins(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	keys, err := pluginKeysWhere(func(p *claude.Plugin) bool {
		return devModePath(cacheBaseDir(), p.Installs) != ""
	})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return keys, cobra.ShellCompDirectiveNoFileComp
}

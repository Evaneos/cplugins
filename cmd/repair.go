package cmd

import (
	"fmt"
	"os"

	"github.com/Evaneos/cplugins/internal/claude"
	"github.com/spf13/cobra"
)

var repairCmd = &cobra.Command{
	Use:   "repair [plugin@marketplace]",
	Short: "Recache broken plugins from their marketplace source",
	Long: `Repair fixes plugins whose installPath no longer exists by re-copying them
from the marketplace source directory into the cache.

Without arguments, repairs all broken plugins. With a plugin key, repairs only that plugin.`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeBrokenPlugins,
	RunE:              runRepair,
}

func init() {
	rootCmd.AddCommand(repairCmd)
}

func runRepair(cmd *cobra.Command, args []string) error {
	plugins, marketplaces, err := loadResources()
	if err != nil {
		return err
	}

	var targets []*claude.Plugin

	if len(args) > 0 {
		p, ok := plugins[args[0]]
		if !ok {
			return fmt.Errorf("plugin %q not found", args[0])
		}
		targets = append(targets, p)
	} else {
		for _, p := range plugins {
			if p.Marketplace == claude.OfficialMarketplace {
				continue
			}
			targets = append(targets, p)
		}
	}

	repaired := 0
	for _, p := range targets {
		if !isBroken(p) {
			continue
		}

		cachePath, version, err := recacheFromMarketplace(p, marketplaces)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "⚠ %s: %v\n", p.Key, err)
			continue
		}

		fmt.Fprintf(cmd.OutOrStdout(), "→ %s → %s (%s)\n", p.Key, cachePath, version)
		repaired++
	}

	if repaired == 0 && len(args) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Nothing to repair.")
	}

	return nil
}

// isBroken reports whether any of p's installs point at a path that no
// longer exists.
func isBroken(p *claude.Plugin) bool {
	for _, inst := range p.Installs {
		if _, err := os.Stat(inst.InstallPath); err != nil {
			return true
		}
	}
	return false
}

func completeBrokenPlugins(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	keys, err := pluginKeysWhere(isBroken)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return keys, cobra.ShellCompDirectiveNoFileComp
}

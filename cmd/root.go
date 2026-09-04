package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "cplugins",
	Short: "Claude Code plugin cache manager",
	Long: `cplugins manages Claude Code's local plugin cache.

Common tasks:
  see what's installed      cplugins status   (aliases: ls, list)
  develop a marketplace one cplugins dev <plugin@mp> <path>  / cplugins undev <plugin@mp>
  fix a broken plugin       cplugins repair
  clean up dead worktrees   cplugins prune
  drop orphaned cache       cplugins clean

Run "cplugins <command> --help" for details on any command.`,
	Version: version,
}

var completionCmd = &cobra.Command{
	Use:   "completion <bash|zsh|fish>",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for the specified shell.

  bash:  cplugins completion bash > /etc/bash_completion.d/cplugins
  zsh:   cplugins completion zsh > "${fpath[1]}/_cplugins"
  fish:  cplugins completion fish > ~/.config/fish/completions/cplugins.fish`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"bash", "zsh", "fish"},
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletionV2(os.Stdout, true)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		default:
			return fmt.Errorf("unsupported shell: %s", args[0])
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

# cplugins

CLI tool for managing Claude Code's plugin cache.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

> **Unofficial, and it writes to files Claude Code owns.** `cplugins` reads and modifies
> `~/.claude/plugins/installed_plugins.json` and the plugin cache beneath it. Claude Code treats
> those as internal: their format is not a published API and can change in any release. This project
> is not affiliated with, authored by, or endorsed by Anthropic; Claude and Claude Code are
> trademarks of Anthropic, PBC. It comes with no warranty of any kind — see [LICENSE](LICENSE).
> The two commands that delete something, `prune` and `clean`, both take `--dry-run`.

## Why

Claude Code handles the plugin lifecycle on its own: `claude plugin install/update/uninstall/list`
covers install and upgrade, and a marketplace with `autoUpdate` enabled refreshes its installed
plugins in the background, a few minutes after each session starts. `cplugins` does not compete
with any of that. It exists for the three things that lifecycle leaves out.

**Persistent dev mode against a remote marketplace.** To edit a plugin that lives in a GitHub
marketplace, Claude Code offers `--plugin-dir`, which overrides the installed plugin for that one
session and has to be passed at every launch. That does not reach background sessions, subagents,
or the daemon. `cplugins dev` patches `installPath` once in the registry: every session, however it
is started, loads the plugin from your worktree until you run `cplugins undev`.

**Registry and cache hygiene.** Nothing reclaims a cache directory the moment it stops being
referenced — the native sweep marks orphans and removes them about two weeks later. `cplugins clean`
reclaims them now, and `cplugins repair` re-caches a plugin whose `installPath` has vanished, which
is what happens every time a dev worktree is deleted from under it. `cplugins prune` goes one step
earlier: a worktree-per-project-scope-install accumulates one `installed_plugins.json` entry per
worktree, and deleting the worktree leaves the entry behind, pinning a cache version nothing else
still points to — `prune` drops those entries so `clean` can reclaim what they were pinning.

**One health view.** `claude plugin list --json` reports what is installed, where, and whether it is
enabled. It does not tell you whether the cache is behind its marketplace source, whether a dev
source directory still exists, or which install entries are duplicated. `cplugins status` computes
that and labels dev-mode plugins with their source branch.

Non-goals: installing, uninstalling, enabling, disabling, and updating marketplace plugins are
`claude plugin`'s job. `cplugins` never touches `known_marketplaces.json`.

## Install

Build from source:

```bash
git clone https://github.com/Evaneos/cplugins.git
cd cplugins
go install .
```

Go 1.26 or later. The binary lands in `$(go env GOPATH)/bin`.

## Configuration

`cplugins` reads Claude Code's registry from `~/.claude`, or from `$CLAUDE_CONFIG_DIR` when that
variable is set to a non-empty value — the same convention Claude Code itself follows.

## Usage

### `cplugins dev <plugin@marketplace> <path>`

Point a plugin to a local source directory for development. Patches `installPath` in `installed_plugins.json` so Claude Code loads directly from the source directory — no cache copy involved, changes are live on the next `/reload-plugins`.

```bash
cplugins dev my-plugin@my-marketplace ~/src/my-marketplace/plugins/my-plugin
```

### `cplugins undev <plugin@marketplace>`

Restore the marketplace version of a dev-mode plugin. Re-caches the plugin from its marketplace source and points `installPath` back at the cache.

```bash
cplugins undev my-plugin@my-marketplace
```

### `cplugins status [plugin@marketplace]`

Aliases: `cplugins list`, `cplugins ls`.

List installed plugins and their cache health. Without arguments, displays a table of installed plugins (excluding `claude-plugins-official`). User-scope plugins are always shown; project-scope plugins are shown only if their `projectPath` is an ancestor of the current directory.

```bash
cplugins status
```

```
PLUGIN        SCOPE    PROJECT     MARKETPLACE       MODE         ENABLED  CACHED  SOURCE  HEALTH
my-plugin     user                 my-marketplace    DEV@~/sr...  yes      0.3.0   -       ok
other-plugin  user                 local-dev         DEV@~/sr...  yes      0.1.0   -       broken
context7      user                 some-marketplace  MARKETPLACE  no       1.0.0   1.1.0   stale
my-plugin     project  my-project  my-marketplace    MARKETPLACE  yes      1.8.2   1.8.2   ok
```

`PROJECT` is the basename of the install's `projectPath`, truncated to 28 characters (with a leading `...`)
— empty for user-scope installs.

`ENABLED` reflects the `enabledPlugins` setting, read from the settings chain (`~/.claude/settings.json`,
`~/.claude/settings.local.json`, then the current project's `.claude/settings.json` and
`settings.local.json`), the last file to mention a plugin winning. A plugin absent from every file is `yes`.

Use `--all-projects` to show project-scope plugins regardless of the current directory.

With a plugin argument, shows detailed information:

```bash
cplugins status my-plugin@my-marketplace
```

Health states:

- **ok** — source dir accessible (dev mode) or cache matches source (marketplace)
- **stale** — marketplace source has a newer version than cache
- **broken** — source dir was deleted (dev mode)
- **dup** — multiple installs with the same scope and, for project-scope, the same `projectPath` (reinstall to fix)

### `cplugins repair [plugin@marketplace]`

Re-cache plugins whose `installPath` no longer exists (e.g. after a dev worktree was removed) by re-copying them from their marketplace source. Without arguments, repairs all broken plugins; with a plugin key, only that one.

```bash
cplugins repair
```

### `cplugins prune [--quiet] [--dry-run]`

Remove `installed_plugins.json` entries for scope: project installs whose `projectPath` no longer
exists on disk — the leftovers of a worktree workflow, where deleting a worktree never removes the
install entry it left behind. A plugin whose every install is removed loses its entry entirely.
User-scope installs are never touched, and a project-scope install whose `projectPath` still exists
is never touched either, even with a stale cache (that's `status`'s and `repair`'s job).

```bash
# See what would be removed
cplugins prune --dry-run

# Prune
cplugins prune

# Silent mode for automation (only prints if something was removed)
cplugins prune --quiet
```

```
→ pruning: my-plugin@my-marketplace (project ~/src/app-feature-branch)
Run `cplugins clean` to reclaim the cache directories these were pinning.
```

`--dry-run` prefixes each line with `→ would prune:` instead, and omits the `cplugins clean` reminder — nothing
was actually removed yet.

Run `cplugins clean` afterwards to reclaim the cache versions these installs were pinning.

### `cplugins clean [--quiet] [--dry-run]`

Remove orphaned cache entries — cache directories no longer referenced by any installed plugin.
`--dry-run` prefixes each line with `→ would remove orphan:` instead of `→ removing orphan:`.

```bash
# See what would be removed
cplugins clean --dry-run

# Clean
cplugins clean

# Silent mode for automation (only prints if something was removed)
cplugins clean --quiet
```

### `cplugins completion <bash|zsh|fish>`

Generate shell completion script.

```bash
# bash
cplugins completion bash > /etc/bash_completion.d/cplugins

# zsh
cplugins completion zsh > "${fpath[1]}/_cplugins"

# fish
cplugins completion fish > ~/.config/fish/completions/cplugins.fish
```

## Scenarios

### Developing a plugin without publishing it

Use `claude plugin init <name>` to scaffold it.

### Developing a marketplace plugin locally

You are working on `my-plugin` and want to test changes live:

```bash
# Point installPath to your worktree
cplugins dev my-plugin@my-marketplace ~/src/my-marketplace-feat-x/plugins/my-plugin

# ... develop, edit files, /reload-plugins in Claude to pick up changes ...

# Switch to a different worktree
cplugins dev my-plugin@my-marketplace ~/src/my-marketplace/plugins/my-plugin

# Done with dev mode, restore marketplace version
cplugins undev my-plugin@my-marketplace
```

### A plugin is stale

A marketplace with auto-update enabled catches up on its own, a few minutes into a session. This is for the
others — third-party and local marketplaces have auto-update off by default.

```bash
cplugins status
# Shows:  context7  user  some-marketplace  MARKETPLACE  1.0.0  1.1.0  stale

claude plugin update context7@some-marketplace
```

## Use it from Claude Code

`sample-skill/` holds a ready-made Claude Code skill that teaches Claude when and how to run `cplugins`. Copy
`sample-skill/skills/cplugins-management/` into your own plugin's `skills/` directory, or into
`~/.claude/skills/`, and adapt it. It is a template, not a plugin this repository installs.

Its bootstrap hook must be declared in `~/.claude/settings.json`, not inside a plugin — a broken plugin can't
run its own repair hook. Add to `~/.claude/settings.json`:

```json
"hooks": {
  "SessionStart": [
    {
      "hooks": [
        { "type": "command", "command": "cplugins repair >/dev/null 2>&1; true" }
      ]
    }
  ]
}
```

## Shell completion

Dynamic completions are provided for all commands:

- `cplugins dev <TAB>` — installed plugins (excluding official)
- `cplugins dev plugin@mp <TAB>` — directory completion
- `cplugins undev <TAB>` — plugins currently in dev mode
- `cplugins status <TAB>` — all installed plugins (excluding official)

Install completions for your shell:

```bash
# bash (add to ~/.bashrc)
eval "$(cplugins completion bash)"

# zsh (add to ~/.zshrc)
eval "$(cplugins completion zsh)"

# fish
cplugins completion fish | source
```

## Development

```bash
make build       # build the binary and install it
make test        # unit and E2E tests
make e2e-tests   # E2E tests only
```

The E2E suite drives the real `claude` binary against a throwaway `HOME`, so `claude` must be on your
`PATH`. It needs no Anthropic account and makes no network calls.

Design decisions that depend on undocumented Claude Code behaviour are recorded in
[docs/design-notes.md](docs/design-notes.md). Read it before changing how the registry is written.

## License

MIT — see [LICENSE](LICENSE). Copyright (c) 2026 Evaneos.

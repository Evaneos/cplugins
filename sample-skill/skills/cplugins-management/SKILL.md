---
name: cplugins-management
description: Workflow for the `cplugins` CLI, which manages Claude Code's local plugin cache — inspect installed plugins and their health (status, aliases ls/list), develop a marketplace plugin locally (dev/undev), repair a broken cache (repair), prune stale project-scope installs (prune), remove orphaned cache entries (clean). Trigger on "cplugins", "list installed plugins", "broken plugin cache", "plugin missing", "slash command gone after a worktree was deleted", "repair/dev/undev/prune a plugin".
---

# cplugins-management

`cplugins` covers what the `claude plugin` CLI doesn't: persistent local development of a marketplace plugin, and hygiene of the plugin registry and cache. Install, uninstall, enable, disable and update stay the job of `claude plugin`.

## Where to start

To see what's installed and its health: `cplugins status` (aliases `list`, `ls`). With a `<plugin@marketplace>` argument, it shows a detailed view of that plugin.

`cplugins --help` lists every command, `cplugins <command> --help` details one.

## Which command for what

- See plugin state / health -> `cplugins status` (aliases `ls`, `list`)
- Develop a marketplace plugin locally -> `cplugins dev <plugin@marketplace> <path>`, then `cplugins undev <plugin@marketplace>`
- Broken cache (plugin missing, see symptoms below) -> `cplugins repair`
- Registry cluttered by project-scope installs of deleted worktrees -> `cplugins prune`, then `cplugins clean`
- Remove orphaned cache entries -> `cplugins clean`
- Install, uninstall, enable, disable, update -> `claude plugin install` / `uninstall` / `enable` / `disable` / `update`

After any change: run `/reload-plugins` in Claude Code to pick it up.

## Developing a marketplace plugin locally

```bash
# path = the plugin's own directory (the one containing plugin.json), not the marketplace root
cplugins dev my-plugin@my-marketplace ~/src/my-plugin
# ... edit, /reload-plugins ...
cplugins undev my-plugin@my-marketplace   # recache from the marketplace source
```

`cplugins status <plugin@marketplace>` shows the mode (`DEV`/`MARKETPLACE`) and, in dev mode, the source's git branch.

## Broken cache

Symptoms:

- `which <binary>` resolves to a path that no longer exists (typical after cleaning up a worktree that was the target of a `cplugins dev`).
- A slash command is missing even though the plugin is enabled.
- A hook never fires.

Run `cplugins repair` (with no argument it repairs every broken plugin; with a `<plugin@marketplace>` argument, just that one).

## Cluttered registry

Claude Code records one project-scope install per working directory. A worktree-based workflow accumulates one per worktree, and nothing removes the entry once the worktree is gone — these pin old cache versions in place.

```bash
cplugins prune --dry-run   # installs whose projectPath no longer exists
cplugins prune
cplugins clean             # reclaim the cache versions those installs were pinning
```

## Bootstrap hook

The hook `hooks/repair-on-start.sh` (SessionStart, runs `cplugins repair`) must be declared in `~/.claude/settings.json`, not inside a plugin — a broken plugin can't run its own repair hook.

Installation: add to `~/.claude/settings.json`:

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

Equivalent, pointing at the script that ships with this skill:

```json
{
  "type": "command",
  "command": "<plugin_root>/skills/cplugins-management/hooks/repair-on-start.sh"
}
```

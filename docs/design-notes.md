# Design notes

Facts about Claude Code that this tool depends on, and the design choices that follow from them.
They are recorded here because they are not obvious from the code, and a contributor who does not
know them is likely to "simplify" the tool back into a bug. Each was verified against Claude Code
2.1.259 on the date noted; re-check them before relying on one.

## The registry carries fields this tool does not model

`~/.claude/plugins/installed_plugins.json` holds, per install entry, at least `scope`,
`projectPath`, `installPath` and `version` — plus `lastUpdated`, `installedAt` and `gitCommitSha`,
which Claude Code writes and uses (the commit SHA feeds release-tag resolution). More may appear.

Consequence: writes must never round-trip the file through a typed struct. `internal/claude/
installed_write.go` decodes every object into an order-preserving map of raw JSON and rewrites only
the keys it actually changes. A typed round-trip silently strips the unmodelled fields from **every**
entry in the file, not only the one being edited.

Writes go through a temporary file in the same directory followed by a rename, because Claude Code
writes this same file from its own background update a few minutes into each session. The existing
file's permission bits are carried over rather than forced to `0644`.

## Claude Code does update installed plugins on its own

A marketplace with `autoUpdate` enabled gets both its catalog refreshed _and_ its installed plugins
updated on disk, in the background, shortly after a session starts. `autoUpdate` is on by default for
Anthropic's own marketplaces and off by default for third-party and local ones.

Consequence: staleness detection is only useful for marketplaces where auto-update is off. This tool
does not update plugins; `claude plugin update` does.

## `claude plugin list --json` is authoritative but slow

It reports `installPath`, `version`, `scope`, `projectPath`, `enabled` and `lastUpdated` — everything
`status` reads from the registry, plus the enabled flag. It costs roughly one second per invocation,
which rules it out for `status` and, above all, for shell completion, which parses the registry on
every keypress.

Consequence: `status` reads the files directly and resolves the enabled flag from the `enabledPlugins`
settings chain instead. When diagnosing a disagreement between this tool and Claude Code, the CLI's
JSON is the reference.

## A marketplace's local layout depends on its source type

For `directory`, `github`, `url` and `git-subdir` marketplaces, `LocalRoot()` holds the marketplace
tree, and each plugin's source sits where its entry in `.claude-plugin/marketplace.json` says: a
relative path from that root (`./plugins/<name>`, `./<name>`, or `./` for a repository that is its own
marketplace), or a bare name under `metadata.pluginRoot`. For `npm`, `archive`, `command` and any
unrecognized source type, `installLocation` does not point at such a tree.

Consequence: `Marketplace.SupportsSourceVersion()` gates every code path that resolves a plugin source
under `LocalRoot()`, and `claude.PluginSourceDir` is the only place that resolves it. An entry whose
source is an object (`github`, `git-subdir`…) is resolved outside the marketplace tree. In both cases
version resolution degrades to `SOURCE -`, and `undev` and `repair` refuse, since caching the wrong
tree would repoint `installPath` at it.

## Project-scope installs accumulate one entry per working directory

Installing a plugin at project scope records an entry keyed by the working directory it was installed
from. A workflow based on git worktrees therefore produces one entry per worktree, and deleting the
worktree leaves the entry behind, pinning the cache version it referenced. Nothing in Claude Code
reclaims those: `claude plugin prune` targets unused auto-installed dependencies, and the background
cache sweep only removes directories no entry references.

Consequence: `prune`, and the ordering of `prune` before `clean`.

## `claude plugin validate` needs a `skills/<name>/` subtree

Pointed at a directory, `validate` only enters component-scanning mode when that directory contains a
`skills/<name>/SKILL.md` (or `agents/`, `commands/`) subtree. A `SKILL.md` sitting at the directory's
own root is treated as a plugin-manifest target and fails with `No manifest found in directory`.

Consequence: the shape of `sample-skill/`.

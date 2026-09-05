# cplugins

`CLAUDE.md` is a symlink to this file.

Go CLI for managing Claude Code's plugin cache.

## Build & test

```bash
make build          # build binary, version stamped from git describe
make test           # unit and E2E tests
make e2e-tests      # E2E tests (require claude in PATH, no auth needed)
golangci-lint run   # lint
```

CI runs build, vet, gofmt, lint, `govulncheck` and the tests, minus the `e2e` package — no runner has the `claude` binary. Anything the E2E suite alone covers is unguarded there.

## Project structure

```
cmd/           # cobra commands (root, dev, undev, status, prune, clean, helpers)
internal/
  claude/      # parsers for installed_plugins.json, known_marketplaces.json, plugin.json; PatchInstallPaths/PatchInstalls/RemoveInstalls writers
  cache/       # orphan detection, entry removal, recache from marketplace source
e2e/
  claude-home/ # template HOME for E2E tests (credentials gitignored)
  *_test.go    # E2E test suite, no auth required
```

## Commands

- `dev <plugin@marketplace> <path>` — patch `installPath` to a local source dir
- `undev <plugin@marketplace>` — re-cache the plugin from its marketplace source and point `installPath` back at the cache
- `status [plugin@marketplace]` (aliases `ls`, `list`) — list plugins and their health; user-scope always visible, project-scope filtered by CWD
- `repair [plugin@marketplace]` — re-cache plugins whose `installPath` no longer exists
- `prune` — remove project-scope installs whose `projectPath` no longer exists, dropping the plugin's key entirely once none are left
- `clean` — remove orphaned cache entries

## Release process

- `release-please` (release-type `go`) runs on every push to `main`, maintaining a Release PR with `CHANGELOG.md` and the next version bump computed from commit history
- Commits and PR titles must follow Conventional Commits (`feat:`, `fix:`, `chore:`, …) — release-please can't decide a bump or write the changelog otherwise
- Merging the Release PR tags the release and creates the GitHub Release; the tag push triggers `.github/workflows/release.yaml`, which builds and attaches binaries via `goreleaser` (`.goreleaser.yaml`)

## Key conventions

- `dev` resolves symlinks at write time (`filepath.EvalSymlinks`) to store canonical paths; `undev` re-caches from the marketplace source
- `installed_plugins.json` installPath patches are via `claude.PatchInstallPaths`/`claude.PatchInstalls`; install removal is via `claude.RemoveInstalls`
- Never modify `known_marketplaces.json` — delegate to `claude plugin marketplace add/remove`
- `claude-plugins-official` plugins are always skipped (managed by Claude Code)
- `claudeDir()` (`cmd/helpers.go`) resolves `$CLAUDE_CONFIG_DIR` when non-empty, else `~/.claude` — every path helper (`installedPluginsPath`, `marketplacesPath`, `cacheBaseDir`, `settingsPaths`) goes through it
- `status`'s `ENABLED` column reads `enabledPlugins` from the settings chain (`claude.MergeEnabledPlugins`/`claude.IsPluginEnabled`) — `claude plugin list --json` exposes the same field but costs ~1s per call, prohibitive for `status` and for shell completion; a key absent from every settings file is enabled
- `claude.Marketplace.SupportsSourceVersion()` gates source-version resolution in `resolveHealth` (`cmd/status.go`) — only `directory`, `github`, `url`, and `git-subdir` are resolvable; other source types (`npm`, `archive`, `command`, or anything unrecognized) report `SOURCE -` without attempting resolution
- Tests use stdlib `testing` only (no external test frameworks)
- Any behavior change (addition, modification, removal) must come with the corresponding E2E test

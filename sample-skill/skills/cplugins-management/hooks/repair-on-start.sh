#!/usr/bin/env bash
# SessionStart hook: repair the cplugins cache if it's broken.
#
# Must be declared in `~/.claude/settings.json`, not inside a plugin — this
# hook exists to fix broken plugins, and a hook shipped inside a broken
# plugin wouldn't run to fix it.
#
# Installation: see SKILL.md.
set -euo pipefail
command -v cplugins >/dev/null 2>&1 || exit 0
cplugins repair >/dev/null 2>&1 </dev/null || true
exit 0

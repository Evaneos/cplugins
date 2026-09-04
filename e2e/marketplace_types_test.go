package e2e_test

// marketplace_types_test.go covers unresolvable marketplace source types
// (npm, archive, command, and anything unrecognized): status lists them with
// SOURCE "-" and HEALTH ok, alongside git-subdir, which resolves.

import (
	"path/filepath"
	"regexp"
	"testing"
)

// assertPluginRow checks that out contains a user-scope MARKETPLACE row for
// key@marketplace with the given SOURCE value and HEALTH ok.
func assertPluginRow(t *testing.T, out, key, marketplace, source string) {
	t.Helper()
	pattern := regexp.MustCompile(
		regexp.QuoteMeta(key) + `\s+user\s+` + regexp.QuoteMeta(marketplace) +
			`\s+MARKETPLACE\s+yes\s+\S+\s+` + regexp.QuoteMeta(source) + `\s+ok`)
	if !pattern.MatchString(out) {
		t.Errorf("expected a row for %s@%s with SOURCE=%q HEALTH=ok, got:\n%s", key, marketplace, source, out)
	}
}

func TestStatus_newSourceTypes_noCrashNoFalseHealth(t *testing.T) {
	home := setupHome(t)

	cases := []struct {
		key    string
		mp     string
		source map[string]any
		// wantSource is the expected SOURCE column: a resolvable type gets its
		// version back, everything else gets "-".
		wantSource string
	}{
		{
			key: "gitsubdir-plugin", mp: "gitsubdir-mp",
			source:     map[string]any{"source": "git-subdir", "repo": "org/monorepo", "path": "plugins/gitsubdir-plugin"},
			wantSource: "1.0.0",
		},
		{
			key: "npm-plugin", mp: "npm-mp",
			source:     map[string]any{"source": "npm", "package": "@org/npm-plugin"},
			wantSource: "-",
		},
		{
			key: "archive-plugin", mp: "archive-mp",
			source:     map[string]any{"source": "archive", "url": "https://example.com/plugin.zip"},
			wantSource: "-",
		},
		{
			key: "command-plugin", mp: "command-mp",
			source:     map[string]any{"source": "command", "command": "print-plugin-dir", "mode": "copy"},
			wantSource: "-",
		},
		{
			key: "future-plugin", mp: "future-mp",
			source:     map[string]any{"source": "some-future-type"},
			wantSource: "-",
		},
	}

	for _, c := range cases {
		installLocation := filepath.Join(home, ".claude", "plugins", "cache", c.mp)
		writePluginJSON(t, filepath.Join(installLocation, "plugins", c.key), c.key, "1.0.0")
		injectMarketplace(t, home, c.mp, c.source, installLocation, false)

		installPath := cacheVersionDir(home, c.mp, c.key, "1.0.0")
		writePluginJSON(t, installPath, c.key, "1.0.0")
		addInstallEntry(t, home, c.key+"@"+c.mp, map[string]any{
			"scope": "user", "projectPath": "",
			"installPath": installPath, "version": "1.0.0",
		})
	}

	out := mustCplugins(t, home, "status")
	assertNotContains(t, out, "broken")
	assertNotContains(t, out, "stale")
	for _, c := range cases {
		assertPluginRow(t, out, c.key, c.mp, c.wantSource)
	}
}

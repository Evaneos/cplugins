package cache

import "testing"

func TestIsUnderCache(t *testing.T) {
	cases := []struct {
		cacheDir    string
		installPath string
		want        bool
	}{
		{"/srv/.claude/plugins/cache", "/srv/.claude/plugins/cache/mp/hello/0.1.0", true},
		{"/srv/.claude/plugins/cache", "/srv/work/my-plugin", false},
		{"/srv/.claude/plugins/cache", "/srv/.claude/plugins/cache", true},
		{"/srv/.claude/plugins/cache", "/srv/.claude/plugins/cachemore", false},
	}
	for _, c := range cases {
		got := IsUnderCache(c.cacheDir, c.installPath)
		if got != c.want {
			t.Errorf("IsUnderCache(%q, %q) = %v, want %v", c.cacheDir, c.installPath, got, c.want)
		}
	}
}

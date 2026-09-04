package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTruncateLeft(t *testing.T) {
	long := "this-name-is-way-too-long-for-the-column"
	cases := []struct {
		s      string
		maxLen int
		want   string
	}{
		{"short", 28, "short"},
		{"exactly-28-characters-long!!", 28, "exactly-28-characters-long!!"},
		{long, 28, "..." + long[len(long)-25:]},
	}

	for _, c := range cases {
		got := truncateLeft(c.s, c.maxLen)
		if got != c.want {
			t.Errorf("truncateLeft(%q, %d) = %q, want %q", c.s, c.maxLen, got, c.want)
		}
		if len(got) > c.maxLen {
			t.Errorf("truncateLeft(%q, %d) = %q, exceeds maxLen", c.s, c.maxLen, got)
		}
	}
}

func TestClaudeDir_defaultsToHomeDotClaude(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	home := t.TempDir()
	t.Setenv("HOME", home)

	want := filepath.Join(home, ".claude")
	if got := claudeDir(); got != want {
		t.Errorf("claudeDir() = %q, want %q", got, want)
	}
}

func TestClaudeDir_respectsEnvVar(t *testing.T) {
	custom := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", custom)

	if got := claudeDir(); got != custom {
		t.Errorf("claudeDir() = %q, want %q", got, custom)
	}
}

func TestClaudeDir_emptyEnvVarBehavesAsAbsent(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	home := t.TempDir()
	t.Setenv("HOME", home)

	want := filepath.Join(home, ".claude")
	if got := claudeDir(); got != want {
		t.Errorf("claudeDir() = %q, want %q", got, want)
	}
}

func TestFindProjectRoot_walksUpToAncestorWithClaudeDir(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	home := t.TempDir()
	t.Setenv("HOME", home)

	project := filepath.Join(home, "work", "myrepo")
	sub := filepath.Join(project, "internal", "api")
	if err := os.MkdirAll(filepath.Join(project, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	if got := findProjectRoot(sub); got != project {
		t.Errorf("findProjectRoot(%q) = %q, want %q", sub, got, project)
	}
}

func TestFindProjectRoot_noAncestorFound(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir := t.TempDir()
	if got := findProjectRoot(dir); got != "" {
		t.Errorf("findProjectRoot(%q) = %q, want \"\"", dir, got)
	}
}

func TestFindProjectRoot_neverMatchesClaudeConfigDirItself(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}

	// home/.claude is Claude Code's own configuration directory, not a
	// project root — even though startDir is home itself.
	if got := findProjectRoot(home); got != "" {
		t.Errorf("findProjectRoot(home) = %q, want \"\" (must not match its own config dir)", got)
	}
}

func TestSettingsPaths_includesProjectChainWhenFound(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	home := t.TempDir()
	t.Setenv("HOME", home)

	project := filepath.Join(home, "work", "myrepo")
	if err := os.MkdirAll(filepath.Join(project, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}

	want := []string{
		filepath.Join(home, ".claude", "settings.json"),
		filepath.Join(home, ".claude", "settings.local.json"),
		filepath.Join(project, ".claude", "settings.json"),
		filepath.Join(project, ".claude", "settings.local.json"),
	}
	got := settingsPaths(project)
	if len(got) != len(want) {
		t.Fatalf("settingsPaths(%q) = %v, want %v", project, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("settingsPaths(%q)[%d] = %q, want %q", project, i, got[i], want[i])
		}
	}
}

func TestSettingsPaths_userOnlyWhenNoProjectRoot(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir := t.TempDir()
	want := []string{
		filepath.Join(home, ".claude", "settings.json"),
		filepath.Join(home, ".claude", "settings.local.json"),
	}
	got := settingsPaths(dir)
	if len(got) != len(want) {
		t.Fatalf("settingsPaths(%q) = %v, want %v", dir, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("settingsPaths(%q)[%d] = %q, want %q", dir, i, got[i], want[i])
		}
	}
}

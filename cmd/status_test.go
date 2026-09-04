package cmd

import (
	"testing"

	"github.com/Evaneos/cplugins/internal/claude"
)

func TestProjectName(t *testing.T) {
	cases := []struct {
		name string
		inst claude.Install
		want string
	}{
		{"user scope → empty", claude.Install{Scope: claude.ScopeUser, ProjectPath: ""}, ""},
		{"project scope → basename", claude.Install{Scope: claude.ScopeProject, ProjectPath: "/srv/work/foo"}, "foo"},
		{"project scope without projectPath → empty", claude.Install{Scope: claude.ScopeProject, ProjectPath: ""}, ""},
	}

	for _, c := range cases {
		got := projectName(c.inst)
		if got != c.want {
			t.Errorf("%s: projectName(%+v) = %q, want %q", c.name, c.inst, got, c.want)
		}
	}
}

func TestDupIdentity(t *testing.T) {
	a := claude.Install{Scope: claude.ScopeProject, ProjectPath: "/srv/work/foo"}
	aTrailingSlash := claude.Install{Scope: claude.ScopeProject, ProjectPath: "/srv/work/foo/"}
	b := claude.Install{Scope: claude.ScopeProject, ProjectPath: "/srv/work/bar"}
	user := claude.Install{Scope: claude.ScopeUser, ProjectPath: ""}

	if dupIdentity(a) != dupIdentity(aTrailingSlash) {
		t.Errorf("dupIdentity should normalize a trailing slash: %q != %q", dupIdentity(a), dupIdentity(aTrailingSlash))
	}
	if dupIdentity(a) == dupIdentity(b) {
		t.Errorf("distinct projectPaths should not collide: %q == %q", dupIdentity(a), dupIdentity(b))
	}
	if dupIdentity(a) == dupIdentity(user) {
		t.Errorf("distinct scopes should not collide: %q == %q", dupIdentity(a), dupIdentity(user))
	}
}

func TestIsAncestorOf(t *testing.T) {
	cases := []struct {
		ancestor string
		path     string
		want     bool
	}{
		{"/foo", "/foo", true},         // equal
		{"/foo", "/foo/bar", true},     // direct subdirectory
		{"/foo", "/foo/bar/baz", true}, // deep subdirectory
		{"/foo", "/foobar", false},     // string prefix but not an ancestor
		{"/foo", "/bar", false},        // different directory
		{"/foo", "/fo", false},         // shorter path
		{"", "/foo", false},            // empty ancestor
		{"/foo/bar", "/foo", false},    // parent ≠ ancestor
	}

	for _, c := range cases {
		got := isAncestorOf(c.ancestor, c.path)
		if got != c.want {
			t.Errorf("isAncestorOf(%q, %q) = %v, want %v", c.ancestor, c.path, got, c.want)
		}
	}
}

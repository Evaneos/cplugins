package e2e_test

import "testing"

// add and remove are not cplugins commands: installing a plugin without a
// marketplace is Claude Code's job (claude plugin init, or a self-marketplace).
func TestUnknownCommand_Add(t *testing.T) {
	home := setupHome(t)
	_, stderr, err := runCplugins(t, home, "add", "/some/path")
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
	assertContains(t, stderr, "unknown command")
}

func TestUnknownCommand_Remove(t *testing.T) {
	home := setupHome(t)
	_, stderr, err := runCplugins(t, home, "remove", "some-plugin@some-marketplace")
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
	assertContains(t, stderr, "unknown command")
}

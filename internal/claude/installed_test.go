package claude

import (
	"path/filepath"
	"testing"
)

func TestParseInstalledPlugins(t *testing.T) {
	path := filepath.Join("testdata", "installed_plugins.json")
	plugins, err := ParseInstalledPlugins(path)
	if err != nil {
		t.Fatalf("ParseInstalledPlugins: %v", err)
	}

	if got := len(plugins); got != 2 {
		t.Fatalf("expected 2 plugins, got %d", got)
	}

	// Check hello-world plugin
	hw, ok := plugins["hello-world@my-marketplace"]
	if !ok {
		t.Fatal("missing plugin hello-world@my-marketplace")
	}
	if hw.Name != "hello-world" {
		t.Errorf("Name = %q, want %q", hw.Name, "hello-world")
	}
	if hw.Marketplace != "my-marketplace" {
		t.Errorf("Marketplace = %q, want %q", hw.Marketplace, "my-marketplace")
	}
	if hw.Key != "hello-world@my-marketplace" {
		t.Errorf("Key = %q, want %q", hw.Key, "hello-world@my-marketplace")
	}
	if len(hw.Installs) != 1 {
		t.Fatalf("hello-world: expected 1 install, got %d", len(hw.Installs))
	}
	if hw.Installs[0].Scope != "user" {
		t.Errorf("hello-world install scope = %q, want %q", hw.Installs[0].Scope, "user")
	}
	if hw.Installs[0].Version != "1.0.0" {
		t.Errorf("hello-world install version = %q, want %q", hw.Installs[0].Version, "1.0.0")
	}

	// Check multi-scope plugin (2 installs: user + project)
	ms, ok := plugins["multi-scope@other-market"]
	if !ok {
		t.Fatal("missing plugin multi-scope@other-market")
	}
	if ms.Name != "multi-scope" {
		t.Errorf("Name = %q, want %q", ms.Name, "multi-scope")
	}
	if ms.Marketplace != "other-market" {
		t.Errorf("Marketplace = %q, want %q", ms.Marketplace, "other-market")
	}
	if len(ms.Installs) != 2 {
		t.Fatalf("multi-scope: expected 2 installs, got %d", len(ms.Installs))
	}
	if ms.Installs[0].Scope != "user" {
		t.Errorf("multi-scope install[0] scope = %q, want %q", ms.Installs[0].Scope, "user")
	}
	if ms.Installs[1].Scope != "project" {
		t.Errorf("multi-scope install[1] scope = %q, want %q", ms.Installs[1].Scope, "project")
	}
	if ms.Installs[1].ProjectPath != "/srv/myproject" {
		t.Errorf("multi-scope install[1] projectPath = %q, want %q", ms.Installs[1].ProjectPath, "/srv/myproject")
	}
}

func TestParseInstalledPlugins_FileNotFound(t *testing.T) {
	_, err := ParseInstalledPlugins("/nonexistent/path/installed_plugins.json")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

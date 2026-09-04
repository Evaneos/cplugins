package claude

import (
	"path/filepath"
	"testing"
)

func TestParseMarketplaces(t *testing.T) {
	path := filepath.Join("testdata", "known_marketplaces.json")
	marketplaces, err := ParseMarketplaces(path)
	if err != nil {
		t.Fatalf("ParseMarketplaces() error = %v", err)
	}

	if len(marketplaces) != 3 {
		t.Fatalf("expected 3 marketplaces, got %d", len(marketplaces))
	}

	// Directory source
	local, ok := marketplaces["local-dev"]
	if !ok {
		t.Fatal("missing marketplace 'local-dev'")
	}
	if local.Name != "local-dev" {
		t.Errorf("Name = %q, want %q", local.Name, "local-dev")
	}
	if local.SourceType != "directory" {
		t.Errorf("SourceType = %q, want %q", local.SourceType, "directory")
	}
	if local.SourcePath != "/srv/work/local-marketplace" {
		t.Errorf("SourcePath = %q, want %q", local.SourcePath, "/srv/work/local-marketplace")
	}
	if local.SourceRepo != "" {
		t.Errorf("SourceRepo = %q, want empty", local.SourceRepo)
	}
	if !local.IsDirectory() {
		t.Error("IsDirectory() = false, want true")
	}
	if !local.AutoUpdate {
		t.Error("AutoUpdate = false, want true")
	}
	if local.InstallLocation != "/srv/.claude/marketplace/local-dev" {
		t.Errorf("InstallLocation = %q, want %q", local.InstallLocation, "/srv/.claude/marketplace/local-dev")
	}

	// GitHub source
	remote, ok := marketplaces["remote-marketplace"]
	if !ok {
		t.Fatal("missing marketplace 'remote-marketplace'")
	}
	if remote.Name != "remote-marketplace" {
		t.Errorf("Name = %q, want %q", remote.Name, "remote-marketplace")
	}
	if remote.SourceType != "github" {
		t.Errorf("SourceType = %q, want %q", remote.SourceType, "github")
	}
	if remote.SourceRepo != "org/remote-marketplace" {
		t.Errorf("SourceRepo = %q, want %q", remote.SourceRepo, "org/remote-marketplace")
	}
	if remote.SourcePath != "" {
		t.Errorf("SourcePath = %q, want empty", remote.SourcePath)
	}
	if remote.IsDirectory() {
		t.Error("IsDirectory() = true, want false")
	}
	if remote.AutoUpdate {
		t.Error("AutoUpdate = true, want false")
	}
}

func TestParseMarketplaces_FileNotFound(t *testing.T) {
	_, err := ParseMarketplaces("/nonexistent/path/known_marketplaces.json")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestParseMarketplaces_urlSource(t *testing.T) {
	path := filepath.Join("testdata", "known_marketplaces.json")
	marketplaces, err := ParseMarketplaces(path)
	if err != nil {
		t.Fatalf("ParseMarketplaces() error = %v", err)
	}

	mp, ok := marketplaces["url-marketplace"]
	if !ok {
		t.Fatal("missing marketplace 'url-marketplace'")
	}
	if mp.SourceType != "url" {
		t.Errorf("SourceType = %q, want %q", mp.SourceType, "url")
	}
	if mp.SourceURL != "https://example.com/plugins.git" {
		t.Errorf("SourceURL = %q, want %q", mp.SourceURL, "https://example.com/plugins.git")
	}
	if mp.SourceRepo != "" {
		t.Errorf("SourceRepo = %q, want empty", mp.SourceRepo)
	}
	if mp.IsDirectory() {
		t.Error("IsDirectory() = true, want false")
	}
	if !mp.IsGit() {
		t.Error("IsGit() = false, want true")
	}
}

func TestLocalRoot(t *testing.T) {
	path := filepath.Join("testdata", "known_marketplaces.json")
	marketplaces, err := ParseMarketplaces(path)
	if err != nil {
		t.Fatalf("ParseMarketplaces() error = %v", err)
	}

	cases := []struct {
		name string
		want string
	}{
		{"local-dev", "/srv/work/local-marketplace"},
		{"remote-marketplace", "/srv/.claude/marketplace/remote-marketplace"},
		{"url-marketplace", "/srv/.claude/marketplace/url-marketplace"},
	}
	for _, tc := range cases {
		mp := marketplaces[tc.name]
		if got := mp.LocalRoot(); got != tc.want {
			t.Errorf("%s: LocalRoot() = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestSupportsSourceVersion(t *testing.T) {
	cases := []struct {
		sourceType string
		want       bool
	}{
		{SourceTypeDirectory, true},
		{SourceTypeGitHub, true},
		{SourceTypeURL, true},
		{SourceTypeGitSubdir, true},
		{SourceTypeNpm, false},
		{SourceTypeArchive, false},
		{SourceTypeCommand, false},
		{"some-future-type", false},
	}
	for _, tc := range cases {
		mp := &Marketplace{SourceType: tc.sourceType, InstallLocation: "/wherever"}
		if got := mp.SupportsSourceVersion(); got != tc.want {
			t.Errorf("SupportsSourceVersion() for %q = %v, want %v", tc.sourceType, got, tc.want)
		}
	}
}

func TestLocalRoot_unresolvableTypes(t *testing.T) {
	for _, sourceType := range []string{SourceTypeGitSubdir, SourceTypeNpm, SourceTypeArchive, SourceTypeCommand, "some-future-type"} {
		mp := &Marketplace{SourceType: sourceType, InstallLocation: "/install/location"}
		if got := mp.LocalRoot(); got != "/install/location" {
			t.Errorf("%s: LocalRoot() = %q, want %q", sourceType, got, "/install/location")
		}
	}
}

func TestIsGit(t *testing.T) {
	path := filepath.Join("testdata", "known_marketplaces.json")
	marketplaces, err := ParseMarketplaces(path)
	if err != nil {
		t.Fatalf("ParseMarketplaces() error = %v", err)
	}

	cases := []struct {
		name      string
		wantIsGit bool
	}{
		{"local-dev", false},
		{"remote-marketplace", true},
		{"url-marketplace", true},
	}
	for _, tc := range cases {
		mp := marketplaces[tc.name]
		if got := mp.IsGit(); got != tc.wantIsGit {
			t.Errorf("%s: IsGit() = %v, want %v", tc.name, got, tc.wantIsGit)
		}
	}
}

package claude

import (
	"encoding/json"
	"fmt"
	"os"
)

// Source type values used in known_marketplaces.json.
const (
	SourceTypeDirectory = "directory"
	SourceTypeGitHub    = "github"
	SourceTypeURL       = "url"
	SourceTypeGitSubdir = "git-subdir"
	SourceTypeNpm       = "npm"
	SourceTypeArchive   = "archive"
	SourceTypeCommand   = "command"
)

// Marketplace represents a Claude Code marketplace entry from known_marketplaces.json.
type Marketplace struct {
	Name            string
	SourceType      string // one of the SourceType* constants
	SourcePath      string // local path (directory source only)
	SourceRepo      string // GitHub repo slug (github source only)
	SourceURL       string // git URL (url source only)
	InstallLocation string
	AutoUpdate      bool
}

// IsDirectory returns true if the marketplace source is a local directory.
func (m *Marketplace) IsDirectory() bool {
	return m.SourceType == SourceTypeDirectory
}

// IsGit returns true if the marketplace source is a remote git repository.
func (m *Marketplace) IsGit() bool {
	return m.SourceType == SourceTypeGitHub || m.SourceType == SourceTypeURL
}

// SupportsSourceVersion reports whether cplugins knows how to locate this
// marketplace's plugin.json to detect a stale cache. Types whose local
// layout it cannot infer (npm, archive, command, or anything unrecognized)
// report false.
func (m *Marketplace) SupportsSourceVersion() bool {
	switch m.SourceType {
	case SourceTypeDirectory, SourceTypeGitHub, SourceTypeURL, SourceTypeGitSubdir:
		return true
	default:
		return false
	}
}

// LocalRoot returns the local filesystem path containing the marketplace
// tree (with its plugins/ subdirectory). For directory sources it is the
// source path; for github/url sources it is the local clone maintained by
// Claude Code under ~/.claude/plugins/marketplaces/<name>/.
func (m *Marketplace) LocalRoot() string {
	if m.IsDirectory() {
		return m.SourcePath
	}
	return m.InstallLocation
}

type rawMarketplaceSource struct {
	Source string `json:"source"`
	Path   string `json:"path,omitempty"`
	Repo   string `json:"repo,omitempty"`
	URL    string `json:"url,omitempty"`
}

type rawMarketplace struct {
	Source          rawMarketplaceSource `json:"source"`
	InstallLocation string               `json:"installLocation"`
	LastUpdated     string               `json:"lastUpdated"`
	AutoUpdate      bool                 `json:"autoUpdate"`
}

// ParseMarketplaces reads and parses a known_marketplaces.json file.
// It returns a map keyed by marketplace name.
func ParseMarketplaces(path string) (map[string]*Marketplace, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading marketplaces file: %w", err)
	}

	var raw map[string]rawMarketplace
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing marketplaces JSON: %w", err)
	}

	result := make(map[string]*Marketplace, len(raw))
	for name, rm := range raw {
		m := &Marketplace{
			Name:            name,
			SourceType:      rm.Source.Source,
			InstallLocation: rm.InstallLocation,
			AutoUpdate:      rm.AutoUpdate,
		}
		switch rm.Source.Source {
		case SourceTypeDirectory:
			m.SourcePath = rm.Source.Path
		case SourceTypeGitHub:
			m.SourceRepo = rm.Source.Repo
		case SourceTypeURL:
			m.SourceURL = rm.Source.URL
		}
		result[name] = m
	}

	return result, nil
}

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Evaneos/cplugins/internal/claude"
)

// Claude Code's plugin auto-update repoints a dev-mode plugin at the cache as
// soon as the marketplace publishes a new version, so dev mode is recorded
// here, outside installed_plugins.json, for repair to restore it.
//
// Layout: {"plugins": {"<plugin@marketplace>": {"path": "<dev path>"}}}.
// Unknown fields are kept on rewrite.

func devStatePath() string {
	return filepath.Join(claudeDir(), "cplugins", "dev.json")
}

type devState struct {
	doc     map[string]any
	plugins map[string]any
	// readOnly is set when the file exists but couldn't be loaded and stays
	// in place: saving would overwrite it.
	readOnly bool
}

// loadDevState never fails, so repair keeps working at session start: a
// missing file yields an empty state, an unparsable one is set aside, and an
// unreadable one is left untouched with saving disabled.
func loadDevState(warn io.Writer) *devState {
	s := &devState{doc: map[string]any{}, plugins: map[string]any{}}
	data, err := os.ReadFile(devStatePath())
	if errors.Is(err, os.ErrNotExist) {
		return s
	}
	if err != nil {
		fmt.Fprintf(warn, "⚠ dev state unreadable, left untouched: %v\n", err)
		s.readOnly = true
		return s
	}
	if err := json.Unmarshal(data, &s.doc); err != nil {
		s.doc = map[string]any{}
		aside := fmt.Sprintf("%s.unreadable-%s", devStatePath(), time.Now().Format("20060102T150405"))
		if rerr := os.Rename(devStatePath(), aside); rerr != nil {
			fmt.Fprintf(warn, "⚠ dev state unparsable (%v), left untouched: %v\n", err, rerr)
			s.readOnly = true
			return s
		}
		fmt.Fprintf(warn, "⚠ dev state unparsable (%v), set aside as %s\n", err, aside)
		return s
	}
	if p, ok := s.doc["plugins"].(map[string]any); ok {
		s.plugins = p
	}
	return s
}

// path returns the recorded dev path of key, or "" when none is recorded.
func (s *devState) path(key string) string {
	entry, _ := s.plugins[key].(map[string]any)
	p, _ := entry["path"].(string)
	return p
}

func (s *devState) keys() []string {
	keys := make([]string, 0, len(s.plugins))
	for k := range s.plugins {
		keys = append(keys, k)
	}
	return keys
}

func (s *devState) record(key, path string) error {
	entry, ok := s.plugins[key].(map[string]any)
	if !ok {
		entry = map[string]any{}
	}
	entry["path"] = path
	s.plugins[key] = entry
	return s.save()
}

func (s *devState) forget(key string) error {
	if _, ok := s.plugins[key]; !ok {
		return nil
	}
	delete(s.plugins, key)
	return s.save()
}

func (s *devState) save() error {
	if s.readOnly {
		return fmt.Errorf("%s could not be loaded, not overwriting it", devStatePath())
	}
	s.doc["plugins"] = s.plugins
	data, err := json.MarshalIndent(s.doc, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(devStatePath()), 0o755); err != nil {
		return err
	}
	return claude.WriteFileAtomic(devStatePath(), append(data, '\n'))
}

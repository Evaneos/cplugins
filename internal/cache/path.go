package cache

import (
	"path/filepath"
	"strings"
)

// IsUnderCache returns true if installPath is under cacheDir.
func IsUnderCache(cacheDir, installPath string) bool {
	rel, err := filepath.Rel(cacheDir, installPath)
	return err == nil && !strings.HasPrefix(rel, "..")
}

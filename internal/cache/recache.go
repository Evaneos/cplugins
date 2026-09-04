package cache

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Evaneos/cplugins/internal/claude"
)

// RecachePlugin copies a plugin from its marketplace source into the cache directory.
// pluginsDir is the "plugins" subdirectory of the marketplace source (e.g., /path/to/mp/plugins).
// Returns the cache path and the version read from plugin.json.
//
// The plugin entry inside pluginsDir may be a symlink to the real plugin directory,
// which is a common pattern for marketplaces that aggregate per-repo plugins.
// EvalSymlinks resolves it so the recursive copy walks into the target.
func RecachePlugin(pluginsDir, pluginName, cacheDir, marketplaceName string) (cachePath, version string, err error) {
	srcDir, err := filepath.EvalSymlinks(filepath.Join(pluginsDir, pluginName))
	if err != nil {
		return "", "", fmt.Errorf("resolving source for %s: %w", pluginName, err)
	}

	version, err = claude.ResolveSourceVersion(srcDir)
	if err != nil {
		return "", "", fmt.Errorf("reading source version for %s: %w", pluginName, err)
	}

	cachePath = filepath.Join(cacheDir, marketplaceName, pluginName, version)

	if err := os.RemoveAll(cachePath); err != nil {
		return "", "", fmt.Errorf("clearing the cache directory for %s: %w", pluginName, err)
	}
	if err := copyDir(srcDir, cachePath); err != nil {
		return "", "", fmt.Errorf("copying %s to cache: %w", pluginName, err)
	}

	return cachePath, version, nil
}

// copySkipDirs: rebuildable development artifact directories, never useful in
// the cache (git repo, virtualenvs, bytecode, JS dependencies). Skipped during
// the copy regardless of where they sit in the source tree.
var copySkipDirs = map[string]bool{
	".git":         true,
	".venv":        true,
	"venv":         true,
	"__pycache__":  true,
	"node_modules": true,
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, rel)
		if d.IsDir() {
			if path != src && copySkipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return os.MkdirAll(dstPath, 0o755)
		}
		// WalkDir does not follow symlinks: for a link-to-directory,
		// d.IsDir() is false and a ReadFile would dereference the link and
		// then fail with "is a directory". Recreate the link as-is instead.
		if d.Type()&fs.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			// os.Symlink fails if dstPath already exists, unlike MkdirAll
			// (idempotent) or WriteFile (overwrites): remove it first to
			// stay safe even on a non-empty destination.
			_ = os.Remove(dstPath)
			return os.Symlink(target, dstPath)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, 0o644)
	})
}

package cache

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Evaneos/cplugins/internal/claude"
)

// RecachePlugin copies a plugin from its source directory in the marketplace
// tree into the cache directory. Returns the cache path and the version read
// from plugin.json.
//
// pluginDir may be a symlink to the real plugin directory, a common pattern for
// marketplaces that aggregate per-repo plugins. EvalSymlinks resolves it so the
// recursive copy walks into the target.
func RecachePlugin(pluginDir, pluginName, cacheDir, marketplaceName string) (cachePath, version string, err error) {
	srcDir, err := filepath.EvalSymlinks(pluginDir)
	if err != nil {
		return "", "", fmt.Errorf("resolving source for %s: %w", pluginName, err)
	}

	version, err = claude.ResolveSourceVersion(srcDir)
	if err != nil {
		return "", "", fmt.Errorf("reading source version for %s: %w", pluginName, err)
	}

	cachePath = filepath.Join(cacheDir, marketplaceName, pluginName, version)
	if rel, err := filepath.Rel(srcDir, canonicalPath(cachePath)); err == nil && filepath.IsLocal(rel) {
		return "", "", fmt.Errorf("cache directory %s lies inside the source of %s", cachePath, pluginName)
	}

	if err := os.RemoveAll(cachePath); err != nil {
		return "", "", fmt.Errorf("clearing the cache directory for %s: %w", pluginName, err)
	}
	if err := copyDir(srcDir, cachePath); err != nil {
		return "", "", fmt.Errorf("copying %s to cache: %w", pluginName, err)
	}

	return cachePath, version, nil
}

// canonicalPath resolves symlinks in path, which may not exist yet, through
// its deepest existing ancestor.
func canonicalPath(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	parent := filepath.Dir(path)
	if parent == path {
		return path
	}
	return filepath.Join(canonicalPath(parent), filepath.Base(path))
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
		info, err := d.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		// A read-only file left by a previous copy would make WriteFile fail.
		_ = os.Remove(dstPath)
		if err := os.WriteFile(dstPath, data, info.Mode().Perm()); err != nil {
			return err
		}
		// WriteFile's mode goes through the umask.
		return os.Chmod(dstPath, info.Mode().Perm())
	})
}

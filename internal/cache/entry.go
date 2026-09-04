package cache

import "os"

// RemoveEntry removes a file or directory at the given path.
func RemoveEntry(path string) error {
	return os.RemoveAll(path)
}

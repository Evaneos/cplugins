package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// orderedObject is a JSON object decoded with its key order preserved, so a
// round-trip through decodeOrderedObject and MarshalJSON reproduces every key
// — known to this package or not — in its original position.
type orderedObject struct {
	keys   []string
	values map[string]json.RawMessage
}

// decodeOrderedObject parses a JSON object, keeping track of the order in
// which its keys appear.
func decodeOrderedObject(data []byte) (*orderedObject, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return nil, fmt.Errorf("expected a JSON object")
	}

	obj := &orderedObject{values: make(map[string]json.RawMessage)}
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := keyTok.(string)
		if !ok {
			return nil, fmt.Errorf("expected a string key")
		}
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, err
		}
		obj.keys = append(obj.keys, key)
		obj.values[key] = raw
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	return obj, nil
}

// get returns the raw value for key, if present.
func (o *orderedObject) get(key string) (json.RawMessage, bool) {
	v, ok := o.values[key]
	return v, ok
}

// set assigns key to value, appending it to the key order on first insertion.
func (o *orderedObject) set(key string, value json.RawMessage) {
	if _, exists := o.values[key]; !exists {
		o.keys = append(o.keys, key)
	}
	o.values[key] = value
}

// delete removes key, if present.
func (o *orderedObject) delete(key string) {
	if _, exists := o.values[key]; !exists {
		return
	}
	delete(o.values, key)
	for i, k := range o.keys {
		if k == key {
			o.keys = append(o.keys[:i], o.keys[i+1:]...)
			break
		}
	}
}

// MarshalJSON encodes the object with its keys in their original order.
func (o *orderedObject) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, key := range o.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		keyJSON, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		buf.Write(keyJSON)
		buf.WriteByte(':')
		buf.Write(o.values[key])
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// stringField reads key as a string, or "" if absent or not a string.
func stringField(entry *orderedObject, key string) string {
	raw, ok := entry.get(key)
	if !ok {
		return ""
	}
	var s string
	_ = json.Unmarshal(raw, &s)
	return s
}

// setStringField assigns key to a JSON-encoded string value.
func setStringField(entry *orderedObject, key, value string) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	entry.set(key, raw)
	return nil
}

// decodeEntries parses a JSON array of install entries, preserving each
// entry's own key order.
func decodeEntries(raw json.RawMessage) ([]*orderedObject, error) {
	var rawEntries []json.RawMessage
	if err := json.Unmarshal(raw, &rawEntries); err != nil {
		return nil, err
	}
	entries := make([]*orderedObject, len(rawEntries))
	for i, r := range rawEntries {
		entry, err := decodeOrderedObject(r)
		if err != nil {
			return nil, err
		}
		entries[i] = entry
	}
	return entries, nil
}

// encodeEntries re-encodes a slice of install entries as a JSON array.
func encodeEntries(entries []*orderedObject) (json.RawMessage, error) {
	var buf bytes.Buffer
	buf.WriteByte('[')
	for i, entry := range entries {
		if i > 0 {
			buf.WriteByte(',')
		}
		raw, err := entry.MarshalJSON()
		if err != nil {
			return nil, err
		}
		buf.Write(raw)
	}
	buf.WriteByte(']')
	return buf.Bytes(), nil
}

// readRegistry reads and parses installed_plugins.json into its root object,
// keeping every key — including ones this package does not know about — and
// their original order.
func readRegistry(path string) (*orderedObject, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading installed plugins: %w", err)
	}
	root, err := decodeOrderedObject(data)
	if err != nil {
		return nil, fmt.Errorf("parsing installed plugins: %w", err)
	}
	return root, nil
}

// writeRegistry serializes root back to path, atomically: it writes to a
// temporary file in the same directory and renames it into place, so a
// concurrent reader never observes a partial write. The temporary file is
// removed if any step fails.
func writeRegistry(path string, root *orderedObject) error {
	raw, err := root.MarshalJSON()
	if err != nil {
		return fmt.Errorf("marshaling installed plugins: %w", err)
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, raw, "", "  "); err != nil {
		return fmt.Errorf("formatting installed plugins: %w", err)
	}

	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".installed_plugins-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()
	renamed := false
	defer func() {
		if !renamed {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("setting temp file permissions: %w", err)
	}
	if _, err := tmp.Write(pretty.Bytes()); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("renaming temp file: %w", err)
	}
	renamed = true
	return nil
}

// getPluginsObject extracts the "plugins" object from root, keeping its key
// order.
func getPluginsObject(root *orderedObject) (*orderedObject, error) {
	raw, ok := root.get("plugins")
	if !ok {
		return nil, fmt.Errorf(`installed plugins file has no "plugins" key`)
	}
	plugins, err := decodeOrderedObject(raw)
	if err != nil {
		return nil, fmt.Errorf("parsing plugins: %w", err)
	}
	return plugins, nil
}

// withPluginsRegistry reads path, hands its "plugins" object to fn, and — if
// fn succeeds — writes the file back atomically with fn's changes applied and
// every other field, known or not, preserved exactly.
func withPluginsRegistry(path string, fn func(plugins *orderedObject) error) error {
	root, err := readRegistry(path)
	if err != nil {
		return err
	}
	plugins, err := getPluginsObject(root)
	if err != nil {
		return err
	}
	if err := fn(plugins); err != nil {
		return err
	}
	pluginsRaw, err := plugins.MarshalJSON()
	if err != nil {
		return fmt.Errorf("marshaling plugins: %w", err)
	}
	root.set("plugins", pluginsRaw)
	return writeRegistry(path, root)
}

// PatchInstallPaths updates the installPath for each scope of a plugin in installed_plugins.json.
// patchFn receives the scope and current installPath and returns the new installPath.
// Every other field — known or not, on the patched entries or anywhere else in the file —
// is preserved exactly, in its original key order.
func PatchInstallPaths(path, key string, patchFn func(scope, current string) string) error {
	return withPluginsRegistry(path, func(plugins *orderedObject) error {
		raw, ok := plugins.get(key)
		if !ok {
			return fmt.Errorf("plugin %q not found", key)
		}
		entries, err := decodeEntries(raw)
		if err != nil {
			return fmt.Errorf("parsing installed plugins: %w", err)
		}
		for _, entry := range entries {
			scope := stringField(entry, "scope")
			current := stringField(entry, "installPath")
			if err := setStringField(entry, "installPath", patchFn(scope, current)); err != nil {
				return fmt.Errorf("marshaling installed plugins: %w", err)
			}
		}
		newRaw, err := encodeEntries(entries)
		if err != nil {
			return fmt.Errorf("marshaling installed plugins: %w", err)
		}
		plugins.set(key, newRaw)
		return nil
	})
}

// PatchInstalls updates the installPath and version for each scope of a plugin in installed_plugins.json.
// patchFn receives the scope, current installPath, and current version, and returns the new installPath and version.
// Every other field — known or not, on the patched entries or anywhere else in the file —
// is preserved exactly, in its original key order.
func PatchInstalls(path, key string, patchFn func(scope, currentPath, currentVersion string) (newPath, newVersion string)) error {
	return withPluginsRegistry(path, func(plugins *orderedObject) error {
		raw, ok := plugins.get(key)
		if !ok {
			return fmt.Errorf("plugin %q not found", key)
		}
		entries, err := decodeEntries(raw)
		if err != nil {
			return fmt.Errorf("parsing installed plugins: %w", err)
		}
		for _, entry := range entries {
			scope := stringField(entry, "scope")
			currentPath := stringField(entry, "installPath")
			currentVersion := stringField(entry, "version")
			newPath, newVersion := patchFn(scope, currentPath, currentVersion)
			if err := setStringField(entry, "installPath", newPath); err != nil {
				return fmt.Errorf("marshaling installed plugins: %w", err)
			}
			if err := setStringField(entry, "version", newVersion); err != nil {
				return fmt.Errorf("marshaling installed plugins: %w", err)
			}
		}
		newRaw, err := encodeEntries(entries)
		if err != nil {
			return fmt.Errorf("marshaling installed plugins: %w", err)
		}
		plugins.set(key, newRaw)
		return nil
	})
}

// RemoveInstalls removes the installs of a plugin for which pred returns true
// from installed_plugins.json, deleting the plugin's key entirely once none
// are left. Returns the number of installs removed.
// Every other field, known or not, is preserved exactly, in its original key order.
func RemoveInstalls(path, key string, pred func(Install) bool) (int, error) {
	removed := 0
	err := withPluginsRegistry(path, func(plugins *orderedObject) error {
		raw, ok := plugins.get(key)
		if !ok {
			return fmt.Errorf("plugin %q not found", key)
		}
		entries, err := decodeEntries(raw)
		if err != nil {
			return fmt.Errorf("parsing installed plugins: %w", err)
		}
		kept := entries[:0]
		for _, entry := range entries {
			install := Install{
				Scope:       stringField(entry, "scope"),
				ProjectPath: stringField(entry, "projectPath"),
				InstallPath: stringField(entry, "installPath"),
				Version:     stringField(entry, "version"),
			}
			if pred(install) {
				removed++
				continue
			}
			kept = append(kept, entry)
		}
		if len(kept) == 0 {
			plugins.delete(key)
			return nil
		}
		newRaw, err := encodeEntries(kept)
		if err != nil {
			return fmt.Errorf("marshaling installed plugins: %w", err)
		}
		plugins.set(key, newRaw)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return removed, nil
}

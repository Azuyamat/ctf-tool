package fsutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// ValidName reports whether name is a single path element, usable as a folder name.
func ValidName(name string) bool {
	return name != "." && name != ".." && filepath.Base(name) == name && !strings.ContainsAny(name, `/\`)
}

// Within returns path relative to base when path lies strictly inside base.
func Within(base, path string) (string, bool) {
	rel, err := filepath.Rel(base, path)
	if err != nil || rel == "." || !filepath.IsLocal(rel) {
		return "", false
	}
	return rel, true
}

// WriteJSON atomically writes v as indented JSON to path, creating parent folders.
func WriteJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

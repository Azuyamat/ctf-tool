package template

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/azuyamat/ctf/internal/fsutil"
)

const (
	DirName = "templates"
	Default = "default"
)

var ErrNotFound = errors.New("template not found")

type Template struct {
	Name  string
	Path  string
	Scope string
}

type source struct {
	scope string
	dir   string
}

// sources returns template directories in lookup order, event first.
func sources(base, event string) []source {
	var srcs []source
	if event != "" {
		srcs = append(srcs, source{scope: "event", dir: filepath.Join(base, event, DirName)})
	}
	return append(srcs, source{scope: "global", dir: filepath.Join(base, DirName)})
}

func Find(base, event, name string) (*Template, error) {
	if !fsutil.ValidName(name) {
		return nil, fmt.Errorf("invalid template name: %s", name)
	}

	for _, src := range sources(base, event) {
		path := filepath.Join(src.dir, name)
		info, err := os.Stat(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			return &Template{Name: name, Path: path, Scope: src.scope}, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrNotFound, name)
}

// List returns all available templates, with event templates shadowing global ones.
func List(base, event string) ([]Template, error) {
	seen := make(map[string]bool)
	var templates []Template
	for _, src := range sources(base, event) {
		entries, err := os.ReadDir(src.dir)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if !entry.IsDir() || seen[entry.Name()] {
				continue
			}
			seen[entry.Name()] = true
			templates = append(templates, Template{
				Name:  entry.Name(),
				Path:  filepath.Join(src.dir, entry.Name()),
				Scope: src.scope,
			})
		}
	}
	return templates, nil
}

func (t *Template) CopyTo(dst string) error {
	return os.CopyFS(dst, os.DirFS(t.Path))
}

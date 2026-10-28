package template

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	texttemplate "text/template"

	"github.com/azuyamat/ctf/internal/fsutil"
	"github.com/azuyamat/ctf/internal/layout"
)

const (
	Default = "default"
	Ext     = ".tmpl"

	ScopeEvent  = "event"
	ScopeGlobal = "global"
)

var ErrNotFound = errors.New("template not found")

type Template struct {
	Name     string
	Path     string
	Scope    string
	Shadowed bool
}

type source struct {
	scope string
	dir   string
}

// sources returns template directories in lookup order, event first.
func sources(l layout.Layout, event string) []source {
	var srcs []source
	if event != "" {
		srcs = append(srcs, source{scope: ScopeEvent, dir: l.EventTemplatesDir(event)})
	}
	return append(srcs, source{scope: ScopeGlobal, dir: l.GlobalTemplatesDir()})
}

func Find(l layout.Layout, event, name string) (*Template, error) {
	if !fsutil.ValidName(name) {
		return nil, fmt.Errorf("invalid template name: %s", name)
	}

	for _, src := range sources(l, event) {
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

// List returns all templates in lookup order, marking those shadowed by an earlier source.
func List(l layout.Layout, event string) ([]Template, error) {
	seen := make(map[string]bool)
	var templates []Template
	for _, src := range sources(l, event) {
		entries, err := os.ReadDir(src.dir)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			templates = append(templates, Template{
				Name:     entry.Name(),
				Path:     filepath.Join(src.dir, entry.Name()),
				Scope:    src.scope,
				Shadowed: seen[entry.Name()],
			})
			seen[entry.Name()] = true
		}
	}
	return templates, nil
}

type Data struct {
	Name       string
	Event      string
	Category   string
	Connection string
	Host       string
	Port       string
	URL        string
	Points     int
}

func (t *Template) CopyTo(dst string, data Data) error {
	src := os.DirFS(t.Path)
	return fs.WalkDir(src, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(dst, filepath.FromSlash(path))
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		content, err := fs.ReadFile(src, path)
		if err != nil {
			return err
		}
		info, err := fs.Stat(src, path)
		if err != nil {
			return err
		}

		target, render := strings.CutSuffix(target, Ext)
		f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
		if err != nil {
			return err
		}
		if render {
			err = execute(f, path, content, data)
		} else {
			_, err = f.Write(content)
		}
		return errors.Join(err, f.Close())
	})
}

func execute(w io.Writer, name string, content []byte, data Data) error {
	tmpl, err := texttemplate.New(name).Parse(string(content))
	if err != nil {
		return err
	}
	return tmpl.Execute(w, data)
}

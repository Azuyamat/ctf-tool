package challenge

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/azuyamat/ctf/internal/config"
	"github.com/azuyamat/ctf/internal/state"
	"github.com/azuyamat/ctf/internal/template"
)

type Challenge struct {
	Name string
	Path string
}

func Root() (string, error) {
	base := config.Current.BaseDir
	if event := state.Current.Event; event != "" {
		return filepath.Join(base, event), nil
	}
	return base, nil
}

func New(name string) (*Challenge, error) {
	root, err := Root()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(root, name)
	if !strings.HasPrefix(path, root+string(filepath.Separator)) {
		return nil, fmt.Errorf("invalid challenge name: %s", name)
	}

	rel, _ := filepath.Rel(root, path)
	if strings.SplitN(rel, string(filepath.Separator), 2)[0] == template.DirName {
		return nil, fmt.Errorf("challenge name is reserved: %s", name)
	}

	return &Challenge{
		Name: name,
		Path: path,
	}, nil
}

// Create makes the challenge folder, copying tmpl into it when non-nil.
func (c *Challenge) Create(tmpl *template.Template) error {
	if err := os.MkdirAll(filepath.Dir(c.Path), 0755); err != nil {
		return err
	}

	if err := os.Mkdir(c.Path, 0755); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("challenge already exists: %s", c.Path)
		}
		return err
	}

	if tmpl == nil {
		return nil
	}
	if err := tmpl.CopyTo(c.Path); err != nil {
		os.RemoveAll(c.Path)
		return fmt.Errorf("copy template %s: %w", tmpl.Name, err)
	}
	return nil
}

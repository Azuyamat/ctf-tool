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

	return &Challenge{
		Name: name,
		Path: path,
	}, nil
}

func (c *Challenge) Create() error {
	if err := os.MkdirAll(filepath.Dir(c.Path), 0755); err != nil {
		return err
	}

	if err := os.Mkdir(c.Path, 0755); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("challenge already exists: %s", c.Path)
		}
		return err
	}
	return nil
}

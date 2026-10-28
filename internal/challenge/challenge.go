package challenge

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/azuyamat/ctf/internal/fsutil"
	"github.com/azuyamat/ctf/internal/template"
)

var challengeFileName = ".ctf.json"

type Challenge struct {
	Name    string    `json:"name"`
	Path    string    `json:"-"`
	Flag    string    `json:"flag,omitempty"`
	Created time.Time `json:"created"`
	Solved  time.Time `json:"solved,omitzero"`
}

// Root is the folder challenges are created in: the event folder when an event is set.
func Root(base, event string) string {
	return filepath.Join(base, event)
}

func New(root, name string) (*Challenge, error) {
	root = filepath.Clean(root)
	rel, ok := fsutil.Within(root, filepath.Join(root, name))
	if !ok {
		return nil, fmt.Errorf("invalid challenge name: %s", name)
	}
	if slices.Contains(strings.Split(rel, string(filepath.Separator)), template.DirName) {
		return nil, fmt.Errorf("challenge name is reserved: %s", name)
	}

	return &Challenge{
		Name:    rel,
		Path:    filepath.Join(root, rel),
		Created: time.Now(),
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

	if tmpl != nil {
		if err := tmpl.CopyTo(c.Path); err != nil {
			os.RemoveAll(c.Path)
			return fmt.Errorf("copy template %s: %w", tmpl.Name, err)
		}
	}

	// Saved after the template copy so it wins over any marker the template contains.
	if err := c.Save(); err != nil {
		os.RemoveAll(c.Path)
		return err
	}
	return nil
}

// SetFlag records the flag. The solve time is kept from the first time a flag was set.
func (c *Challenge) SetFlag(flag string) {
	c.Flag = flag
	if c.Solved.IsZero() {
		c.Solved = time.Now()
	}
}

func (c *Challenge) configFilePath() string {
	return filepath.Join(c.Path, challengeFileName)
}

func (c *Challenge) Load() error {
	data, err := os.ReadFile(c.configFilePath())
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}
	if err := json.Unmarshal(data, c); err != nil {
		return fmt.Errorf("unmarshal config %s: %w", c.configFilePath(), err)
	}
	return nil
}

func (c *Challenge) Save() error {
	if err := fsutil.WriteJSON(c.configFilePath(), c); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	return nil
}

var ErrNotFound = errors.New("not inside a challenge")

// LoadFromPath finds the challenge containing path by walking up to base.
// It is bounded by base rather than Root so it works across events.
func LoadFromPath(base, path string) (*Challenge, error) {
	base = filepath.Clean(base)
	path = filepath.Clean(path)
	if _, ok := fsutil.Within(base, path); !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, path)
	}

	for p := path; p != base; p = filepath.Dir(p) {
		challenge := &Challenge{Path: p}
		if err := challenge.Load(); err == nil {
			return challenge, nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("load challenge config: %w", err)
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrNotFound, path)
}

// LoadFromCwd loads the challenge containing the working directory.
func LoadFromCwd(base string) (*Challenge, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get current dir: %w", err)
	}
	return LoadFromPath(base, cwd)
}

// LoadCurrent loads the challenge containing the working directory, falling
// back to the last created challenge.
func LoadCurrent(base, last string) (*Challenge, error) {
	ch, err := LoadFromCwd(base)
	if !errors.Is(err, ErrNotFound) || last == "" {
		return ch, err
	}
	return LoadFromPath(base, last)
}

// ListChallenges returns the challenges under root. A missing root has none.
func ListChallenges(root string) ([]*Challenge, error) {
	var challenges []*Challenge
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == root && errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if path != root && d.Name() == template.DirName {
			return filepath.SkipDir
		}
		challenge := &Challenge{Path: path}
		if err := challenge.Load(); err == nil {
			challenges = append(challenges, challenge)
			return filepath.SkipDir
		} else if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("load challenge config: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk challenges: %w", err)
	}

	return challenges, nil
}

package challenge

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/azuyamat/ctf/internal/config"
	"github.com/azuyamat/ctf/internal/state"
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
		Name:    name,
		Path:    path,
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

func (c *Challenge) SetFlag(flag string) {
	c.Flag = flag
	c.Solved = time.Now()
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
		return fmt.Errorf("unmarshal config: %w", err)
	}
	return nil
}

func (c *Challenge) Save() error {
	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(c.configFilePath(), data, 0644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	return nil
}

var ErrNotFound = errors.New("not inside a challenge")

// LoadFromPath finds the challenge containing path by walking up to the base dir.
// It is bounded by the base dir rather than Root so it works across events.
func LoadFromPath(path string) (*Challenge, error) {
	base := filepath.Clean(config.Current.BaseDir)
	path = filepath.Clean(path)
	if !strings.HasPrefix(path, base+string(filepath.Separator)) {
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

// LoadCurrent loads the challenge containing the working directory, falling
// back to the last created challenge.
func LoadCurrent() (*Challenge, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get current dir: %w", err)
	}

	ch, err := LoadFromPath(cwd)
	if !errors.Is(err, ErrNotFound) || state.Current.LastChallenge == "" {
		return ch, err
	}
	return LoadFromPath(state.Current.LastChallenge)
}

func ListChallenges() ([]*Challenge, error) {
	root, err := Root()
	if err != nil {
		return nil, err
	}

	var challenges []*Challenge
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if d.Name() == template.DirName {
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

package config

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/azuyamat/ctf/internal/challenge"
	"github.com/azuyamat/ctf/internal/event"
	"github.com/azuyamat/ctf/internal/fsutil"
	"github.com/azuyamat/ctf/internal/template"
)

type Config struct {
	BaseDir         string `json:"baseDir"`
	DownloadDir     string `json:"downloadDir"`
	Editor          string `json:"editor"`
	DefaultTemplate string `json:"defaultTemplate"`
	ReadmeName      string `json:"readmeName"`
	FilesDir        string `json:"filesDir"`
}

var Keys = []Key{
	stringKey("baseDir", "Base directory for challenges", func(c *Config) *string { return &c.BaseDir }, NormalizePath),
	stringKey("downloadDir", "Your default downloads directory", func(c *Config) *string { return &c.DownloadDir }, NormalizePath),
	stringKey("editor", "Command ctf open runs, empty to use $EDITOR", func(c *Config) *string { return &c.Editor }, normalizeEditor),
	stringKey("defaultTemplate", "Template used when no -t is given", func(c *Config) *string { return &c.DefaultTemplate }, normalizeTemplate),
	eventKey("readmeName", "File pull writes the challenge description to", func(c *Config) *string { return &c.ReadmeName }, func(e *event.Event) *string { return &e.ReadmeName }, normalizeFileName),
	eventKey("filesDir", "Folder in a challenge that downloads go into", func(c *Config) *string { return &c.FilesDir }, func(e *event.Event) *string { return &e.FilesDir }, normalizeFileName),
}

func eventKey(name, description string, field func(*Config) *string, eventField func(*event.Event) *string, normalize func(string) (string, error)) Key {
	k := stringKey(name, description, field, normalize)
	k.GetEvent = func(e *event.Event) string {
		return *eventField(e)
	}
	k.SetEvent = func(e *event.Event, value string) error {
		if value != "" {
			var err error
			if value, err = normalize(value); err != nil {
				return err
			}
		}
		*eventField(e) = value
		return nil
	}
	return k
}

func stringKey(name, description string, field func(*Config) *string, normalize func(string) (string, error)) Key {
	return Key{
		Name:        name,
		Description: description,
		Get: func(c *Config) string {
			return *field(c)
		},
		Set: func(c *Config, value string) error {
			value, err := normalize(value)
			if err != nil {
				return err
			}
			*field(c) = value
			return nil
		},
	}
}

func normalizeEditor(value string) (string, error) {
	return strings.Join(strings.Fields(value), " "), nil
}

func normalizeTemplate(value string) (string, error) {
	if !fsutil.ValidName(value) {
		return "", fmt.Errorf("invalid template name: %q", value)
	}
	return value, nil
}

func normalizeFileName(value string) (string, error) {
	return value, challenge.ValidateName(value)
}

func FindKey(name string) *Key {
	for i := range Keys {
		if Keys[i].Name == name {
			return &Keys[i]
		}
	}
	return nil
}

// NormalizePath expands a leading ~ and returns the absolute, cleaned path.
func NormalizePath(path string) (string, error) {
	if path == "" {
		return "", errors.New("path cannot be empty")
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[1:])
	}
	return filepath.Abs(path)
}

func ConfigFolderPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ctf"), nil
}

func configFilePath() (string, error) {
	folder, err := ConfigFolderPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(folder, "config.json"), nil
}

func Defaults() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return &Config{
		BaseDir:         filepath.Join(home, "ctf"),
		DownloadDir:     filepath.Join(home, "Downloads"),
		DefaultTemplate: template.Default,
		ReadmeName:      challenge.DefaultReadmeName,
		FilesDir:        challenge.DefaultFilesDir,
	}, nil
}

// LoadOrCreateConfig loads the config file, writing the defaults if it does not exist.
// Keys missing from the file keep their default values.
func LoadOrCreateConfig() (*Config, error) {
	path, err := configFilePath()
	if err != nil {
		return nil, err
	}
	config, err := Defaults()
	if err != nil {
		return nil, err
	}

	if err := fsutil.ReadJSON(path, config); errors.Is(err, fs.ErrNotExist) {
		return config, config.Save()
	} else if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	// Re-set every key so hand-edited values get normalized too.
	for _, key := range Keys {
		if err := key.Set(config, key.Get(config)); err != nil {
			return nil, fmt.Errorf("config %s: %s: %w", path, key.Name, err)
		}
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config %s: %w", path, err)
	}
	return config, nil
}

func (c *Config) Validate() error {
	return c.Paths(nil).Validate()
}

func (c *Config) Paths(ev *event.Event) challenge.Paths {
	p := challenge.Paths{Readme: c.ReadmeName, Files: c.FilesDir}
	if ev != nil {
		p.Readme = cmp.Or(ev.ReadmeName, p.Readme)
		p.Files = cmp.Or(ev.FilesDir, p.Files)
	}
	return p
}

func (c *Config) Save() error {
	path, err := configFilePath()
	if err != nil {
		return err
	}
	if err := fsutil.WriteJSON(path, c, 0644); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	return nil
}

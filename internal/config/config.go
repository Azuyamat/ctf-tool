package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/azuyamat/ctf/internal/challenge"
	"github.com/azuyamat/ctf/internal/fsutil"
	"github.com/azuyamat/ctf/internal/layout"
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
	stringKey("readmeName", "File pull writes the challenge description to", func(c *Config) *string { return &c.ReadmeName }, normalizeFileName),
	stringKey("filesDir", "Folder in a challenge that downloads go into", func(c *Config) *string { return &c.FilesDir }, normalizeFileName),
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
	if !fsutil.ValidName(value) {
		return "", fmt.Errorf("invalid file name: %q", value)
	}
	if value == challenge.FileName || layout.IsReserved(value) {
		return "", fmt.Errorf("%s is used by ctf", value)
	}
	return value, nil
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

func defaultConfig() (*Config, error) {
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
	config, err := defaultConfig()
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
	if c.ReadmeName == c.FilesDir {
		return fmt.Errorf("readmeName and filesDir can't both be %s", c.FilesDir)
	}
	return nil
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

package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/azuyamat/ctf/internal/fsutil"
)

type Config struct {
	BaseDir     string `json:"baseDir"`
	DownloadDir string `json:"downloadDir"`
}

var Keys = []Key{
	{
		Name:        "baseDir",
		Description: "Base directory for challenges",
		Get: func(c *Config) string {
			return c.BaseDir
		},
		Set: func(c *Config, value string) error {
			path, err := NormalizePath(value)
			if err != nil {
				return err
			}
			c.BaseDir = path
			return nil
		},
	},
	{
		Name:        "downloadDir",
		Description: "Your default downloads directory",
		Get: func(c *Config) string {
			return c.DownloadDir
		},
		Set: func(c *Config, value string) error {
			path, err := NormalizePath(value)
			if err != nil {
				return err
			}
			c.DownloadDir = path
			return nil
		},
	},
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
		BaseDir:     filepath.Join(home, "ctf"),
		DownloadDir: filepath.Join(home, "Downloads"),
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
	return config, nil
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

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	BaseDir     string
	DownloadDir string
}

var Current *Config
var Keys = []Key{
	{
		Name:        "baseDir",
		Description: "Base directory for challenges",
		Get: func(c *Config) string {
			return c.BaseDir
		},
		Set: func(c *Config, value string) error {
			c.BaseDir = value
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
			c.DownloadDir = value
			return nil
		},
	},
}

func FindKey(name string) *Key {
	for _, key := range Keys {
		if key.Name == name {
			return &key
		}
	}
	return nil
}

func ConfigFolderPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ctf"), nil
}

func LoadOrCreateConfig() (*Config, error) {
	err := createConfigFolderIfNotExists()
	if err != nil {
		return nil, err
	}

	config, err := loadConfig()
	if err != nil {
		return nil, err
	}

	return config, nil
}

func createConfigFolderIfNotExists() error {
	path, err := ConfigFolderPath()
	if err != nil {
		return err
	}
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	return err
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

func loadConfig() (*Config, error) {
	path, err := configFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		config, err := defaultConfig()
		if err != nil {
			return nil, err
		}
		return config, config.Save()
	}
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func (c *Config) Save() error {
	path, err := configFilePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

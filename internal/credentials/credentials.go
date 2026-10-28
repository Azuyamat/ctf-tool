package credentials

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/azuyamat/ctf/internal/fsutil"
	"github.com/azuyamat/ctf/internal/platform"
	"github.com/azuyamat/ctf/internal/state"
)

type Credentials struct {
	Tokens   map[string]string `json:"tokens"`
	Sessions map[string]string `json:"sessions,omitempty"`
	Users    map[string]string `json:"users,omitempty"`
}

func filePath() (string, error) {
	dir, err := state.StateFolderPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "credentials.json"), nil
}

func Load() (*Credentials, error) {
	path, err := filePath()
	if err != nil {
		return nil, err
	}

	c := &Credentials{}
	if err := fsutil.ReadJSON(path, c); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("read credentials: %w", err)
	}
	if c.Tokens == nil {
		c.Tokens = make(map[string]string)
	}
	if c.Sessions == nil {
		c.Sessions = make(map[string]string)
	}
	if c.Users == nil {
		c.Users = make(map[string]string)
	}
	return c, nil
}

func (c *Credentials) Auth(instance string) platform.Auth {
	return platform.Auth{Token: c.Tokens[instance], Session: c.Sessions[instance]}
}

func (c *Credentials) Set(instance string, auth platform.Auth) {
	c.Delete(instance)
	if auth.Token != "" {
		c.Tokens[instance] = auth.Token
	}
	if auth.Session != "" {
		c.Sessions[instance] = auth.Session
	}
}

func (c *Credentials) User(instance string) string {
	return c.Users[instance]
}

func (c *Credentials) SetUser(instance, name string) {
	c.Users[instance] = name
}

func (c *Credentials) Delete(instance string) bool {
	_, token := c.Tokens[instance]
	_, session := c.Sessions[instance]
	delete(c.Tokens, instance)
	delete(c.Sessions, instance)
	delete(c.Users, instance)
	return token || session
}

func (c *Credentials) Save() error {
	path, err := filePath()
	if err != nil {
		return err
	}
	if err := fsutil.WriteJSON(path, c, 0600); err != nil {
		return fmt.Errorf("write credentials: %w", err)
	}
	return nil
}

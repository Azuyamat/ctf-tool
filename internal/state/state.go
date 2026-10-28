package state

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/azuyamat/ctf/internal/fsutil"
)

type State struct {
	Event         string `json:"event,omitempty"`
	LastChallenge string `json:"lastChallenge,omitempty"`
}

func StateFolderPath() (string, error) {
	if dir := os.Getenv("XDG_STATE_HOME"); filepath.IsAbs(dir) {
		return filepath.Join(dir, "ctf"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "ctf"), nil
}

func stateFilePath() (string, error) {
	folder, err := StateFolderPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(folder, "state.json"), nil
}

func Load() (*State, error) {
	path, err := stateFilePath()
	if err != nil {
		return nil, err
	}

	var state State
	if err := fsutil.ReadJSON(path, &state); errors.Is(err, fs.ErrNotExist) {
		return &State{}, nil
	} else if err != nil {
		return nil, fmt.Errorf("read state: %w", err)
	}
	return &state, nil
}

func (s *State) Save() error {
	path, err := stateFilePath()
	if err != nil {
		return err
	}
	if err := fsutil.WriteJSON(path, s, 0644); err != nil {
		return fmt.Errorf("write state %s: %w", path, err)
	}
	return nil
}

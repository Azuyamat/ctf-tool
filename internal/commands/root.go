package commands

import (
	"os"
	"path/filepath"

	"github.com/azuyamat/ctf/internal/challenge"
	"github.com/azuyamat/ctf/internal/config"
	"github.com/azuyamat/ctf/internal/fsutil"
	"github.com/azuyamat/ctf/internal/state"
	"github.com/azuyamat/gear/command"
)

func Root() *command.RootCommand {
	return command.NewRootCommand("ctf", "Manage CTF challenges effortlessly").
		AddChild(cmdNew()).
		AddChild(cmdEvent()).
		AddChild(cmdLast()).
		AddChild(cmdTemplates()).
		AddChild(cmdFlag()).
		AddChild(cmdLs()).
		AddChild(cmdConfig()).
		AddChild(cmdGrab()).
		AddChild(cmdOpen())
}

func challengeRoot() string {
	return challenge.Root(config.Current.BaseDir, state.Current.Event)
}

func shortPath(path string) string {
	if rel, ok := fsutil.Within(config.Current.BaseDir, path); ok {
		return rel
	}
	if home, err := os.UserHomeDir(); err == nil {
		if rel, ok := fsutil.Within(home, path); ok {
			return filepath.Join("~", rel)
		}
	}
	return path
}

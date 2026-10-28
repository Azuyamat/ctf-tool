package commands

import (
	"github.com/azuyamat/ctf/internal/challenge"
	"github.com/azuyamat/ctf/internal/config"
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
		AddChild(cmdConfig())
}

func challengeRoot() string {
	return challenge.Root(config.Current.BaseDir, state.Current.Event)
}

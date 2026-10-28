package commands

import "github.com/azuyamat/gear/command"

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

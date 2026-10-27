package commands

import "github.com/azuyamat/gear/command"

func cmdNew() command.Command {
	return command.NewExecutableCommand("new", "Create a new CTF challenge folder").
		Handler(runNew)
}

func runNew(ctx *command.Context, args command.ValidatedArgs) error {
	return nil
}

package commands

import (
	"fmt"

	"github.com/azuyamat/gear/command"
)

func cmdVersion(version string) command.Command {
	return command.NewExecutableCommand("version", "Print the ctf version").
		Group("Setup").
		Handler(func(ctx *command.Context, args command.ValidatedArgs) error {
			fmt.Println(version)
			return nil
		})
}

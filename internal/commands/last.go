package commands

import (
	"fmt"
	"os"

	"github.com/azuyamat/ctf/internal/state"
	"github.com/azuyamat/gear/command"
)

func cmdLast() command.Command {
	return command.NewExecutableCommand("last", "Show the last created challenge folder").
		Handler(runLast)
}

func runLast(ctx *command.Context, args command.ValidatedArgs) error {
	fmt.Fprintf(os.Stdout, "%s\n", state.Current.LastChallenge)
	return nil
}

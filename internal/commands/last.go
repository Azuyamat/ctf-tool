package commands

import (
	"errors"
	"fmt"

	"github.com/azuyamat/ctf/internal/state"
	"github.com/azuyamat/gear/command"
)

func cmdLast() command.Command {
	return command.NewExecutableCommand("last", "Show the last created challenge folder").
		Handler(runLast)
}

// runLast fails rather than printing a message when there is no last challenge,
// so `cd "$(ctf last)"` never sees non-path output.
func runLast(ctx *command.Context, args command.ValidatedArgs) error {
	if state.Current.LastChallenge == "" {
		return errors.New("no challenge created yet")
	}
	fmt.Println(state.Current.LastChallenge)
	return nil
}

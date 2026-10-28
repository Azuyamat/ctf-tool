package commands

import (
	"errors"
	"fmt"

	"github.com/azuyamat/gear/command"
)

func (c *cli) cmdLast() command.Command {
	return command.NewExecutableCommand("last", "Show the last created challenge folder").
		Handler(c.runLast)
}

// runLast fails rather than printing a message when there is no last challenge,
// so `cd "$(ctf last)"` never sees non-path output.
func (c *cli) runLast(ctx *command.Context, args command.ValidatedArgs) error {
	if c.State.LastChallenge == "" {
		return errors.New("no challenge created yet")
	}
	fmt.Println(c.State.LastChallenge)
	return nil
}

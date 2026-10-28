package commands

import (
	"errors"

	"github.com/azuyamat/ctf/internal/ui"
	"github.com/azuyamat/gear/command"
)

func (c *cli) cmdFlag() command.Command {
	return command.NewExecutableCommand("flag", "Show or set the flag of the current challenge").
		Group("Challenges").
		Args(
			command.NewStringArg(
				"flag",
				"Flag string to set for the current challenge",
			).AsOptional(),
		).
		Handler(c.runFlag)
}

func (c *cli) runFlag(ctx *command.Context, args command.ValidatedArgs) error {
	ch, err := c.Challenge(true)
	if err != nil {
		return err
	}

	if !args.Has("flag") {
		if ch.Flag == "" {
			ui.Notef("no flag set for %s", c.shortPath(ch.Path))
		} else {
			ui.Println(ui.Accent(ch.Flag))
		}
		return nil
	}

	flag := command.ArgValue[string](args, "flag")
	if flag == "" {
		return errors.New("flag cannot be empty")
	}
	if ev, err := c.EventFor(ch); err == nil && !ev.MatchesFlagFormat(flag) {
		ui.Notef("warning: flag doesn't match the event's format %s", ev.FlagFormat)
	}
	ch.SetFlag(flag)
	if err := ch.Save(); err != nil {
		return err
	}
	ui.Successf("flag set for %s  %s", ui.Bold(c.shortPath(ch.Path)), ui.Accent(flag))
	return nil
}

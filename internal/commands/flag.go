package commands

import (
	"errors"
	"fmt"

	"github.com/azuyamat/ctf/internal/challenge"
	"github.com/azuyamat/ctf/internal/config"
	"github.com/azuyamat/ctf/internal/state"
	"github.com/azuyamat/ctf/internal/ui"
	"github.com/azuyamat/gear/command"
)

func cmdFlag() command.Command {
	return command.NewExecutableCommand("flag", "Show or set the flag of the current challenge").
		Args(
			command.NewStringArg(
				"flag",
				"Flag string to set for the current challenge",
			).AsOptional(),
		).
		Handler(runFlag)
}

func runFlag(ctx *command.Context, args command.ValidatedArgs) error {
	ch, err := challenge.LoadCurrent(config.Current.BaseDir, state.Current.LastChallenge)
	if err != nil {
		return fmt.Errorf("load challenge: %w", err)
	}

	if !args.Has("flag") {
		if ch.Flag == "" {
			ui.Notef("no flag set for %s", shortPath(ch.Path))
		} else {
			ui.Println(ui.Accent(ch.Flag))
		}
		return nil
	}

	flag := args.String("flag")
	if flag == "" {
		return errors.New("flag cannot be empty")
	}
	ch.SetFlag(flag)
	if err := ch.Save(); err != nil {
		return err
	}
	ui.Successf("flag set for %s  %s", ui.Bold(shortPath(ch.Path)), ui.Accent(flag))
	return nil
}

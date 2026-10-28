package commands

import (
	"errors"
	"fmt"
	"os"

	"github.com/azuyamat/ctf/internal/challenge"
	"github.com/azuyamat/ctf/internal/config"
	"github.com/azuyamat/ctf/internal/state"
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
			fmt.Fprintf(os.Stderr, "no flag set for %s\n", ch.Path)
		} else {
			fmt.Println(ch.Flag)
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
	fmt.Printf("flag set for %s: %s\n", ch.Path, flag)
	return nil
}

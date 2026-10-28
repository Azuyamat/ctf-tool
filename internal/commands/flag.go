package commands

import (
	"fmt"
	"os"

	"github.com/azuyamat/ctf/internal/challenge"
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
	if !args.Has("flag") {
		challenge, err := getCurrentChallenge()
		if err != nil {
			return err
		}
		if challenge.Flag == "" {
			fmt.Fprintf(os.Stderr, "no flag set\n")
		} else {
			fmt.Fprintf(os.Stdout, "%s\n", challenge.Flag)
		}
		return nil
	}

	flag := args.String("flag")
	challenge, err := getCurrentChallenge()
	if err != nil {
		return err
	}
	challenge.SetFlag(flag)
	if err := challenge.Save(); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "flag set to %s\n", flag)
	return nil
}

func getCurrentChallenge() (*challenge.Challenge, error) {
	ch, err := challenge.LoadCurrent()
	if err != nil {
		return nil, fmt.Errorf("load challenge: %w", err)
	}
	return ch, nil
}

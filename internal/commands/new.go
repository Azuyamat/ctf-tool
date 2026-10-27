package commands

import (
	"fmt"

	"github.com/azuyamat/ctf/internal/challenge"
	"github.com/azuyamat/gear/command"
)

var CTF_FOLDER_NAME = "ctf"

func cmdNew() command.Command {
	return command.NewExecutableCommand("new", "Create a new CTF challenge folder").
		Args(
			command.NewStringArg(
				"challenge",
				"Challenge name, supports slashes",
			),
		).
		Handler(runNew)
}

func runNew(ctx *command.Context, args command.ValidatedArgs) error {
	challengeName := args.String("challenge")
	ch, err := challenge.New(challengeName)
	if err != nil {
		return err
	}

	err = ch.Create()
	if err != nil {
		return err
	}

	fmt.Printf("challenge created %s\n", ch.Path)

	return nil
}

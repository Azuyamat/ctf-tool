package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/azuyamat/ctf/internal/challenge"
	"github.com/azuyamat/gear/command"
)

func cmdLs() command.Command {
	return command.NewExecutableCommand("ls", "List all challenges in the current directory").
		Handler(runLs)
}

func runLs(ctx *command.Context, args command.ValidatedArgs) error {
	root, err := challenge.Root()
	if err != nil {
		return err
	}
	challenges, err := challenge.ListChallenges()
	if err != nil {
		return err
	}
	for _, ch := range challenges {
		var solved string
		if !ch.Solved.IsZero() {
			solved = "(solved)"
		}
		name, err := filepath.Rel(root, ch.Path)
		if err != nil {
			name = ch.Name
		}
		fmt.Fprintf(os.Stdout, "%s %s\n", name, solved)
	}
	return nil
}

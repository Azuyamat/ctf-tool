package commands

import (
	"fmt"
	"path/filepath"

	"github.com/azuyamat/ctf/internal/challenge"
	"github.com/azuyamat/gear/command"
)

func cmdLs() command.Command {
	return command.NewExecutableCommand("ls", "List challenges in the current event, or all if none is set").
		Handler(runLs)
}

func runLs(ctx *command.Context, args command.ValidatedArgs) error {
	root := challengeRoot()
	challenges, err := challenge.ListChallenges(root)
	if err != nil {
		return err
	}
	for _, ch := range challenges {
		name, err := filepath.Rel(root, ch.Path)
		if err != nil {
			name = ch.Name
		}
		if ch.Solved.IsZero() {
			fmt.Println(name)
		} else {
			fmt.Printf("%s (solved)\n", name)
		}
	}
	return nil
}

package commands

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/azuyamat/ctf/internal/challenge"
	"github.com/azuyamat/ctf/internal/config"
	"github.com/azuyamat/gear/command"
)

func cmdOpen() command.Command {
	return command.NewExecutableCommand("open", "Open the current challenge in $EDITOR").
		Handler(runOpen)
}

func runOpen(ctx *command.Context, args command.ValidatedArgs) error {
	ch, err := challenge.LoadFromCwd(config.Current.BaseDir)
	if err != nil {
		return fmt.Errorf("load challenge: %w", err)
	}

	editor := strings.Fields(os.Getenv("EDITOR"))
	if len(editor) == 0 {
		return errors.New("EDITOR is not set")
	}

	cmd := exec.Command(editor[0], append(editor[1:], ch.Path)...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

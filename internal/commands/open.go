package commands

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/azuyamat/gear/command"
)

func (c *cli) cmdOpen() command.Command {
	return command.NewExecutableCommand("open", "Open the current challenge in $EDITOR").
		Group("Challenges").
		Flags(
			command.NewBoolFlag("web", "w", "Open the challenge's page on the CTF site instead", false),
		).
		Handler(c.runOpen)
}

func (c *cli) runOpen(ctx *command.Context, args command.ValidatedArgs) error {
	ch, err := c.Challenge(false)
	if err != nil {
		return err
	}
	if command.FlagValue[bool](args, "web") {
		if ch.Remote == nil || ch.Remote.URL == "" {
			return fmt.Errorf("%s has no link, run ctf pull to get one", c.shortPath(ch.Path))
		}
		return openURL(ch.Remote.URL)
	}

	editor := strings.Fields(os.Getenv("EDITOR"))
	if len(editor) == 0 {
		return errors.New("EDITOR is not set")
	}

	cmd := exec.Command(editor[0], append(editor[1:], ch.Path)...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

func openURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open %s: %w", url, err)
	}
	return cmd.Process.Release()
}

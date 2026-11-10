package commands

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/azuyamat/ctf/internal/event"
	"github.com/azuyamat/ctf/internal/ui"
	"github.com/azuyamat/gear/command"
)

func (c *cli) cmdClone() command.Command {
	return command.NewExecutableCommand("clone", "Clone an event from a remote repository").
		Group("Events").
		Long("Clone a git repository as a new event and switch to it. Handy when a teammate has already pushed the event's challenge folders.").
		Example(
			"ctf clone git@github.com:team/picoctf-2026.git",
			"ctf clone https://github.com/team/picoctf-2026.git",
		).
		Args(
			command.NewStringArg("repository", "The repository to clone"),
		).
		Handler(c.runClone)
}

func (c *cli) runClone(ctx *command.Context, args command.ValidatedArgs) error {
	if _, err := exec.LookPath("git"); err != nil {
		return errors.New("git not found in PATH")
	}

	repo := command.ArgValue[string](args, "repository")
	name := repoName(repo)
	if name == "" {
		return fmt.Errorf("couldn't derive an event name from %q", repo)
	}

	ev, err := event.New(c.Layout(), name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(ev.Path); err == nil {
		return fmt.Errorf("event already exists: %s", ev.Name)
	}

	cmd := exec.CommandContext(ctx.Context(), "git", "clone", repo, ev.Path)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone: %w", err)
	}

	if err := ev.Create(); err != nil {
		os.RemoveAll(ev.Path)
		return err
	}
	ev.ReadmeName, ev.FilesDir = c.Config.ReadmeName, c.Config.FilesDir
	if err := ev.Save(); err != nil {
		os.RemoveAll(ev.Path)
		return err
	}

	c.State.Event = ev.Name

	ui.Successf("cloned and switched to %s", ui.Bold(ev.Name))
	return nil
}

// repoName derives a name from the last path segment of a git URL, handling
// both scp-like (git@host:user/repo.git) and URL forms.
func repoName(repo string) string {
	repo = strings.TrimSuffix(strings.TrimRight(repo, "/"), ".git")
	if i := strings.LastIndexAny(repo, "/:"); i != -1 {
		repo = repo[i+1:]
	}
	return repo
}

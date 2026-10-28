package commands

import (
	"errors"
	"fmt"

	"github.com/azuyamat/ctf/internal/challenge"
	"github.com/azuyamat/ctf/internal/connection"
	"github.com/azuyamat/ctf/internal/template"
	"github.com/azuyamat/ctf/internal/ui"
	"github.com/azuyamat/gear/command"
)

func (c *cli) cmdNew() command.Command {
	return command.NewExecutableCommand("new", "Create a new CTF challenge folder").
		Group("Challenges").
		Example(
			"ctf new web/login-bypass",
			"ctf new pwn/heap -t pwn -r 'nc chall.example.com 1337'",
		).
		Args(
			command.NewStringArg(
				"challenge",
				"Challenge name, supports slashes",
			),
		).
		Flags(
			command.NewStringFlag("template", "t", "Template to copy into the challenge (default: the defaultTemplate setting)", "").Complete(c.completeTemplates),
			command.NewStringFlag("remote", "r", "Where the challenge runs: host:port, \"nc host port\" or a URL", ""),
		).
		Handler(c.runNew)
}

func (c *cli) runNew(ctx *command.Context, args command.ValidatedArgs) error {
	ch, err := challenge.New(c.ChallengeRoot(), command.ArgValue[string](args, "challenge"))
	if err != nil {
		return err
	}

	if remote := command.FlagValue[string](args, "remote"); remote != "" {
		if connection.Parse(remote).Host == "" {
			return fmt.Errorf("invalid --remote %q, expected host:port, \"nc host port\" or a URL", remote)
		}
		ch.Connection = remote
	}

	tmpl, err := c.resolveTemplate(command.FlagValue[string](args, "template"))
	if err != nil {
		return err
	}

	if err := ch.Create(tmpl, c.State.Event); err != nil {
		return err
	}

	c.State.LastChallenge = ch.Path

	detail := ""
	if tmpl != nil {
		detail = "  " + ui.Dim(fmt.Sprintf("template %s · %s", tmpl.Name, tmpl.Scope))
	}
	ui.Successf("created %s%s", ui.Bold(c.shortPath(ch.Path)), detail)
	return nil
}

// resolveTemplate finds the named template. With no name it falls back to the
// defaultTemplate setting, returning nil without error if that is the built-in
// default and it doesn't exist.
func (c *cli) resolveTemplate(name string) (*template.Template, error) {
	l, event := c.Layout(), c.State.Event
	if name != "" {
		return template.Find(l, event, name)
	}

	name = c.Config.DefaultTemplate
	tmpl, err := template.Find(l, event, name)
	if errors.Is(err, template.ErrNotFound) && name == template.Default {
		return nil, nil
	}
	return tmpl, err
}

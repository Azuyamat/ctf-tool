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
		Args(
			command.NewStringArg(
				"challenge",
				"Challenge name, supports slashes",
			),
		).
		Flags(
			command.NewStringFlag("template", "t", "Template to copy into the challenge (default: \"default\" if it exists)", ""),
			command.NewStringFlag("remote", "r", "Where the challenge runs: host:port, \"nc host port\" or a URL", ""),
		).
		Handler(c.runNew)
}

func (c *cli) runNew(ctx *command.Context, args command.ValidatedArgs) error {
	ch, err := challenge.New(c.ChallengeRoot(), args.String("challenge"))
	if err != nil {
		return err
	}

	if remote := args.FlagString("remote"); remote != "" {
		if connection.Parse(remote).Host == "" {
			return fmt.Errorf("invalid --remote %q, expected host:port, \"nc host port\" or a URL", remote)
		}
		ch.Connection = remote
	}

	tmpl, err := c.resolveTemplate(args.FlagString("template"))
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
// default template, returning nil without error if there is none.
func (c *cli) resolveTemplate(name string) (*template.Template, error) {
	l, event := c.Layout(), c.State.Event
	if name != "" {
		return template.Find(l, event, name)
	}

	tmpl, err := template.Find(l, event, template.Default)
	if errors.Is(err, template.ErrNotFound) {
		return nil, nil
	}
	return tmpl, err
}

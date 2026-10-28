package commands

import (
	"errors"
	"fmt"

	"github.com/azuyamat/ctf/internal/challenge"
	"github.com/azuyamat/ctf/internal/config"
	"github.com/azuyamat/ctf/internal/state"
	"github.com/azuyamat/ctf/internal/template"
	"github.com/azuyamat/ctf/internal/ui"
	"github.com/azuyamat/gear/command"
)

func cmdNew() command.Command {
	return command.NewExecutableCommand("new", "Create a new CTF challenge folder").
		Args(
			command.NewStringArg(
				"challenge",
				"Challenge name, supports slashes",
			),
		).
		Flags(
			command.NewStringFlag("template", "t", "Template to copy into the challenge (default: \"default\" if it exists)", ""),
		).
		Handler(runNew)
}

func runNew(ctx *command.Context, args command.ValidatedArgs) error {
	ch, err := challenge.New(challengeRoot(), args.String("challenge"))
	if err != nil {
		return err
	}

	tmpl, err := resolveTemplate(args.FlagString("template"))
	if err != nil {
		return err
	}

	data := template.Data{Name: ch.Name, Event: state.Current.Event}
	if err := ch.Create(tmpl, data); err != nil {
		return err
	}

	state.Current.LastChallenge = ch.Path

	detail := ""
	if tmpl != nil {
		detail = "  " + ui.Dim(fmt.Sprintf("template %s · %s", tmpl.Name, tmpl.Scope))
	}
	ui.Successf("created %s%s", ui.Bold(shortPath(ch.Path)), detail)
	return nil
}

// resolveTemplate finds the named template. With no name it falls back to the
// default template, returning nil without error if there is none.
func resolveTemplate(name string) (*template.Template, error) {
	base, event := config.Current.BaseDir, state.Current.Event
	if name != "" {
		return template.Find(base, event, name)
	}

	tmpl, err := template.Find(base, event, template.Default)
	if errors.Is(err, template.ErrNotFound) {
		return nil, nil
	}
	return tmpl, err
}

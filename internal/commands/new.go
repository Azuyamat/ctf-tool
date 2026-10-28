package commands

import (
	"errors"
	"fmt"

	"github.com/azuyamat/ctf/internal/challenge"
	"github.com/azuyamat/ctf/internal/state"
	"github.com/azuyamat/ctf/internal/template"
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
	challengeName := args.String("challenge")
	ch, err := challenge.New(challengeName)
	if err != nil {
		return err
	}

	tmpl, err := resolveTemplate(args.FlagString("template"))
	if err != nil {
		return err
	}

	err = ch.Create(tmpl)
	if err != nil {
		return err
	}

	state.Current.LastChallenge = ch.Path

	if tmpl != nil {
		fmt.Printf("challenge created %s (template %s, %s)\n", ch.Path, tmpl.Name, tmpl.Scope)
	} else {
		fmt.Printf("challenge created %s\n", ch.Path)
	}

	return nil
}

// resolveTemplate finds the named template. With no name it falls back to the
// default template, returning nil without error if there is none.
func resolveTemplate(name string) (*template.Template, error) {
	if name != "" {
		return template.Find(name)
	}

	tmpl, err := template.Find(template.Default)
	if errors.Is(err, template.ErrNotFound) {
		return nil, nil
	}
	return tmpl, err
}

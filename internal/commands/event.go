package commands

import (
	"errors"
	"strings"

	"github.com/azuyamat/ctf/internal/challenge"
	"github.com/azuyamat/ctf/internal/event"
	"github.com/azuyamat/ctf/internal/fsutil"
	"github.com/azuyamat/ctf/internal/ui"
	"github.com/azuyamat/gear/command"
)

func (c *cli) cmdEvent() command.Command {
	return command.NewExecutableCommand("event", "Show or set the current CTF event").
		Group("Events").
		Long("Show the current event, or switch to another one. New challenges are created under the current event's folder.\n\nLink an event to its CTFd instance with --ctfd to pull challenges and submit flags from the terminal.").
		Example(
			"ctf event picoctf-2026",
			"# link the current event to its CTFd instance\nctf event --ctfd https://play.picoctf.org",
			"ctf event --flag-format 'picoCTF\\{.+\\}'",
		).
		Args(
			command.NewStringArg(
				"name",
				"Event name, challenges are created under it",
			).AsOptional(),
		).
		Flags(
			command.NewBoolFlag("clear", "c", "Unset the current event", false),
			command.NewStringFlag("ctfd", "", "Link the event to a CTFd instance URL", ""),
			command.NewBoolFlag("insecure", "", "Allow a plain http URL", false).Requires("ctfd"),
			command.NewBoolFlag("unlink", "", "Remove the event's platform link", false),
			command.NewStringFlag("flag-format", "", "Regular expression flags must match, e.g. 'picoCTF\\{.+\\}'", ""),
			command.NewBoolFlag("no-flag-format", "", "Remove the event's flag format", false),
		).
		MutuallyExclusive("ctfd", "unlink").
		MutuallyExclusive("flag-format", "no-flag-format").
		Handler(c.runEvent)
}

func (c *cli) runEvent(ctx *command.Context, args command.ValidatedArgs) error {
	if command.FlagValue[bool](args, "clear") {
		c.State.Event = ""
		ui.Successf("event cleared")
		return nil
	}

	rawURL, unlink := command.FlagValue[string](args, "ctfd"), command.FlagValue[bool](args, "unlink")
	format, noFormat := command.FlagValue[string](args, "flag-format"), command.FlagValue[bool](args, "no-flag-format")
	editing := rawURL != "" || unlink || format != "" || noFormat

	name := c.State.Event
	if args.Has("name") {
		name = command.ArgValue[string](args, "name")
	} else if !editing {
		return c.printEventSummary()
	} else if name == "" {
		return errors.New("no event set, pass an event name")
	}

	ev, err := event.Load(c.Layout(), name)
	if err != nil {
		return err
	}
	isNew := !ev.Exists()
	if err := ev.Create(); err != nil {
		return err
	}
	if isNew {
		ev.ReadmeName, ev.FilesDir = c.Config.ReadmeName, c.Config.FilesDir
		if err := ev.Save(); err != nil {
			return err
		}
	}

	switch {
	case rawURL != "":
		if err := ev.Link(event.PlatformCTFd, rawURL, command.FlagValue[bool](args, "insecure")); err != nil {
			return err
		}
		if err := ev.Save(); err != nil {
			return err
		}
	case unlink:
		if !ev.Unlink() {
			ui.Notef("%s is not linked", ev.Name)
		} else if err := ev.Save(); err != nil {
			return err
		} else {
			ui.Successf("unlinked %s", ui.Bold(ev.Name))
		}
	}

	if format != "" || noFormat {
		if noFormat {
			ev.FlagFormat = ""
		} else if err := ev.SetFlagFormat(format); err != nil {
			return err
		}
		if err := ev.Save(); err != nil {
			return err
		}
	}

	c.State.Event = ev.Name
	var details []string
	if ev.Platform != nil {
		details = append(details, platformLabel(ev.Platform))
	}
	if ev.FlagFormat != "" {
		details = append(details, "flag "+ev.FlagFormat)
	}
	detail := ""
	if len(details) > 0 {
		detail = "  " + ui.Dim(strings.Join(details, " · "))
	}
	ui.Successf("event set to %s%s", ui.Bold(ev.Name), detail)
	return nil
}

func platformLabel(p *event.Platform) string {
	return p.Kind + " " + p.Host()
}

func (c *cli) printEventSummary() error {
	if c.State.Event == "" {
		ui.Notef("no event set")
		return nil
	}

	ev, err := event.Load(c.Layout(), c.State.Event)
	if err != nil {
		return err
	}

	root := c.ChallengeRoot()
	challenges, err := challenge.ListChallenges(root)
	if err != nil {
		return err
	}

	parts := []string{ui.Bold(ev.Name)}
	if ev.Platform != nil {
		parts = append(parts, platformLabel(ev.Platform))
	}
	parts = append(parts, progressLine(challenges, false))
	if last, ok := fsutil.Within(root, c.State.LastChallenge); ok {
		parts = append(parts, "last: "+last)
	}
	ui.Println(strings.Join(parts, ui.Dim(" · ")))
	return nil
}

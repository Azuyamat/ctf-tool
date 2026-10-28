package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/azuyamat/ctf/internal/challenge"
	"github.com/azuyamat/ctf/internal/config"
	"github.com/azuyamat/ctf/internal/fsutil"
	"github.com/azuyamat/ctf/internal/state"
	"github.com/azuyamat/ctf/internal/template"
	"github.com/azuyamat/ctf/internal/ui"
	"github.com/azuyamat/gear/command"
)

func cmdEvent() command.Command {
	return command.NewExecutableCommand("event", "Show or set the current CTF event").
		Args(
			command.NewStringArg(
				"name",
				"Event name, challenges are created under it",
			).AsOptional(),
		).
		Flags(
			command.NewBoolFlag("clear", "c", "Unset the current event", false),
		).
		Handler(runEvent)
}

func runEvent(ctx *command.Context, args command.ValidatedArgs) error {
	if args.FlagBool("clear") {
		state.Current.Event = ""
		ui.Successf("event cleared")
		return nil
	}

	if !args.Has("name") {
		return printEventSummary()
	}

	name := args.String("name")
	if err := validateEventName(name); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(config.Current.BaseDir, name), 0755); err != nil {
		return err
	}

	state.Current.Event = name
	ui.Successf("event set to %s", ui.Bold(name))
	return nil
}

func printEventSummary() error {
	if state.Current.Event == "" {
		ui.Notef("no event set")
		return nil
	}

	root := challengeRoot()
	challenges, err := challenge.ListChallenges(root)
	if err != nil {
		return err
	}

	parts := []string{
		ui.Bold(state.Current.Event),
		fmt.Sprintf("%d/%d solved", countSolved(challenges), len(challenges)),
	}
	if last, ok := fsutil.Within(root, state.Current.LastChallenge); ok {
		parts = append(parts, "last: "+last)
	}
	ui.Println(strings.Join(parts, ui.Dim(" · ")))
	return nil
}

func validateEventName(name string) error {
	if !fsutil.ValidName(name) {
		return fmt.Errorf("invalid event name: %s", name)
	}
	if name == template.DirName {
		return fmt.Errorf("event name is reserved: %s", name)
	}
	return nil
}

package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/azuyamat/ctf/internal/config"
	"github.com/azuyamat/ctf/internal/fsutil"
	"github.com/azuyamat/ctf/internal/state"
	"github.com/azuyamat/ctf/internal/template"
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
		fmt.Println("event cleared")
		return nil
	}

	if !args.Has("name") {
		if state.Current.Event == "" {
			fmt.Println("no event set")
		} else {
			fmt.Println(state.Current.Event)
		}
		return nil
	}

	name := args.String("name")
	if err := validateEventName(name); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(config.Current.BaseDir, name), 0755); err != nil {
		return err
	}

	state.Current.Event = name
	fmt.Printf("event set to %s\n", name)

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

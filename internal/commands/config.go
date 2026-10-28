package commands

import (
	"cmp"
	"fmt"

	"github.com/azuyamat/ctf/internal/config"
	"github.com/azuyamat/ctf/internal/event"
	"github.com/azuyamat/ctf/internal/ui"
	"github.com/azuyamat/gear/command"
)

func (c *cli) cmdConfig() command.Command {
	return command.NewSubcommand("config", "Manage configuration").
		Group("Setup").
		AddChild(c.cmdConfigSet()).
		AddChild(c.cmdConfigGet()).
		AddChild(c.cmdConfigUnset()).
		AddChild(c.cmdConfigList())
}

func eventFlag(description string) command.Flag {
	return command.NewBoolFlag("event", "e", description, false)
}

func (c *cli) cmdConfigSet() command.Command {
	return command.NewExecutableCommand("set", "Set a configuration value").
		Args(
			command.NewStringArg("key", "Configuration key to set").Complete(completeConfigKeys),
			command.NewStringArg("value", "Value to set for the configuration key"),
		).
		Flags(eventFlag("Set it for the current event only")).
		Example(
			"ctf config set editor 'code -r'",
			"ctf config set --event readmeName CHALLENGE.md",
		).
		Handler(c.runConfigSet)
}

func (c *cli) cmdConfigGet() command.Command {
	return command.NewExecutableCommand("get", "Print the value in use, or the event's own with --event").
		Args(
			command.NewStringArg("key", "Configuration key to get").Complete(completeConfigKeys),
		).
		Flags(eventFlag("Print the current event's own value")).
		Handler(c.runConfigGet)
}

func (c *cli) cmdConfigUnset() command.Command {
	return command.NewExecutableCommand("unset", "Reset a value to its default, or remove the event's own with --event").
		Args(
			command.NewStringArg("key", "Configuration key to unset").Complete(completeConfigKeys),
		).
		Flags(eventFlag("Remove the current event's value so the global one applies")).
		Handler(c.runConfigUnset)
}

func (c *cli) cmdConfigList() command.Command {
	return command.NewExecutableCommand("list", "List the values in use and where they come from").
		Handler(c.runConfigList)
}

func completeConfigKeys(_ *command.Context, args command.ValidatedArgs, _ string) []string {
	eventOnly := command.FlagValue[bool](args, "event")
	var out []string
	for _, key := range config.Keys {
		if !eventOnly || key.EventScoped() {
			out = append(out, key.Name+"\t"+key.Description)
		}
	}
	return out
}

func findKey(name string) (*config.Key, error) {
	key := config.FindKey(name)
	if key == nil {
		return nil, fmt.Errorf("unknown configuration key: %s", name)
	}
	return key, nil
}

func (c *cli) eventKey(args command.ValidatedArgs) (*config.Key, *event.Event, error) {
	key, err := findKey(command.ArgValue[string](args, "key"))
	if err != nil {
		return nil, nil, err
	}
	if !command.FlagValue[bool](args, "event") {
		return key, nil, nil
	}
	if !key.EventScoped() {
		return nil, nil, fmt.Errorf("%s can only be set globally", key.Name)
	}
	ev, err := c.Event()
	if err != nil {
		return nil, nil, err
	}
	return key, ev, nil
}

func (c *cli) currentEvent() *event.Event {
	ev, err := c.Event()
	if err != nil {
		return nil
	}
	return ev
}

func (c *cli) runConfigSet(ctx *command.Context, args command.ValidatedArgs) error {
	key, ev, err := c.eventKey(args)
	if err != nil {
		return err
	}
	value := command.ArgValue[string](args, "value")
	if ev != nil {
		if value == "" {
			return fmt.Errorf("value cannot be empty, use ctf config unset --event %s", key.Name)
		}
		return c.saveEventValue(key, ev, value)
	}

	if err := key.Set(c.Config, value); err != nil {
		return err
	}
	if err := c.Config.Validate(); err != nil {
		return err
	}
	current := c.currentEvent()
	if key.EventScoped() && current != nil {
		if err := c.Config.Paths(current).Validate(); err != nil {
			return fmt.Errorf("%s: %w", current.Name, err)
		}
	}
	if err := c.Config.Save(); err != nil {
		return err
	}

	ui.Successf("%s set to %s", ui.Bold(key.Name), cmp.Or(key.Get(c.Config), ui.Dim("(empty)")))
	if key.EventScoped() && current != nil && key.GetEvent(current) != "" {
		ui.Notef("%s keeps its own %s (%s), use --event to change it", current.Name, key.Name, key.GetEvent(current))
	}
	return nil
}

func (c *cli) saveEventValue(key *config.Key, ev *event.Event, value string) error {
	if err := key.SetEvent(ev, value); err != nil {
		return err
	}
	if err := c.Config.Paths(ev).Validate(); err != nil {
		return err
	}
	if err := ev.Save(); err != nil {
		return err
	}
	if value == "" {
		ui.Successf("%s unset for %s, using %s", ui.Bold(key.Name), ui.Bold(ev.Name), key.Get(c.Config))
	} else {
		ui.Successf("%s set to %s for %s", ui.Bold(key.Name), value, ui.Bold(ev.Name))
	}
	return nil
}

func (c *cli) runConfigGet(ctx *command.Context, args command.ValidatedArgs) error {
	key, ev, err := c.eventKey(args)
	if err != nil {
		return err
	}
	if ev != nil {
		fmt.Println(key.GetEvent(ev))
		return nil
	}
	value, _ := c.resolve(key, c.currentEvent())
	fmt.Println(value)
	return nil
}

func (c *cli) runConfigUnset(ctx *command.Context, args command.ValidatedArgs) error {
	key, ev, err := c.eventKey(args)
	if err != nil {
		return err
	}
	if ev != nil {
		return c.saveEventValue(key, ev, "")
	}

	defaults, err := config.Defaults()
	if err != nil {
		return err
	}
	if err := key.Set(c.Config, key.Get(defaults)); err != nil {
		return err
	}
	if err := c.Config.Validate(); err != nil {
		return err
	}
	if err := c.Config.Save(); err != nil {
		return err
	}
	ui.Successf("%s reset to %s", ui.Bold(key.Name), cmp.Or(key.Get(c.Config), ui.Dim("(empty)")))
	return nil
}

func (c *cli) resolve(key *config.Key, ev *event.Event) (string, string) {
	if key.EventScoped() && ev != nil {
		if value := key.GetEvent(ev); value != "" {
			return value, "event"
		}
	}
	value := key.Get(c.Config)
	if defaults, err := config.Defaults(); err == nil && key.Get(defaults) == value {
		return value, "default"
	}
	return value, "global"
}

func (c *cli) runConfigList(ctx *command.Context, args command.ValidatedArgs) error {
	ev := c.currentEvent()
	rows := make([][]string, len(config.Keys))
	for i, key := range config.Keys {
		value, source := c.resolve(&key, ev)
		if source == "event" {
			source = "event " + ev.Name
			if global := key.Get(c.Config); global != value {
				source += ui.Dim(" (global: " + global + ")")
			}
		}
		rows[i] = []string{ui.Bold(key.Name), value, ui.Accent(source), ui.Dim(key.Description)}
	}
	ui.Println(ui.Table(rows))
	return nil
}

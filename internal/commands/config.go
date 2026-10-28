package commands

import (
	"fmt"

	"github.com/azuyamat/ctf/internal/config"
	"github.com/azuyamat/ctf/internal/ui"
	"github.com/azuyamat/gear/command"
)

func (c *cli) cmdConfig() command.Command {
	return command.NewSubcommand("config", "Manage configuration").
		AddChild(c.cmdConfigSet()).
		AddChild(c.cmdConfigGet()).
		AddChild(c.cmdConfigList())
}

func (c *cli) cmdConfigSet() command.Command {
	return command.NewExecutableCommand("set", "Set a configuration value").
		Args(
			command.NewStringArg("key", "Configuration key to set"),
			command.NewStringArg("value", "Value to set for the configuration key"),
		).
		Handler(c.runConfigSet)
}

func (c *cli) cmdConfigGet() command.Command {
	return command.NewExecutableCommand("get", "Get a configuration value").
		Args(
			command.NewStringArg("key", "Configuration key to get"),
		).
		Handler(c.runConfigGet)
}

func (c *cli) cmdConfigList() command.Command {
	return command.NewExecutableCommand("list", "List all configuration values").
		Handler(c.runConfigList)
}

func findKey(name string) (*config.Key, error) {
	key := config.FindKey(name)
	if key == nil {
		return nil, fmt.Errorf("unknown configuration key: %s", name)
	}
	return key, nil
}

func (c *cli) runConfigSet(ctx *command.Context, args command.ValidatedArgs) error {
	key, err := findKey(args.String("key"))
	if err != nil {
		return err
	}

	if err := key.Set(c.Config, args.String("value")); err != nil {
		return err
	}
	if err := c.Config.Save(); err != nil {
		return err
	}

	ui.Successf("%s set to %s", ui.Bold(key.Name), key.Get(c.Config))
	return nil
}

func (c *cli) runConfigGet(ctx *command.Context, args command.ValidatedArgs) error {
	key, err := findKey(args.String("key"))
	if err != nil {
		return err
	}

	fmt.Println(key.Get(c.Config))
	return nil
}

func (c *cli) runConfigList(ctx *command.Context, args command.ValidatedArgs) error {
	rows := make([][]string, len(config.Keys))
	for i, key := range config.Keys {
		rows[i] = []string{ui.Bold(key.Name), key.Get(c.Config), ui.Dim(key.Description)}
	}
	ui.Println(ui.Table(rows))
	return nil
}

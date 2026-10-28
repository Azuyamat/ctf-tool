package commands

import (
	"fmt"

	"github.com/azuyamat/ctf/internal/config"
	"github.com/azuyamat/ctf/internal/ui"
	"github.com/azuyamat/gear/command"
)

func cmdConfig() command.Command {
	return command.NewSubcommand("config", "Manage configuration").
		AddChild(cmdConfigSet()).
		AddChild(cmdConfigGet()).
		AddChild(cmdConfigList())
}

func cmdConfigSet() command.Command {
	return command.NewExecutableCommand("set", "Set a configuration value").
		Args(
			command.NewStringArg("key", "Configuration key to set"),
			command.NewStringArg("value", "Value to set for the configuration key"),
		).
		Handler(runConfigSet)
}

func cmdConfigGet() command.Command {
	return command.NewExecutableCommand("get", "Get a configuration value").
		Args(
			command.NewStringArg("key", "Configuration key to get"),
		).
		Handler(runConfigGet)
}

func cmdConfigList() command.Command {
	return command.NewExecutableCommand("list", "List all configuration values").
		Handler(runConfigList)
}

func findKey(name string) (*config.Key, error) {
	key := config.FindKey(name)
	if key == nil {
		return nil, fmt.Errorf("unknown configuration key: %s", name)
	}
	return key, nil
}

func runConfigSet(ctx *command.Context, args command.ValidatedArgs) error {
	key, err := findKey(args.String("key"))
	if err != nil {
		return err
	}

	if err := key.Set(config.Current, args.String("value")); err != nil {
		return err
	}
	if err := config.Current.Save(); err != nil {
		return err
	}

	ui.Successf("%s set to %s", ui.Bold(key.Name), key.Get(config.Current))
	return nil
}

func runConfigGet(ctx *command.Context, args command.ValidatedArgs) error {
	key, err := findKey(args.String("key"))
	if err != nil {
		return err
	}

	fmt.Println(key.Get(config.Current))
	return nil
}

func runConfigList(ctx *command.Context, args command.ValidatedArgs) error {
	rows := make([][]string, len(config.Keys))
	for i, key := range config.Keys {
		rows[i] = []string{ui.Bold(key.Name), key.Get(config.Current), ui.Dim(key.Description)}
	}
	ui.Println(ui.Table(rows))
	return nil
}

package commands

import (
	"fmt"

	"github.com/azuyamat/ctf/internal/config"
	"github.com/azuyamat/gear/command"
)

func cmdConfig() command.Command {
	return command.NewSubcommand("config", "Manage configuration").AddChild(cmdConfigSet()).AddChild(cmdConfigGet()).AddChild(cmdConfigList())
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

func runConfigSet(ctx *command.Context, args command.ValidatedArgs) error {
	key := args.String("key")
	value := args.String("value")

	keyDef := config.FindKey(key)
	if keyDef == nil {
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	if err := keyDef.Set(config.Current, value); err != nil {
		return err
	}

	fmt.Printf("configuration key '%s' set to '%s'\n", key, value)
	return nil
}

func runConfigGet(ctx *command.Context, args command.ValidatedArgs) error {
	key := args.String("key")

	keyDef := config.FindKey(key)
	if keyDef == nil {
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	value := keyDef.Get(config.Current)
	fmt.Printf("%s\n", value)
	return nil
}

func runConfigList(ctx *command.Context, args command.ValidatedArgs) error {
	for _, keyDef := range config.Keys {
		value := keyDef.Get(config.Current)
		fmt.Printf("%s: %s\n", keyDef.Name, value)
	}
	return nil
}

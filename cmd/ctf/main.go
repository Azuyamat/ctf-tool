package main

import (
	"os"
	"runtime/debug"

	"github.com/azuyamat/ctf/internal/commands"
	"github.com/azuyamat/ctf/internal/config"
	"github.com/azuyamat/ctf/internal/state"
	"github.com/azuyamat/ctf/internal/ui"
)

var version = "dev"

func buildVersion() string {
	if version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return version
}

func main() {
	if err := run(); err != nil {
		ui.Error(err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.LoadOrCreateConfig()
	if err != nil {
		return err
	}
	config.Current = cfg

	st, err := state.Load()
	if err != nil {
		return err
	}
	state.Current = st

	if err := commands.Root(buildVersion()).Run(os.Args[1:]); err != nil {
		return err
	}

	// Config is saved by the commands that change it; state changes on most runs.
	return state.Current.Save()
}

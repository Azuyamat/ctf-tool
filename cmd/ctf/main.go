package main

import (
	"fmt"
	"os"

	"github.com/azuyamat/ctf/internal/commands"
	"github.com/azuyamat/ctf/internal/config"
	"github.com/azuyamat/ctf/internal/state"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
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

	if err := commands.Root().Run(os.Args[1:]); err != nil {
		return err
	}

	// Config is saved by the commands that change it; state changes on most runs.
	return state.Current.Save()
}

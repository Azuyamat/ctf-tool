package main

import (
	"fmt"
	"os"

	"github.com/azuyamat/ctf/internal/commands"
	"github.com/azuyamat/ctf/internal/config"
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

	if err := commands.Root().Run(os.Args[1:]); err != nil {
		return err
	}

	return config.Current.Save()
}

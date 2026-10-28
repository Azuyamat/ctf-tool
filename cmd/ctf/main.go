package main

import (
	"os"
	"runtime/debug"

	"github.com/azuyamat/ctf/internal/app"
	"github.com/azuyamat/ctf/internal/commands"
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
	a, err := app.Load(buildVersion())
	if err != nil {
		return err
	}

	if err := commands.Root(a).Run(os.Args[1:]); err != nil {
		return err
	}

	// Config is saved by the commands that change it; state changes on most runs.
	return a.Save()
}

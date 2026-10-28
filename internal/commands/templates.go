package commands

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/azuyamat/ctf/internal/config"
	"github.com/azuyamat/ctf/internal/state"
	"github.com/azuyamat/ctf/internal/template"
	"github.com/azuyamat/gear/command"
)

func cmdTemplates() command.Command {
	return command.NewExecutableCommand("templates", "List available challenge templates").
		Handler(runTemplates)
}

func runTemplates(ctx *command.Context, args command.ValidatedArgs) error {
	templates, err := template.List(config.Current.BaseDir, state.Current.Event)
	if err != nil {
		return err
	}

	if len(templates) == 0 {
		fmt.Println("no templates found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, t := range templates {
		fmt.Fprintf(w, "%s\t%s\t%s\n", t.Name, t.Scope, t.Path)
	}
	return w.Flush()
}

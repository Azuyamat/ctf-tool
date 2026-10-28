package commands

import (
	"charm.land/lipgloss/v2/tree"
	"github.com/azuyamat/ctf/internal/config"
	"github.com/azuyamat/ctf/internal/state"
	"github.com/azuyamat/ctf/internal/template"
	"github.com/azuyamat/ctf/internal/ui"
	"github.com/azuyamat/gear/command"
)

func cmdTemplates() command.Command {
	return command.NewExecutableCommand("templates", "List available challenge templates").
		Handler(runTemplates)
}

func runTemplates(ctx *command.Context, args command.ValidatedArgs) error {
	base, event := config.Current.BaseDir, state.Current.Event
	templates, err := template.List(base, event)
	if err != nil {
		return err
	}

	if len(templates) == 0 {
		ui.Notef("no templates found")
		return nil
	}

	root := scopeTree(template.GlobalDir(base), template.ScopeGlobal)
	eventTree := scopeTree(template.EventDir(base, event), template.ScopeEvent)
	for _, t := range templates {
		if t.Scope == template.ScopeEvent {
			eventTree.Child(templateLeaf(t))
		} else {
			root.Child(templateLeaf(t))
		}
	}
	if eventTree.Children().Length() > 0 {
		root.Child(eventTree)
	}

	ui.Println(root)
	return nil
}

func scopeTree(dir, scope string) *tree.Tree {
	return ui.Tree(ui.Bold(shortPath(dir)) + "  " + ui.Dim(scope))
}

func templateLeaf(t template.Template) string {
	if t.Shadowed {
		return ui.Dim(t.Name + "  overridden by event")
	}
	return t.Name
}

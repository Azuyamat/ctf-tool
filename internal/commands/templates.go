package commands

import (
	"charm.land/lipgloss/v2/tree"
	"github.com/azuyamat/ctf/internal/template"
	"github.com/azuyamat/ctf/internal/ui"
	"github.com/azuyamat/gear/command"
)

func (c *cli) cmdTemplates() command.Command {
	return command.NewExecutableCommand("templates", "List available challenge templates").
		Handler(c.runTemplates)
}

func (c *cli) runTemplates(ctx *command.Context, args command.ValidatedArgs) error {
	l, event := c.Layout(), c.State.Event
	templates, err := template.List(l, event)
	if err != nil {
		return err
	}

	if len(templates) == 0 {
		ui.Notef("no templates found")
		return nil
	}

	root := c.scopeTree(l.GlobalTemplatesDir(), template.ScopeGlobal)
	eventTree := c.scopeTree(l.EventTemplatesDir(event), template.ScopeEvent)
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

func (c *cli) scopeTree(dir, scope string) *tree.Tree {
	return ui.Tree(ui.Bold(c.shortPath(dir)) + "  " + ui.Dim(scope))
}

func templateLeaf(t template.Template) string {
	if t.Shadowed {
		return ui.Dim(t.Name + "  overridden by event")
	}
	return t.Name
}

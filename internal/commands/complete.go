package commands

import (
	"slices"
	"strings"

	"github.com/azuyamat/ctf/internal/challenge"
	"github.com/azuyamat/ctf/internal/template"
	"github.com/azuyamat/gear/command"
)

func (c *cli) completeCategories(*command.Context, command.ValidatedArgs, string) []string {
	root := c.ChallengeRoot()
	challenges, err := challenge.ListChallenges(root)
	if err != nil {
		return nil
	}
	var categories []string
	for _, e := range listEntries(root, challenges) {
		for _, category := range []string{e.category, e.ch.Category()} {
			if category != "" && !slices.ContainsFunc(categories, func(s string) bool { return strings.EqualFold(s, category) }) {
				categories = append(categories, category)
			}
		}
	}
	return categories
}

func (c *cli) completeTemplates(*command.Context, command.ValidatedArgs, string) []string {
	templates, err := template.List(c.Layout(), c.State.Event)
	if err != nil {
		return nil
	}
	var names []string
	for _, t := range templates {
		if !t.Shadowed {
			names = append(names, t.Name+"\t"+t.Scope)
		}
	}
	return names
}

package commands

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/azuyamat/ctf/internal/challenge"
	"github.com/azuyamat/ctf/internal/ui"
	"github.com/azuyamat/gear/command"
	"github.com/charmbracelet/x/ansi"
)

const (
	progressWidth = 12
	flagWidth     = 24
)

func cmdLs() command.Command {
	return command.NewExecutableCommand("ls", "List challenges in the current event, or all if none is set").
		Handler(runLs)
}

func runLs(ctx *command.Context, args command.ValidatedArgs) error {
	root := challengeRoot()
	challenges, err := challenge.ListChallenges(root)
	if err != nil {
		return err
	}

	header := ui.Bold(shortPath(root))
	if len(challenges) == 0 {
		ui.Println(header + "  " + ui.Dim("no challenges yet"))
		return nil
	}

	solved := countSolved(challenges)
	progress := fmt.Sprintf("%d/%d solved", solved, len(challenges))
	ui.Println(header + "  " + ui.Bar(solved, len(challenges), progressWidth) + "  " + progress)
	ui.Println()
	ui.Println(ui.Table(challengeRows(root, challenges)))
	return nil
}

func challengeRows(root string, challenges []*challenge.Challenge) [][]string {
	var categories []string
	var ungrouped [][]string
	grouped := make(map[string][][]string)
	for _, ch := range challenges {
		rel, err := filepath.Rel(root, ch.Path)
		if err != nil {
			rel = ch.Name
		}
		category, name, ok := strings.Cut(filepath.ToSlash(rel), "/")
		if !ok {
			ungrouped = append(ungrouped, challengeRow("", category, ch))
			continue
		}
		if _, seen := grouped[category]; !seen {
			categories = append(categories, category)
		}
		grouped[category] = append(grouped[category], challengeRow("  ", name, ch))
	}

	table := ungrouped
	for _, category := range categories {
		table = append(table, []string{ui.Bold(category)})
		table = append(table, grouped[category]...)
	}
	return table
}

func challengeRow(indent, name string, ch *challenge.Challenge) []string {
	if !ch.IsSolved() {
		return []string{indent + ui.Dim("·") + " " + name}
	}
	return []string{
		indent + ui.Green("✓") + " " + name,
		ui.Dim(ui.Ago(ch.Solved)),
		ui.Accent(ansi.Truncate(ch.Flag, flagWidth, "…")),
	}
}

func countSolved(challenges []*challenge.Challenge) int {
	n := 0
	for _, ch := range challenges {
		if ch.IsSolved() {
			n++
		}
	}
	return n
}

package commands

import (
	"cmp"
	"fmt"
	"path/filepath"
	"slices"
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

func (c *cli) cmdLs() command.Command {
	return command.NewExecutableCommand("ls", "List challenges in the current event, or all if none is set").
		Args(
			command.NewStringArg("category", "Only list this category").AsOptional(),
		).
		Flags(
			command.NewBoolFlag("unsolved", "u", "Only list unsolved challenges", false),
		).
		Handler(c.runLs)
}

type listed struct {
	category string
	name     string
	ch       *challenge.Challenge
}

func (c *cli) runLs(ctx *command.Context, args command.ValidatedArgs) error {
	root := c.ChallengeRoot()
	challenges, err := challenge.ListChallenges(root)
	if err != nil {
		return err
	}

	header := ui.Bold(c.shortPath(root))
	if len(challenges) == 0 {
		ui.Println(header + "  " + ui.Dim("no challenges yet"))
		return nil
	}

	entries := listEntries(root, challenges)
	if args.Has("category") {
		want := args.String("category")
		entries = slices.DeleteFunc(entries, func(e listed) bool {
			return !strings.EqualFold(e.category, want) && !strings.EqualFold(e.ch.Category(), want)
		})
		if len(entries) == 0 {
			ui.Println(header + "  " + ui.Dim("no challenges in "+want))
			return nil
		}
		header += ui.Dim(" · " + want)
	}

	shown := make([]*challenge.Challenge, len(entries))
	for i, e := range entries {
		shown[i] = e.ch
	}
	ui.Println(header + "  " + progressLine(shown, true))

	if args.FlagBool("unsolved") {
		entries = slices.DeleteFunc(entries, func(e listed) bool { return e.ch.IsSolved() })
		if len(entries) == 0 {
			ui.Println()
			ui.Println(ui.Dim("everything is solved"))
			return nil
		}
	}
	ui.Println()
	ui.Println(ui.Table(challengeRows(entries)))
	return nil
}

func listEntries(root string, challenges []*challenge.Challenge) []listed {
	entries := make([]listed, len(challenges))
	for i, ch := range challenges {
		rel, err := filepath.Rel(root, ch.Path)
		if err != nil {
			rel = ch.Name
		}
		category, name, ok := strings.Cut(filepath.ToSlash(rel), "/")
		if !ok {
			category, name = "", category
		}
		entries[i] = listed{category: category, name: name, ch: ch}
	}
	slices.SortStableFunc(entries, func(a, b listed) int {
		return cmp.Or(
			cmp.Compare(a.category, b.category),
			cmp.Compare(points(a.ch), points(b.ch)),
			cmp.Compare(a.name, b.name),
		)
	})
	return entries
}

func challengeRows(entries []listed) [][]string {
	showPoints := slices.ContainsFunc(entries, func(e listed) bool { return e.ch.Remote != nil })

	var table [][]string
	current := ""
	for _, e := range entries {
		indent := ""
		if e.category != "" {
			indent = "  "
			if e.category != current {
				table = append(table, []string{ui.Bold(e.category)})
				current = e.category
			}
		}
		table = append(table, challengeRow(indent, e.name, e.ch, showPoints))
	}
	return table
}

func challengeRow(indent, name string, ch *challenge.Challenge, showPoints bool) []string {
	mark := ui.Dim("·")
	if ch.IsSolved() {
		mark = ui.Green("✓")
	}
	row := []string{indent + mark + " " + name}
	if showPoints {
		pts := ""
		if ch.Remote != nil && ch.Remote.Points > 0 {
			pts = fmt.Sprintf("%d", ch.Remote.Points)
		}
		row = append(row, ui.Dim(pts))
	}
	if !ch.IsSolved() {
		return row
	}

	when := ui.Dim(ui.Ago(ch.Solved))
	if ch.SolvedBy == challenge.SolvedByTeam {
		when = ui.Accent("team") + " " + when
	}
	return append(row, when, ui.Accent(ansi.Truncate(ch.Flag, flagWidth, "…")))
}

func points(ch *challenge.Challenge) int {
	if ch.Remote == nil {
		return 0
	}
	return ch.Remote.Points
}

func progressLine(challenges []*challenge.Challenge, bar bool) string {
	solved := countSolved(challenges)
	line := fmt.Sprintf("%d/%d solved", solved, len(challenges))
	if bar {
		line = ui.Bar(solved, len(challenges), progressWidth) + "  " + line
	}
	if got, total, ok := pointTotals(challenges); ok {
		line += ui.Dim(" · ") + fmt.Sprintf("%d / %d pts", got, total)
	}
	return line
}

func pointTotals(challenges []*challenge.Challenge) (int, int, bool) {
	got, total, remote := 0, 0, false
	for _, ch := range challenges {
		if ch.Remote == nil {
			continue
		}
		remote = true
		total += ch.Remote.Points
		if ch.IsSolved() {
			got += ch.Remote.Points
		}
	}
	return got, total, remote
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

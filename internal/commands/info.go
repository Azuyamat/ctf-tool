package commands

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/azuyamat/ctf/internal/challenge"
	"github.com/azuyamat/ctf/internal/ui"
	"github.com/azuyamat/gear/command"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
)

const maxTextWidth = 100

var (
	mdBold   = regexp.MustCompile(`\*\*([^*]+)\*\*|__([^_]+)__`)
	mdCode   = regexp.MustCompile("`([^`]+)`")
	mdLink   = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	htmlTags = regexp.MustCompile(`</?[a-zA-Z][^>]*>`)
	htmlBr   = regexp.MustCompile(`(?i)<br\s*/?>|</p>`)
)

func (c *cli) cmdInfo() command.Command {
	return command.NewExecutableCommand("info", "Show the current challenge's details and description").
		Group("Challenges").
		Handler(c.runInfo)
}

func (c *cli) runInfo(ctx *command.Context, args command.ValidatedArgs) error {
	ch, err := c.Challenge(true)
	if err != nil {
		return err
	}

	title := c.shortPath(ch.Path)
	if ch.Remote != nil && ch.Remote.Name != "" {
		title = ch.Remote.Name
	}
	ui.Println(ui.Bold(title))
	ui.Println()
	ev, _ := c.EventFor(ch)
	ui.Println(ui.Table(infoRows(c.shortPath(ch.Path), ch, c.CachedUser(ev), c.Config.Paths(ev).Readme)))

	if ch.Remote != nil {
		if desc := strings.TrimSpace(ch.Remote.Description); desc != "" {
			ui.Println()
			ui.Println(renderMarkdown(desc, textWidth()))
		}
	}
	return nil
}

func infoRows(path string, ch *challenge.Challenge, me, readme string) [][]string {
	var rows [][]string
	add := func(label, value string) {
		if value != "" {
			rows = append(rows, []string{ui.Dim(label), value})
		}
	}

	add("category", ch.Category())
	r := ch.Remote
	if r != nil {
		if r.Points > 0 {
			add("points", fmt.Sprintf("%d", r.Points))
		}
		add("tags", strings.Join(r.Tags, ", "))
	}
	add("connection", ch.Connection)
	if r != nil && (r.MaxAttempts > 0 || ch.Local.Attempts > 0) {
		attempts := fmt.Sprintf("%d", ch.Local.Attempts)
		if r.MaxAttempts > 0 {
			attempts = fmt.Sprintf("%d / %d", ch.Local.Attempts, r.MaxAttempts)
		}
		add("attempts", attempts)
	}
	add("solved", solvedLabel(ch, me))
	if ch.Flag != "" {
		add("flag", ui.Accent(ch.Flag))
	}
	add("files", strings.Join(infoFiles(ch), ", "))
	if r != nil && len(r.Hints) > 0 {
		unlocked := 0
		for _, h := range r.Hints {
			if h.Unlocked {
				unlocked++
			}
		}
		add("hints", fmt.Sprintf("%d of %d unlocked, see %s", unlocked, len(r.Hints), readme))
	}
	if r != nil {
		add("link", r.URL)
	}
	add("path", path)
	if r == nil {
		add("created", ui.Ago(ch.Created))
	}
	return rows
}

func solvedLabel(ch *challenge.Challenge, me string) string {
	if !ch.IsSolved() {
		return ui.Dim("no")
	}
	label := ui.Green("✓") + " " + ui.Ago(ch.Solved)
	switch {
	case ch.SolvedByTeammate(me):
		label += " by " + ch.Solver + ui.Dim(" (team)")
	case !ch.SolvedOnServer() && ch.Remote != nil:
		label += ui.Dim(" (recorded locally, not submitted)")
	}
	return label
}

func infoFiles(ch *challenge.Challenge) []string {
	if len(ch.Local.Downloads) > 0 {
		files := make([]string, 0, len(ch.Local.Downloads))
		for _, rel := range ch.Local.Downloads {
			files = append(files, rel)
		}
		slices.Sort(files)
		return files
	}
	if ch.Remote != nil {
		return ch.Remote.Files
	}
	return nil
}

func textWidth() int {
	width, _, err := term.GetSize(os.Stdout.Fd())
	if err != nil || width <= 0 {
		return maxTextWidth
	}
	return min(width, maxTextWidth)
}

func renderMarkdown(text string, width int) string {
	text = htmlBr.ReplaceAllString(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	text = htmlTags.ReplaceAllString(text, "")

	var out []string
	fenced := false
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			fenced = !fenced
			continue
		}
		if fenced {
			out = append(out, "  "+ui.Accent(line))
			continue
		}
		if heading := strings.TrimLeft(trimmed, "#"); heading != trimmed && strings.HasPrefix(heading, " ") {
			out = append(out, ui.Bold(strings.TrimSpace(heading)))
			continue
		}
		out = append(out, inlineMarkdown(ansi.Wordwrap(line, width, "")))
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n")
}

func inlineMarkdown(s string) string {
	s = mdLink.ReplaceAllStringFunc(s, func(m string) string {
		parts := mdLink.FindStringSubmatch(m)
		if parts[1] == parts[2] {
			return ui.Accent(parts[2])
		}
		return parts[1] + " " + ui.Dim("("+parts[2]+")")
	})
	s = mdCode.ReplaceAllStringFunc(s, func(m string) string {
		return ui.Accent(mdCode.FindStringSubmatch(m)[1])
	})
	return mdBold.ReplaceAllStringFunc(s, func(m string) string {
		parts := mdBold.FindStringSubmatch(m)
		return ui.Bold(parts[1] + parts[2])
	})
}

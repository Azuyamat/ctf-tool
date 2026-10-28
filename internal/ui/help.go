package ui

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/azuyamat/gear/command"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
)

const (
	helpIndent    = "  "
	helpGap       = "   "
	helpMaxWidth  = 100
	helpMinColumn = 24
)

func Help(version string) func(io.Writer, command.Help) {
	return func(w io.Writer, h command.Help) {
		out := colorprofile.NewWriter(w, os.Environ())
		fmt.Fprint(out, renderHelp(h, version, helpWidth()))
	}
}

func helpWidth() int {
	width := 80
	if w, _, err := term.GetSize(os.Stdout.Fd()); err == nil && w > 0 {
		width = w
	} else if n, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && n > 0 {
		width = n
	}
	return min(width, helpMaxWidth)
}

type helpRow struct {
	name  string
	desc  string
	notes []string
}

func renderHelp(h command.Help, version string, width int) string {
	var b strings.Builder
	section := func(title string) {
		b.WriteString("\n" + titleStyle.Render(title) + "\n")
	}

	header := Bold(h.Path)
	if version != "" && !strings.Contains(h.Path, " ") {
		header += " " + Dim(version)
	}
	if h.Summary != "" {
		header += Dim(" — ") + h.Summary
	}
	b.WriteString(header + "\n")
	if h.Long != "" {
		b.WriteString("\n" + wrap(strings.TrimSpace(h.Long), width) + "\n")
	}
	if h.Deprecated != "" {
		b.WriteString("\n" + Red("Deprecated: ") + h.Deprecated + "\n")
	}

	section("Usage")
	for _, u := range h.Usage {
		b.WriteString(helpIndent + Dim("$ ") + usageLine(u) + "\n")
	}

	if len(h.Examples) > 0 {
		section("Examples")
		for _, ex := range h.Examples {
			for _, line := range strings.Split(strings.TrimRight(ex, "\n"), "\n") {
				if strings.HasPrefix(line, "#") {
					b.WriteString(helpIndent + Dim(line) + "\n")
				} else {
					b.WriteString(helpIndent + Dim("$ ") + line + "\n")
				}
			}
		}
	}

	var groups [][]helpRow
	for _, g := range h.CommandGroups {
		var rows []helpRow
		for _, c := range g.Commands {
			name := Bold(c.Name)
			if len(c.Aliases) > 0 {
				name += Dim(", " + strings.Join(c.Aliases, ", "))
			}
			rows = append(rows, helpRow{name: name, desc: c.Description})
		}
		groups = append(groups, rows)
	}
	column := 0
	for _, rows := range groups {
		column = max(column, nameWidth(rows))
	}
	for i, g := range h.CommandGroups {
		section(g.Title)
		writeHelpRows(&b, groups[i], column, width)
	}

	if len(h.Args) > 0 {
		var rows []helpRow
		for _, a := range h.Args {
			name := a.Name
			if a.Variadic {
				name += "..."
			}
			row := helpRow{name: Accent(name), desc: a.Description}
			if len(a.Choices) > 0 {
				row.notes = append(row.notes, "one of "+strings.Join(a.Choices, ", "))
			}
			rows = append(rows, row)
		}
		section("Arguments")
		writeHelpRows(&b, rows, nameWidth(rows), width)
	}

	for _, s := range []struct {
		title string
		flags []command.HelpFlag
	}{{"Flags", h.Flags}, {"Global flags", h.GlobalFlags}} {
		if len(s.flags) == 0 {
			continue
		}
		rows := flagHelpRows(s.flags)
		section(s.title)
		writeHelpRows(&b, rows, nameWidth(rows), width)
	}

	if h.Footer != "" {
		b.WriteString("\n" + Dim(wrap(h.Footer, width)) + "\n")
	}
	return b.String()
}

func flagHelpRows(flags []command.HelpFlag) []helpRow {
	var rows []helpRow
	for _, f := range flags {
		name := "    "
		if f.Shorthand != "" {
			name = Bold("-"+f.Shorthand) + Dim(", ")
		}
		if f.Negatable {
			name += Bold("--") + Dim("[no-]") + Bold(f.Name)
		} else {
			name += Bold("--" + f.Name)
		}
		if f.Placeholder != "" {
			name += " " + Dim(f.Placeholder)
		}

		var notes []string
		if f.Default != "" {
			notes = append(notes, "default "+f.Default)
		}
		if f.Env != "" {
			notes = append(notes, "$"+f.Env)
		}
		if f.Repeatable {
			notes = append(notes, "repeatable")
		}
		desc := f.Description
		if f.Required {
			desc += " " + Red("(required)")
		}
		rows = append(rows, helpRow{name: name, desc: desc, notes: notes})
	}
	return rows
}

func nameWidth(rows []helpRow) int {
	w := 0
	for _, r := range rows {
		w = max(w, lipgloss.Width(r.name))
	}
	return w
}

func writeHelpRows(b *strings.Builder, rows []helpRow, column, width int) {
	descWidth := width - len(helpIndent) - column - len(helpGap)
	if descWidth < helpMinColumn {
		descWidth = 0
	}
	pad := strings.Repeat(" ", len(helpIndent)+column+len(helpGap))
	for _, r := range rows {
		lines := strings.Split(wrap(r.desc, descWidth), "\n")
		if len(r.notes) > 0 {
			lines = append(lines, strings.Split(Dim(wrap(strings.Join(r.notes, " · "), descWidth)), "\n")...)
		}
		gap := strings.Repeat(" ", column-lipgloss.Width(r.name)) + helpGap
		b.WriteString(helpIndent + r.name + gap + lines[0] + "\n")
		for _, l := range lines[1:] {
			b.WriteString(pad + l + "\n")
		}
	}
}

func wrap(text string, width int) string {
	if width <= 0 {
		return text
	}
	return ansi.Wordwrap(text, width, "")
}

func usageLine(u string) string {
	words := strings.Fields(u)
	for i, w := range words {
		switch {
		case w == "[flags]":
			words[i] = Dim(w)
		case strings.HasPrefix(w, "<"):
			words[i] = Accent(w)
		case strings.HasPrefix(w, "["):
			words[i] = Dim("[") + Accent(strings.Trim(w, "[]")) + Dim("]")
		default:
			words[i] = Bold(w)
		}
	}
	return strings.Join(words, " ")
}

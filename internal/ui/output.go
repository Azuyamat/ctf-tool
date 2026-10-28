package ui

import (
	"fmt"
	"os"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/tree"
	"github.com/charmbracelet/colorprofile"
)

var (
	stdout = colorprofile.NewWriter(os.Stdout, os.Environ())
	stderr = colorprofile.NewWriter(os.Stderr, os.Environ())

	boldStyle   = lipgloss.NewStyle().Bold(true)
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack)
	accentStyle = lipgloss.NewStyle().Foreground(lipgloss.Cyan)
	titleStyle  = accentStyle.Bold(true)
	greenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Green)
	redStyle    = lipgloss.NewStyle().Foreground(lipgloss.Red)
)

func Bold(s string) string   { return boldStyle.Render(s) }
func Dim(s string) string    { return dimStyle.Render(s) }
func Accent(s string) string { return accentStyle.Render(s) }
func Green(s string) string  { return greenStyle.Render(s) }
func Red(s string) string    { return redStyle.Render(s) }

func Println(a ...any) {
	fmt.Fprintln(stdout, a...)
}

func Successf(format string, a ...any) {
	fmt.Fprintln(stdout, Green("✓"), fmt.Sprintf(format, a...))
}

func Promptf(format string, a ...any) {
	fmt.Fprintf(stderr, format, a...)
}

func Notef(format string, a ...any) {
	fmt.Fprintln(stderr, Dim(fmt.Sprintf(format, a...)))
}

func Error(err error) {
	fmt.Fprintln(stderr, redStyle.Render("✗"), err)
}

func Tree(root string) *tree.Tree {
	return tree.Root(root).
		Enumerator(tree.RoundedEnumerator).
		EnumeratorStyle(dimStyle.PaddingRight(1))
}

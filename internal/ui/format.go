package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

const columnGap = "  "

func Table(rows [][]string) string {
	var widths []int
	for _, row := range rows {
		for i, cell := range row {
			if i == len(widths) {
				widths = append(widths, 0)
			}
			widths[i] = max(widths[i], lipgloss.Width(cell))
		}
	}

	lines := make([]string, len(rows))
	for i, row := range rows {
		var b strings.Builder
		for j, cell := range row {
			if j > 0 {
				b.WriteString(columnGap)
			}
			b.WriteString(cell)
			b.WriteString(strings.Repeat(" ", widths[j]-lipgloss.Width(cell)))
		}
		lines[i] = strings.TrimRight(b.String(), " ")
	}
	return strings.Join(lines, "\n")
}

func Bar(done, total, width int) string {
	filled := 0
	if total > 0 {
		filled = done * width / total
	}
	return Green(strings.Repeat("█", filled)) + Dim(strings.Repeat("░", width-filled))
}

func Ago(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func Size(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

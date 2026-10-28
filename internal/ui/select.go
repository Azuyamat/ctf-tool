package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const selectChromeLines = 6

type Choice interface {
	Label() string
	Detail() string
}

type SelectModel struct {
	title     string
	choices   []Choice
	selected  map[int]bool
	visible   []int
	cursor    int
	offset    int
	width     int
	height    int
	filter    string
	filtering bool
	confirmed bool
	done      bool
}

func NewSelect(title string, choices []Choice) SelectModel {
	m := SelectModel{
		title:    title,
		choices:  choices,
		selected: make(map[int]bool),
	}
	m.setFilter("")
	return m
}

func (m SelectModel) SelectFirst(n int) SelectModel {
	for i := range min(n, len(m.choices)) {
		m.selected[i] = true
	}
	return m
}

func (m SelectModel) Init() tea.Cmd {
	return nil
}

func (m SelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.scroll()
	case tea.KeyPressMsg:
		return m, m.handleKey(msg)
	}
	return m, nil
}

func (m *SelectModel) handleKey(msg tea.KeyPressMsg) tea.Cmd {
	key := msg.String()
	switch key {
	case "ctrl+c":
		return m.quit(false)
	case "up":
		m.move(-1)
		return nil
	case "down":
		m.move(1)
		return nil
	}

	if m.filtering {
		switch key {
		case "enter":
			m.filtering = false
		case "esc":
			m.filtering = false
			m.setFilter("")
		case "backspace":
			_, size := utf8.DecodeLastRuneInString(m.filter)
			m.setFilter(m.filter[:len(m.filter)-size])
		default:
			if msg.Text != "" {
				m.setFilter(m.filter + msg.Text)
			}
		}
		return nil
	}

	switch key {
	case "k":
		m.move(-1)
	case "j":
		m.move(1)
	case "space":
		m.toggle()
	case "/":
		m.filtering = true
	case "esc":
		if m.filter != "" {
			m.setFilter("")
			return nil
		}
		return m.quit(false)
	case "q":
		return m.quit(false)
	case "enter":
		return m.quit(true)
	}
	return nil
}

func (m *SelectModel) quit(confirmed bool) tea.Cmd {
	m.confirmed = confirmed
	m.done = true
	return tea.Quit
}

func (m *SelectModel) move(delta int) {
	n := len(m.visible)
	if n == 0 {
		return
	}
	m.cursor = (m.cursor + delta + n) % n
	m.scroll()
}

func (m *SelectModel) toggle() {
	if len(m.visible) == 0 {
		return
	}
	i := m.visible[m.cursor]
	if m.selected[i] {
		delete(m.selected, i)
	} else {
		m.selected[i] = true
	}
}

func (m *SelectModel) setFilter(filter string) {
	needle := strings.ToLower(filter)
	var visible []int
	for i, choice := range m.choices {
		if strings.Contains(strings.ToLower(choice.Label()), needle) {
			visible = append(visible, i)
		}
	}
	m.filter, m.visible = filter, visible
	m.cursor, m.offset = 0, 0
}

func (m SelectModel) rows() int {
	if m.height == 0 {
		return len(m.visible)
	}
	return max(1, m.height-selectChromeLines)
}

func (m *SelectModel) scroll() {
	rows := m.rows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	} else if m.cursor >= m.offset+rows {
		m.offset = m.cursor - rows + 1
	}
}

func (m SelectModel) View() tea.View {
	if m.done {
		return tea.NewView("")
	}

	lines := []string{titleStyle.Render(m.title), m.filterLine(), ""}

	end := min(m.offset+m.rows(), len(m.visible))
	window := m.visible[m.offset:end]
	labelWidth := 0
	for _, i := range window {
		labelWidth = max(labelWidth, lipgloss.Width(m.choices[i].Label()))
	}
	for row, i := range window {
		lines = append(lines, m.choiceLine(i, m.offset+row == m.cursor, labelWidth))
	}
	if len(m.visible) == 0 {
		lines = append(lines, Dim("  no matches"))
	}

	position := ""
	if len(window) < len(m.visible) {
		position = Dim(fmt.Sprintf("  %d–%d of %d", m.offset+1, end, len(m.visible)))
	}
	lines = append(lines, position, "", m.helpLine())

	return tea.NewView(strings.Join(lines, "\n"))
}

func (m SelectModel) filterLine() string {
	switch {
	case m.filtering:
		return Dim("/ ") + m.filter + Accent("▏")
	case m.filter != "":
		return Dim("/ " + m.filter)
	default:
		return ""
	}
}

func (m SelectModel) choiceLine(i int, current bool, labelWidth int) string {
	choice := m.choices[i]

	pointer := " "
	label := choice.Label()
	if current {
		pointer = Accent("›")
		label = Bold(label)
	}

	box := Dim("[ ]")
	if m.selected[i] {
		box = Green("[x]")
	}

	padding := strings.Repeat(" ", labelWidth-lipgloss.Width(choice.Label()))
	line := fmt.Sprintf("%s %s %s%s  %s", pointer, box, label, padding, Dim(choice.Detail()))
	if m.width > 0 {
		line = ansi.Truncate(line, m.width, "…")
	}
	return line
}

func (m SelectModel) helpLine() string {
	if m.filtering {
		return Dim("type to filter · enter done · esc clear")
	}
	return Dim("space select · / filter · enter confirm · q quit")
}

func (m SelectModel) Selected() []Choice {
	if !m.confirmed {
		return nil
	}

	var selected []Choice
	for i, choice := range m.choices {
		if m.selected[i] {
			selected = append(selected, choice)
		}
	}
	if len(selected) == 0 && len(m.visible) > 0 {
		selected = append(selected, m.choices[m.visible[m.cursor]])
	}
	return selected
}

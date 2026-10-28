package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

type Choice interface {
	String() string
}

type SelectModel struct {
	choices   []Choice
	selected  map[int]struct{}
	cursor    int
	confirmed bool
}

func NewSelect(
	choices []Choice,
) SelectModel {
	return SelectModel{
		choices:  choices,
		selected: make(map[int]struct{}),
		cursor:   0,
	}
}

func (m SelectModel) SelectFirst(n int) SelectModel {
	for i := range min(n, len(m.choices)) {
		m.selected[i] = struct{}{}
	}
	return m
}

func (m SelectModel) Init() tea.Cmd {
	return nil
}

func (m SelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			m.cursor = (m.cursor + 1) % len(m.choices)
		case "space":
			if _, ok := m.selected[m.cursor]; ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		case "enter":
			m.confirmed = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m SelectModel) View() tea.View {
	var s strings.Builder
	s.WriteString("Select files to grab:\n\n")
	for i, choice := range m.choices {
		cursor := " " // no cursor
		if m.cursor == i {
			cursor = ">" // cursor!
		}

		checked := " " // not selected
		if _, ok := m.selected[i]; ok {
			checked = "x" // selected!
		}

		s.WriteString(fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice))
	}

	s.WriteString("\nspace to select, enter to confirm, q to quit\n")

	return tea.NewView(s.String())
}

func (m SelectModel) Selected() []Choice {
	if !m.confirmed {
		return nil
	}
	if len(m.selected) == 0 {
		return []Choice{m.choices[m.cursor]}
	}
	var selected []Choice
	for i, choice := range m.choices {
		if _, ok := m.selected[i]; ok {
			selected = append(selected, choice)
		}
	}
	return selected
}

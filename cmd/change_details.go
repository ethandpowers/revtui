package main

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type changeDetailsModel struct {
	change Change
}

func (m changeDetailsModel) Init() tea.Cmd { return nil }

func (m changeDetailsModel) Update(msg tea.Msg) (changeDetailsModel, tea.Cmd) {
	return m, nil
}

func (m changeDetailsModel) View(width int, height int) string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2)

	if width > 0 {
		boxStyle = boxStyle.
			Width(max(0, width-2)).
			Height(max(0, height-2))
	}

	s := boxStyle.Render("")
	return s
}

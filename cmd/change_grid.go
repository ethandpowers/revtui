package main

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type changeGridModel struct {
	backend Backend
	columns []changeGridColModel

	xCursor int
	yCursor int
}

func (m changeGridModel) getRenderedCols() []changeGridColModel {
	renderedCols := make([]changeGridColModel, 0, 5)

	for _, col := range m.columns {
		if col.status != ReviewStatusUnknown || len(col.changes) > 0 {
			renderedCols = append(renderedCols, col)
		}
	}

	return renderedCols
}

func (m changeGridModel) Init() tea.Cmd {
	return nil
}

func (m changeGridModel) Update(msg tea.Msg) (changeGridModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		cols := m.getRenderedCols()
		colCount := len(cols)

		switch msg.String() {
		case "left", "h":
			m.xCursor = max(m.xCursor-1, 0)
		case "down", "j":
			m.yCursor++
		case "up", "k":
			m.yCursor = max(m.yCursor-1, 0)
		case "right", "l":
			m.xCursor = min(m.xCursor+1, colCount-1)
		}

		changeCount := len(cols[m.xCursor].changes)
		if m.yCursor > changeCount-1 {
			m.yCursor = changeCount - 1
		}
	}
	return m, nil
}

func (m changeGridModel) View(width int, height int) string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Width(width).
		Height(height - 1)

	const spacing = "  " // 1 character of space just wasn't doing it for me
	colCount := len(m.getRenderedCols())
	colWidth := (width-6)/colCount - len(spacing)
	var renderedCols []string

	for i, col := range m.getRenderedCols() {
		var cursor *int
		if i == m.xCursor {
			cursor = &m.yCursor
		}
		renderedCols = append(renderedCols, col.View(colWidth, height-5, cursor))
		if i < colCount-1 {
			renderedCols = append(renderedCols, spacing)
		}
	}

	s := lipgloss.JoinHorizontal(lipgloss.Top, renderedCols...)

	return boxStyle.Render(s)
}

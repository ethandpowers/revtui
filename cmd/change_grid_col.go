package main

import (
	"fmt"
	"image/color"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type changeGridColModel struct {
	status       ReviewStatus
	changes      []Change
	scrollOffset int
}

func (m changeGridColModel) Init() tea.Cmd {
	return nil
}

func (m changeGridColModel) Update(msg tea.Msg) (changeGridColModel, tea.Cmd) {
	return m, nil
}

func (m changeGridColModel) View(width int, visibleChangeCount int, cursor *int) string {
	var s strings.Builder
	s.WriteString(reviewStatusString(m.status, true))
	s.WriteString("\n")

	for range width {
		s.WriteString("─")
	}

	header := getReviewStatusStyle(m.status).Render(s.String())

	s.Reset()

	for i, change := range m.changes {
		lastVisibleChangeIndex := min(len(m.changes), m.scrollOffset+visibleChangeCount-1)
		if i < m.scrollOffset || i > lastVisibleChangeIndex {
			continue
		}

		var bgColor color.Color = nil
		isActiveCell := cursor != nil && *cursor == i-m.scrollOffset
		if isActiveCell {
			bgColor = lipgloss.Blue
		}

		s.WriteString("\n")
		s.WriteString(
			lipgloss.NewStyle().
				Background(bgColor).
				Render(fmt.Sprintf("%-*.*s", width, width, change.Title)),
		)
		s.WriteString("\n")

		metaText := userDisplayName(&change.Author)
		if change.Branch != "" {
			metaText += " · " + change.Branch
		}
		s.WriteString(
			lipgloss.NewStyle().
				Foreground(lipgloss.BrightBlack).
				Background(bgColor).
				Render(fmt.Sprintf("%-*.*s", width, width, metaText)),
		)
		s.WriteString("\n")
		s.WriteString(
			lipgloss.NewStyle().
				Foreground(lipgloss.BrightBlack).
				Background(bgColor).
				Render(fmt.Sprintf("%-*.*s", width, width, change.Project)),
		)

		if i < lastVisibleChangeIndex {
			s.WriteString("\n")
		}
	}

	return header + s.String()
}

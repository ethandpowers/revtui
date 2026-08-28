package main

import (
	"fmt"
	"image/color"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type changeGridColModel struct {
	status  ReviewStatus
	changes []Change
}

func (m changeGridColModel) Init() tea.Cmd {
	return nil
}

func (m changeGridColModel) Update(msg tea.Msg) (changeGridColModel, tea.Cmd) {
	return m, nil
}

func (m changeGridColModel) View(width int, height int, cursor *int) string {
	var s strings.Builder
	s.WriteString(reviewStatusString(m.status, true))
	s.WriteString("\n")

	for range width {
		s.WriteString("─")
	}

	header := getReviewStatusStyle(m.status).Render(s.String())

	s.Reset()

	visibleChangeCount := (height - 2) / 4
	if visibleChangeCount*4+3 <= height-2 {
		visibleChangeCount++
	}

	changeStart := 0
	if cursor != nil {
		changeStart = *cursor - visibleChangeCount + 1
	}

	if changeStart < 0 {
		changeStart = 0
	}

	for i, change := range m.changes {
		lastVisibleChangeIndex := min(len(m.changes), changeStart+visibleChangeCount-1)
		if i < changeStart || i > lastVisibleChangeIndex {
			continue
		}

		var bgColor color.Color = nil
		isActiveCell := cursor != nil && *cursor == i
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

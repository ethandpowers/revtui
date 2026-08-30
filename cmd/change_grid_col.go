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
	headerStyle := getReviewStatusStyle(m.status)
	changeCountStyle := lipgloss.NewStyle().Foreground(lipgloss.BrightBlack)

	s.WriteString(headerStyle.Render(reviewStatusString(m.status, true)))
	s.WriteString(changeCountStyle.Render(fmt.Sprintf(" %d", len(m.changes))))
	s.WriteString("\n")

	var separator strings.Builder
	for range width {
		separator.WriteString("─")
	}

	s.WriteString(headerStyle.Render(separator.String()))

	header := s.String()

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

		textWidth := width
		flagCells := ""
		if change.Flags.HasConflicts {
			flagCells += lipgloss.NewStyle().Foreground(lipgloss.Red).Render("│")
			textWidth--
		}
		if change.Flags.IsWorkInProgress {
			flagCells += lipgloss.NewStyle().Foreground(lipgloss.Color("130")).Render("│")
			textWidth--
		}

		s.WriteString("\n")
		s.WriteString(flagCells)
		s.WriteString(
			lipgloss.NewStyle().
				Background(bgColor).
				Render(fmt.Sprintf("%-*.*s", textWidth, textWidth, change.Title)),
		)
		s.WriteString("\n")

		s.WriteString(flagCells)
		metaText := userDisplayName(&change.Author)
		if change.Branch != "" {
			metaText += " · " + change.Branch
		}
		s.WriteString(
			lipgloss.NewStyle().
				Foreground(lipgloss.BrightBlack).
				Background(bgColor).
				Render(fmt.Sprintf("%-*.*s", textWidth, textWidth, metaText)),
		)
		s.WriteString("\n")
		s.WriteString(flagCells)
		s.WriteString(
			lipgloss.NewStyle().
				Foreground(lipgloss.BrightBlack).
				Background(bgColor).
				Render(fmt.Sprintf("%-*.*s", textWidth, textWidth, change.Project)),
		)

		if i < lastVisibleChangeIndex {
			s.WriteString("\n")
		}
	}

	return header + s.String()
}

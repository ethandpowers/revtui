package main

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type columnConfig struct {
	longestFlags    int
	longestChangeID int
	longestReview   int
	longestSubject  int
	longestOwner    int
}

type changeListModel struct {
	backend Backend
	changes []Change

	cursor       int
	columnConfig columnConfig
}

func (m changeListModel) renderChangeRow(i int) string {
	change := m.changes[i]

	reviewStatusStyle := getReviewStatusStyle(change.Review.Primary)
	rowStyle := lipgloss.NewStyle()
	cursor := " "

	if m.cursor == i {
		cursor = ">"
		bgColor := lipgloss.Blue
		rowStyle = rowStyle.Background(bgColor)
		reviewStatusStyle = reviewStatusStyle.Background(bgColor)
	}

	cells := []string{
		rowStyle.Render(cursor + " "),
		renderFlagsCell(change.Flags, m.columnConfig.longestFlags+1, rowStyle),
		rowStyle.Width(m.columnConfig.longestChangeID + 1).Render(change.ChangeID),
		reviewStatusStyle.Width(m.columnConfig.longestReview + 1).Render(reviewStatusString(change.Review.Primary, false)),
		rowStyle.Width(m.columnConfig.longestSubject + 1).Render(change.Title),
		rowStyle.Width(m.columnConfig.longestOwner).Render(userDisplayName(&change.Author)),
	}

	return strings.Join(cells, "")
}

func (m changeListModel) Init() tea.Cmd {
	return nil
}

func (m changeListModel) Update(msg tea.Msg) (changeListModel, tea.Cmd) {
	switch msg := msg.(type) {

	case changesLoadedMsg:

		longestFlags := len(FlagsField)
		longestChangeID := len(ChangeIDField)
		longestReview := len(ReviewField)
		longestSubject := len(SubjectField)
		longestOwner := len(OwnerField)

		for _, change := range msg.changes {
			longestFlags = max(longestFlags, len(changeFlagsString(change.Flags)))
			longestChangeID = max(longestChangeID, len(change.ChangeID))
			longestReview = max(longestReview, len(reviewStatusString(change.Review.Primary, false)))
			longestSubject = max(longestSubject, len(change.Title))
			longestOwner = max(longestOwner, len(userDisplayName(&change.Author)))
		}

		m.columnConfig = columnConfig{
			longestFlags,
			longestChangeID,
			longestReview,
			longestSubject,
			longestOwner,
		}
		return m, nil

	case checkoutMsg:
		return m, stopLoading(msg.message, msg.err)

	case tea.KeyPressMsg:

		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.changes)-1 {
				m.cursor++
			}

		case "c":
			if len(m.changes) == 0 {
				return m, nil
			}

			return m, tea.Batch(startLoading(""), checkoutChangeCmd(m.changes[m.cursor], m.backend))

		case "enter":
			if len(m.changes) == 0 {
				return m, nil
			}

			return m, showDetails(m.changes[m.cursor])
		}
	}

	return m, nil
}

func (m changeListModel) View(width int, height int) string {
	s := ""
	rows := []string{
		fmt.Sprintf(
			"  %-*s %-*s %-*s %-*s %-*s",
			m.columnConfig.longestFlags,
			FlagsField,
			m.columnConfig.longestChangeID,
			ChangeIDField,
			m.columnConfig.longestReview,
			ReviewField,
			m.columnConfig.longestSubject,
			SubjectField,
			m.columnConfig.longestOwner,
			OwnerField,
		),
	}

	mainViewportSize := max(height-6, 0)

	scrollOffset := 0
	if m.cursor >= mainViewportSize && mainViewportSize > 0 {
		scrollOffset = m.cursor - mainViewportSize + 1
	}

	for i := range mainViewportSize {
		changeIndex := i + scrollOffset
		if len(m.changes) > changeIndex {
			rows = append(rows, m.renderChangeRow(changeIndex))
		} else {
			rows = append(rows, "")
		}
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2)

	if width > 0 {
		boxStyle = boxStyle.Width(width)
	}

	s = boxStyle.Render(strings.Join(rows, "\n"))

	return s
}

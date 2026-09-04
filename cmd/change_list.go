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
	scrollOffset int
	columnConfig columnConfig

	width  int
	height int
}

func (m changeListModel) SelectedChange() *Change {
	if len(m.changes) == 0 {
		return nil
	}

	return &m.changes[m.cursor+m.scrollOffset]
}

func (m changeListModel) getVisibleChangeCount() int {
	return max(0, m.height-5)
}

func (m changeListModel) renderChangeRow(i int) string {
	change := m.changes[i]

	reviewStatusStyle := getReviewStatusStyle(change.Review.Primary)
	rowStyle := lipgloss.NewStyle()
	cursor := " "

	if m.cursor+m.scrollOffset == i {
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

	case tea.KeyPressMsg:

		switch msg.String() {
		case "down", "j":
			rowsVisible := m.getVisibleChangeCount()
			numChanges := len(m.changes)
			if m.cursor < rowsVisible-1 && m.cursor < numChanges-1 {
				m.cursor++
			} else if m.cursor+m.scrollOffset < numChanges-1 {
				m.scrollOffset++
			}
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			} else if m.scrollOffset > 0 {
				m.scrollOffset--
			}
		}
	}

	return m, nil
}

func (m changeListModel) View() string {
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

	visibleChanges := m.getVisibleChangeCount()

	for i := range visibleChanges {
		changeIndex := i + m.scrollOffset
		if len(m.changes) > changeIndex {
			rows = append(rows, m.renderChangeRow(changeIndex))
		} else {
			rows = append(rows, "")
		}
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2)

	if m.width > 0 {
		boxStyle = boxStyle.Width(m.width)
	}

	s = boxStyle.Render(strings.Join(rows, "\n"))

	return s
}

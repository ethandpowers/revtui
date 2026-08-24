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

func reviewStatusString(status ReviewStatus) string {
	switch status {
	case ReviewStatusNotReady:
		return "not ready"
	case ReviewStatusReadyForReview:
		return "ready for review"
	case ReviewStatusReviewed:
		return "reviewed"
	case ReviewStatusVerified:
		return "verified"
	case ReviewStatusBlocked:
		return "blocked"
	case ReviewStatusUnknown:
		return "unknown"
	}

	return "unknown"
}

func changeFlagsString(flags ChangeFlags) string {
	s := ""

	if flags.HasConflicts {
		s += "!"
	}

	if flags.IsWorkInProgress {
		s += "W"
	}

	return s
}

func flagStyle(flag rune, rowStyle lipgloss.Style) lipgloss.Style {
	s := lipgloss.NewStyle()

	switch flag {
	case '!':
		s = s.Foreground(lipgloss.Red).Bold(true)
	case 'W':
		s = s.Foreground(lipgloss.Color("130")).Bold(true)
	}

	if background := rowStyle.GetBackground(); background != nil {
		s = s.Background(background)
	}

	return s
}

func renderFlagsCell(flags ChangeFlags, width int, rowStyle lipgloss.Style) string {
	plainFlags := changeFlagsString(flags)
	parts := make([]string, 0, len(plainFlags)+1)

	for _, flag := range plainFlags {
		parts = append(parts, flagStyle(flag, rowStyle).Render(string(flag)))
	}

	paddingWidth := max(0, width-lipgloss.Width(plainFlags))
	if paddingWidth > 0 {
		parts = append(parts, rowStyle.Width(paddingWidth).Render(""))
	}

	return strings.Join(parts, "")
}

type changeListModel struct {
	backend Backend
	changes []Change

	cursor       int
	columnConfig columnConfig
}

type changesLoadedMsg struct {
	changes []Change
	err     error
}

func loadChangesCmd(backend Backend) tea.Cmd {
	return func() tea.Msg {
		changes, err := backend.GetChanges()
		return changesLoadedMsg{
			changes: changes,
			err:     err,
		}
	}
}

type checkoutMsg struct {
	message string
	err     error
}

func checkoutChangeCmd(change Change, backend Backend) tea.Cmd {
	return func() tea.Msg {
		err := backend.Checkout(change)
		return checkoutMsg{
			"Done",
			err,
		}
	}
}

func (m changeListModel) getReviewStatusStyle(status ReviewStatus) lipgloss.Style {
	s := lipgloss.NewStyle()

	switch status {
	case ReviewStatusVerified:
		s = s.Foreground(lipgloss.BrightGreen).Bold(true)
	case ReviewStatusReviewed:
		s = s.Foreground(lipgloss.Green)
	case ReviewStatusReadyForReview:
		s = s.Foreground(lipgloss.Cyan)
	case ReviewStatusBlocked:
		s = s.Foreground(lipgloss.Yellow).Bold(true)
	case ReviewStatusNotReady:
		s = s.Foreground(lipgloss.BrightBlack)
	case ReviewStatusUnknown:
		s = s.Foreground(lipgloss.BrightBlack).Italic(true)
	}

	return s
}

func (m changeListModel) renderChangeRow(i int) string {
	change := m.changes[i]

	reviewStatusStyle := m.getReviewStatusStyle(change.Review.Primary)
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
		reviewStatusStyle.Width(m.columnConfig.longestReview + 1).Render(reviewStatusString(change.Review.Primary)),
		rowStyle.Width(m.columnConfig.longestSubject + 1).Render(change.Title),
		rowStyle.Width(m.columnConfig.longestOwner).Render(userDisplayName(&change.Author)),
	}

	return strings.Join(cells, "")
}

func (m changeListModel) Init() tea.Cmd {
	return loadChangesCmd(m.backend)
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
			longestReview = max(longestReview, len(reviewStatusString(change.Review.Primary)))
			longestSubject = max(longestSubject, len(change.Title))
			longestOwner = max(longestOwner, len(userDisplayName(&change.Author)))
		}

		m.changes = msg.changes
		m.columnConfig = columnConfig{
			longestFlags,
			longestChangeID,
			longestReview,
			longestSubject,
			longestOwner,
		}
		return m, stopLoading("", msg.err)

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
		boxStyle = boxStyle.Width(max(0, width-2))
	}

	s = boxStyle.Render(strings.Join(rows, "\n"))

	return s
}

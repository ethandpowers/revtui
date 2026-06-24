package main

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type columnConfig struct {
	longestChangeID int
	longestSubject  int
	longestOwner    int
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

func (m changeListModel) renderChangeRow(i int) string {
	change := m.changes[i]
	style := lipgloss.NewStyle()
	cursor := " "

	if m.cursor == i {
		cursor = ">"
		style = style.Background(lipgloss.Blue)
	}

	return style.Render(fmt.Sprintf(
		"%s %-*s %-*s %-*s",
		cursor,
		m.columnConfig.longestChangeID,
		change.ChangeID,
		m.columnConfig.longestSubject,
		change.Title,
		m.columnConfig.longestOwner,
		userDisplayName(&change.Author),
	))
}

func (m changeListModel) Init() tea.Cmd {
	return loadChangesCmd(m.backend)
}

func (m changeListModel) Update(msg tea.Msg) (changeListModel, tea.Cmd) {
	switch msg := msg.(type) {

	case changesLoadedMsg:

		longestChangeID := len(ChangeIDField)
		longestSubject := len(SubjectField)
		longestOwner := len(OwnerField)

		for _, change := range msg.changes {
			longestChangeID = max(longestChangeID, len(change.ChangeID))
			longestSubject = max(longestSubject, len(change.Title))
			longestOwner = max(longestOwner, len(userDisplayName(&change.Author)))
		}

		m.changes = msg.changes
		m.columnConfig = columnConfig{
			longestChangeID,
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
			"  %-*s %-*s %-*s",
			m.columnConfig.longestChangeID,
			ChangeIDField,
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

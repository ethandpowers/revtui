package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func reviewStatusString(status ReviewStatus, upper bool) string {
	caser := cases.Title(language.English)

	var s string

	switch status {
	case ReviewStatusNotReady:
		s = "not ready"
	case ReviewStatusReadyForReview:
		s = "ready for review"
	case ReviewStatusReviewed:
		s = "reviewed"
	case ReviewStatusVerified:
		s = "verified"
	case ReviewStatusBlocked:
		s = "blocked"
	case ReviewStatusUnknown:
		s = "unknown"
	}

	if upper {
		s = caser.String(s)
	}

	return s
}

func getReviewStatusStyle(status ReviewStatus) lipgloss.Style {
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

	background := rowStyle.GetBackground()
	s = s.Background(background)

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

func leftTruncate(s string, width int) string {
	if width <= 0 {
		return ""
	}

	runes := []rune(s)
	if len(runes) <= width {
		return s
	}

	return string(runes[len(runes)-width:])
}

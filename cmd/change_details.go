package main

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ethandpowers/revtui/internal/patch"
)

type changeDetailsModel struct {
	backend        Backend
	change         Change
	patch          *patch.Patch
	prettyDetails  string
	patchLineCount int
	err            error
	cursor         int
	width          int
	height         int
	lastTime       time.Time
	lastKey        string
}

func (m *changeDetailsModel) maxCursor() int {
	return max(0, m.patchLineCount-m.height+3)
}

type patchLoadedMsg struct {
	patch string
	err   error
}

func fetchPatchCmd(backend Backend, change Change) tea.Cmd {
	return func() tea.Msg {
		patch, err := backend.GetPatch(change)
		return patchLoadedMsg{patch, err}
	}
}

func (m changeDetailsModel) Init() tea.Cmd {
	return fetchPatchCmd(m.backend, m.change)
}

func (m changeDetailsModel) Update(msg tea.Msg) (changeDetailsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		key := msg.String()

		switch key {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < m.maxCursor() {
				m.cursor++
			}

		case "pgup":
			m.cursor -= m.height - 4
			if m.cursor < 0 {
				m.cursor = 0
			}

		case "pgdown":
			m.cursor += m.height - 4
			if m.cursor > m.maxCursor() {
				m.cursor = m.maxCursor()
			}

		case "G":
			m.cursor = m.maxCursor()

		case "g":
			if m.lastKey == "g" && time.Since(m.lastTime) < 300*time.Millisecond {
				m.cursor = 0
				m.lastKey = ""
			} else {
				m.lastKey = msg.String()
				m.lastTime = time.Now()
			}
		}

	case patchLoadedMsg:
		parsedPatch, err := patch.ParsePatch(msg.patch)
		m.patch = &parsedPatch
		m.err = err
		if err != nil {
			break
		}

		m.prettyDetails = buildPrettyDetails(m.change, parsedPatch)

		m.patchLineCount = len(strings.Split(m.prettyDetails, "\n"))
		m.cursor = 0

		return m, nil
	}
	return m, nil
}

func (m changeDetailsModel) View() string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Width(m.width).
		Height(m.height)

	content := ""
	if m.err != nil {
		content = fmt.Sprintf("Error: %s", m.err.Error())
	} else if len(m.prettyDetails) > 0 {

		lines := strings.Split(m.prettyDetails, "\n")
		start := max(0, min(m.cursor, len(lines)-1))
		end := min(start+m.height-4, len(lines))
		content += strings.Join(lines[start:end], "\n")
	}

	return boxStyle.Render(content)
}

func buildPrettyDetails(change Change, p patch.Patch) string {
	var s strings.Builder

	hashStyle := lipgloss.
		NewStyle()

	s.WriteString(hashStyle.Render(" " + p.Metadata.Hash))
	s.WriteString("\n")

	s.WriteString(fmt.Sprintf(" %s  %s", change.Project, change.Branch))
	s.WriteString("\n")

	authorStyle := lipgloss.
		NewStyle().
		Foreground(lipgloss.BrightMagenta)

	s.WriteString(authorStyle.Render(fmt.Sprintf(" %s", userDisplayName(&change.Author))))
	s.WriteString("\n")

	statusStyle := getReviewStatusStyle(change.Review.Primary)

	s.WriteString(statusStyle.Render(reviewStatusString(change.Review.Primary, false)))
	s.WriteString("\n\n")

	titleStyle := lipgloss.
		NewStyle().
		Bold(true)
	s.WriteString(titleStyle.Render(change.Title))
	s.WriteString("\n\n")

	s.WriteString(p.Metadata.Body)
	s.WriteString("\n\n")

	dividerStyle := lipgloss.NewStyle().Foreground(lipgloss.BrightBlack)
	divider := dividerStyle.Render(strings.Repeat("─", 72))
	s.WriteString(divider)
	s.WriteString("\n")

	oldLineStyle := lipgloss.
		NewStyle().
		Foreground(lipgloss.BrightRed)
	newLineStyle := lipgloss.
		NewStyle().
		Foreground(lipgloss.BrightGreen)

	hunkHeaderStyle := lipgloss.
		NewStyle().
		Foreground(lipgloss.Cyan)

	for fileIndex, file := range p.Files {
		if fileIndex != 0 {
			s.WriteString(divider)
			s.WriteString("\n")
		}
		s.WriteString("--- ")
		s.WriteString(file.OldPath)
		s.WriteString("\n")

		s.WriteString("+++ ")
		s.WriteString(file.NewPath)
		s.WriteString("\n")

		s.WriteString(divider)
		s.WriteString("\n")

		for _, hunk := range file.Hunks {
			header := fmt.Sprintf("@@ -%d +%d @@", hunk.OldCount, hunk.NewCount)
			s.WriteString(hunkHeaderStyle.Render(header))
			s.WriteString("\n")

			for _, line := range hunk.Lines {
				lineWithPrefix := line.Type.String() + line.Text

				switch line.Type {
				case patch.DiffLineContext:
					s.WriteString(lineWithPrefix)
				case patch.DiffLineRemoved:
					s.WriteString(oldLineStyle.Render(lineWithPrefix))
				case patch.DiffLineAdded:
					s.WriteString(newLineStyle.Render(lineWithPrefix))
				}

				s.WriteString("\n")
			}
		}
	}

	return s.String()
}

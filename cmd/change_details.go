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

		m.buildPrettyDetails()
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

func (m *changeDetailsModel) buildPrettyDetails() {
	change := m.change
	p := m.patch
	maxWidth := m.width - 6

	var s strings.Builder

	maxWidthStyle := lipgloss.NewStyle().Width(maxWidth)

	hashStyle := lipgloss.
		NewStyle().
		Inherit(maxWidthStyle)

	s.WriteString(hashStyle.Render(" " + p.Metadata.Hash))
	s.WriteString("\n")

	s.WriteString(maxWidthStyle.Render(fmt.Sprintf(" %s  %s", change.Project, change.Branch)))
	s.WriteString("\n")

	authorStyle := lipgloss.
		NewStyle().
		Inherit(maxWidthStyle).
		Foreground(lipgloss.BrightMagenta)

	s.WriteString(authorStyle.Render(fmt.Sprintf(" %s", userDisplayName(&change.Author))))
	s.WriteString("\n")

	statusStyle := maxWidthStyle.Inherit(getReviewStatusStyle(change.Review.Primary))

	s.WriteString(statusStyle.Render(reviewStatusString(change.Review.Primary, false)))
	s.WriteString("\n\n")

	titleStyle := lipgloss.
		NewStyle().
		Inherit(maxWidthStyle).
		Bold(true)
	s.WriteString(titleStyle.Render(change.Title))
	s.WriteString("\n\n")

	s.WriteString(maxWidthStyle.Render(p.Metadata.Body))
	s.WriteString("\n\n")

	dividerStyle := lipgloss.NewStyle().Foreground(lipgloss.BrightBlack)
	divider := dividerStyle.Render(strings.Repeat("─", min(72, maxWidth)))
	s.WriteString(divider)
	s.WriteString("\n")

	for fileIndex, file := range p.Files {

		if fileIndex != 0 {
			s.WriteString(divider)
			s.WriteString("\n")
		}
		s.WriteString(maxWidthStyle.Render("--- " + file.OldPath))
		s.WriteString("\n")

		s.WriteString(maxWidthStyle.Render("+++ " + file.NewPath))
		s.WriteString("\n")

		s.WriteString(divider)
		s.WriteString("\n")

		if maxWidth < 160 {
			s.WriteString(prettyDiffUnified(file, maxWidth))
		} else {
			s.WriteString(prettyDiffSideBySide(file, maxWidth))
		}
	}

	m.prettyDetails = s.String()
	m.patchLineCount = len(strings.Split(m.prettyDetails, "\n"))
}

func prettyDiffUnified(f patch.FileDiff, width int) string {
	var s strings.Builder

	maxWidthStyle := lipgloss.NewStyle().Width(width)
	oldLineStyle := lipgloss.
		NewStyle().
		Inherit(maxWidthStyle).
		Foreground(lipgloss.BrightRed)

	newLineStyle := lipgloss.
		NewStyle().
		Inherit(maxWidthStyle).
		Foreground(lipgloss.BrightGreen)

	hunkHeaderStyle := lipgloss.
		NewStyle().
		Inherit(maxWidthStyle).
		Foreground(lipgloss.Cyan)

	noNewLineStyle := lipgloss.
		NewStyle().
		Inherit(maxWidthStyle).
		Foreground(lipgloss.BrightMagenta)

	for _, hunk := range f.Hunks {
		header := fmt.Sprintf("@@ -%d +%d @@", hunk.OldCount, hunk.NewCount)
		s.WriteString(hunkHeaderStyle.Render(header))
		s.WriteString("\n")

		for _, line := range hunk.Lines {
			lineWithPrefix := line.Type.String() + line.Text

			switch line.Type {
			case patch.DiffLineContext:
				s.WriteString(maxWidthStyle.Render(lineWithPrefix))
			case patch.DiffLineRemoved:
				s.WriteString(oldLineStyle.Render(lineWithPrefix))
			case patch.DiffLineAdded:
				s.WriteString(newLineStyle.Render(lineWithPrefix))
			}

			s.WriteString("\n")

			if line.NoNewlineAtEOF {
				s.WriteString(noNewLineStyle.Render("No newline at end of file"))
				s.WriteString("\n")
			}
		}
	}

	return s.String()
}

func prettyDiffSideBySide(f patch.FileDiff, width int) string {
	var s strings.Builder

	columnWidth := max(1, width/2)

	maxWidthStyle := lipgloss.NewStyle().Width(columnWidth)

	oldLineStyle := lipgloss.
		NewStyle().
		Inherit(maxWidthStyle).
		Foreground(lipgloss.BrightRed)

	newLineStyle := lipgloss.
		NewStyle().
		Inherit(maxWidthStyle).
		Foreground(lipgloss.BrightGreen)

	hunkHeaderStyle := lipgloss.
		NewStyle().
		Inherit(maxWidthStyle).
		Foreground(lipgloss.Cyan)

	noNewLineStyle := lipgloss.
		NewStyle().
		Inherit(maxWidthStyle).
		Foreground(lipgloss.BrightMagenta)

	noNewLineStr := noNewLineStyle.Render("No newline at end of file")

	for _, hunk := range f.Hunks {
		removed := make([]patch.DiffLine, 0)
		added := make([]patch.DiffLine, 0)

		// removedLineNum := 0
		// addedLineNum := 0

		removedHeader := hunkHeaderStyle.Render(fmt.Sprintf("@@ -%d @@", hunk.OldCount))
		addedHeader := hunkHeaderStyle.Render(fmt.Sprintf("@@ +%d @@", hunk.NewCount))
		s.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, removedHeader, addedHeader))
		s.WriteString("\n")

		flush := func() {
			for i := range max(len(added), len(removed)) {
				removeStr := maxWidthStyle.Render("")
				addStr := maxWidthStyle.Render("")
				oldMissingNewLine := false
				newMissingNewLine := false

				if i < len(removed) {
					line := removed[i]
					removeStr = oldLineStyle.Render(line.Type.String() + line.Text)
					oldMissingNewLine = line.NoNewlineAtEOF
				}

				if i < len(added) {
					line := added[i]
					addStr = newLineStyle.Render(line.Type.String() + line.Text)
					newMissingNewLine = line.NoNewlineAtEOF
				}

				line := lipgloss.JoinHorizontal(lipgloss.Top, removeStr, addStr)
				s.WriteString(line)
				s.WriteString("\n")

				if oldMissingNewLine || newMissingNewLine {
					oldCell := maxWidthStyle.Render("")
					newCell := maxWidthStyle.Render("")

					if oldMissingNewLine {
						oldCell = noNewLineStr
					}

					if newMissingNewLine {
						newCell = noNewLineStr
					}

					s.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, oldCell, newCell))
					s.WriteString("\n")
				}
			}

			removed = removed[:0]
			added = added[:0]
		}

		for _, line := range hunk.Lines {
			switch line.Type {
			case patch.DiffLineContext:
				flush()
				str := maxWidthStyle.Render(line.Type.String() + line.Text)
				s.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, str, str))
				s.WriteString("\n")

			case patch.DiffLineRemoved:
				removed = append(removed, line)
			case patch.DiffLineAdded:
				added = append(added, line)
			}
		}

		flush()
	}
	return s.String()
}

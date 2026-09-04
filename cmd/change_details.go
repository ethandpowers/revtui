package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ethandpowers/revtui/internal/patch"
)

type changeDetailsModel struct {
	backend              Backend
	change               Change
	patch                *patch.Patch
	filesColWidth        int
	renderedFileColWidth int
	isFileFocused        bool
	renderedFile         string
	activeFileLineCount  int
	err                  error
	filesScrollOffset    int
	filesCursor          int
	diffCursor           int
	width                int
	height               int
	lastTime             time.Time
	lastKey              string
}

func (m *changeDetailsModel) maxDiffCursor() int {
	return max(0, m.activeFileLineCount-m.height+2)
}

func (m changeDetailsModel) getVisibleFileCount() int {
	return max(0, m.height-2)
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
			if m.isFileFocused {
				if m.diffCursor > 0 {
					m.diffCursor--
				}
			} else {
				if m.filesCursor > 0 {
					m.filesCursor--
					m.diffCursor = 0
				} else if m.filesScrollOffset > 0 {
					m.filesScrollOffset--
					m.diffCursor = 0
				}
			}

		case "down", "j":
			if m.isFileFocused {
				if m.diffCursor < m.maxDiffCursor() {
					m.diffCursor++
				}
			} else {
				rowsVisible := m.getVisibleFileCount()
				numFileRows := len(m.patch.Files) + 1
				if m.filesCursor < rowsVisible-1 && m.filesCursor < numFileRows-1 {
					m.filesCursor++
					m.diffCursor = 0
				} else if m.filesCursor+m.filesScrollOffset < numFileRows-1 {
					m.filesScrollOffset++
					m.diffCursor = 0
				}
			}

		case "pgup":
			m.diffCursor -= m.height - 4
			if m.diffCursor < 0 {
				m.diffCursor = 0
			}

		case "pgdown":
			m.diffCursor += m.height - 4
			if m.diffCursor > m.maxDiffCursor() {
				m.diffCursor = m.maxDiffCursor()
			}

		case "G":
			m.diffCursor = m.maxDiffCursor()

		case "g":
			if m.lastKey == "g" && time.Since(m.lastTime) < 300*time.Millisecond {
				m.diffCursor = 0
				m.lastKey = ""
			} else {
				m.lastKey = msg.String()
				m.lastTime = time.Now()
			}
		case "enter":
			m.isFileFocused = true
		case "esc":
			m.isFileFocused = false
		}
		m.renderActiveFile()

	case patchLoadedMsg:
		parsedPatch, err := patch.ParsePatch(msg.patch)
		m.patch = &parsedPatch
		m.err = err
		if err != nil {
			break
		}

		m.diffCursor = 0
		m.renderActiveFile()

		return m, nil
	}
	return m, nil
}

func (m changeDetailsModel) View() string {
	content := ""
	if m.err != nil {
		content = fmt.Sprintf("Error: %s", m.err.Error())
	} else if m.patch != nil {
		content = m.prettyDetails()
	}

	return content
}

func (m *changeDetailsModel) prettyDetails() string {
	fileList := m.renderFileList(m.filesColWidth)
	activeFile := m.renderedFile

	return lipgloss.JoinHorizontal(lipgloss.Top, fileList, activeFile)
}

func (m changeDetailsModel) renderFileList(width int) string {
	textWidth := width - 4
	boxStyle := lipgloss.
		NewStyle().
		Width(width).
		Height(m.height).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder())

	if !m.isFileFocused {
		boxStyle = boxStyle.BorderForeground(lipgloss.Yellow)
	}

	defaultFileStyle := lipgloss.
		NewStyle().
		Width(textWidth).
		Align(lipgloss.Right)

	var s strings.Builder

	rowCount := min(m.getVisibleFileCount(), len(m.patch.Files)+1-m.filesScrollOffset)

	for i := range rowCount {
		fileIndex := i + m.filesScrollOffset
		rowStyle := lipgloss.NewStyle().Inherit(defaultFileStyle)

		var rowText string

		if fileIndex == 0 {
			rowText = "Commit Message"
		} else {
			file := &m.patch.Files[fileIndex-1]

			if len(file.NewPath) == 0 {
				// rowStyle = rowStyle.Background(lipgloss.Red).Foreground(lipgloss.Black)
				rowStyle = rowStyle.Foreground(lipgloss.Red)
				rowText = leftTruncate(file.OldPath, textWidth)
			} else {
				if len(file.OldPath) == 0 {
					// rowStyle = rowStyle.Background(lipgloss.Green).Foreground(lipgloss.Black)
					rowStyle = rowStyle.Foreground(lipgloss.Green)
				}
				rowText = leftTruncate(file.NewPath, textWidth)
			}
		}

		if m.filesCursor+m.filesScrollOffset == fileIndex {
			rowStyle = rowStyle.Background(lipgloss.Blue)
		}

		path := rowStyle.Render(rowText)
		s.WriteString(path)
		if i < rowCount-1 {
			s.WriteString("\n")
		}
	}

	return boxStyle.Render(s.String())
}

func (m *changeDetailsModel) renderActiveFile() {
	fileIndex := m.filesScrollOffset + m.filesCursor
	var content string
	if fileIndex == 0 {
		content = m.renderCommitMsgAndHeader(m.renderedFileColWidth)
	} else {
		width := m.renderedFileColWidth - 4
		f := m.patch.Files[fileIndex-1]
		var s strings.Builder

		dividerStyle := lipgloss.NewStyle().Foreground(lipgloss.BrightBlack)
		divider := dividerStyle.Render(strings.Repeat("─", width))
		s.WriteString(divider)
		s.WriteString("\n")

		maxWidthStyle := lipgloss.NewStyle().Width(width)

		greatestLineNo := 0

		for _, hunk := range f.Hunks {
			last := max(hunk.OldStart+hunk.OldCount, hunk.NewStart+hunk.NewCount)
			greatestLineNo = max(greatestLineNo, last)
		}

		lineNoStr := strconv.Itoa(greatestLineNo)

		s.WriteString(maxWidthStyle.Render("--- " + f.OldPath))
		s.WriteString("\n")

		s.WriteString(maxWidthStyle.Render("+++ " + f.NewPath))
		s.WriteString("\n")

		s.WriteString(divider)
		s.WriteString("\n")

		if width < 170 {
			s.WriteString(prettyDiffUnified(f, width))
		} else {
			s.WriteString(prettyDiffSideBySide(f, width, len(lineNoStr)))
		}

		content = s.String()
	}

	lines := strings.Split(content, "\n")
	start := max(0, min(m.diffCursor, len(lines)-1))
	end := min(start+m.height-2, len(lines))

	m.activeFileLineCount = len(lines)

	content = strings.Join(lines[start:end], "\n")

	boxStyle := lipgloss.
		NewStyle().
		Width(m.renderedFileColWidth).
		Height(m.height).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder())

	if m.isFileFocused {
		boxStyle = boxStyle.BorderForeground(lipgloss.Yellow)
	}

	m.renderedFile = boxStyle.Render(content)
}

func (m changeDetailsModel) renderCommitMsgAndHeader(width int) string {
	change := m.change
	p := m.patch
	var s strings.Builder

	maxWidthStyle := lipgloss.NewStyle().Width(width)

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

	return s.String()
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

func prettyDiffSideBySide(f patch.FileDiff, width int, lineNoMaxDigits int) string {
	var s strings.Builder

	columnWidth := max(1, width/2)

	maxWidthStyle := lipgloss.NewStyle().Width(columnWidth)

	oldLineStyle := lipgloss.
		NewStyle().
		Foreground(lipgloss.BrightRed)

	newLineStyle := lipgloss.
		NewStyle().
		Foreground(lipgloss.BrightGreen)

	hunkHeaderStyle := lipgloss.
		NewStyle().
		Inherit(maxWidthStyle).
		Foreground(lipgloss.Cyan)

	noNewLineStyle := lipgloss.
		NewStyle().
		Inherit(maxWidthStyle).
		Foreground(lipgloss.BrightMagenta)

	lineNoStyle := lipgloss.
		NewStyle().
		Width(lineNoMaxDigits).
		Align(lipgloss.Right).
		Foreground(lipgloss.BrightBlack)

	noNewLineStr := noNewLineStyle.Render("No newline at end of file")

	for _, hunk := range f.Hunks {
		removed := make([]patch.DiffLine, 0)
		added := make([]patch.DiffLine, 0)

		oldLineNo := hunk.OldStart
		newLineNo := hunk.NewStart

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
					lineNo := strconv.Itoa(oldLineNo)
					lineNoStyled := lineNoStyle.Render(lineNo)
					oldLineNo++

					unpadded := lineNoStyled + " " + oldLineStyle.Render(line.Type.String()+line.Text)
					removeStr = maxWidthStyle.Render(unpadded)
					oldMissingNewLine = line.NoNewlineAtEOF
				}

				if i < len(added) {
					line := added[i]
					lineNo := strconv.Itoa(newLineNo)
					lineNoStyled := lineNoStyle.Render(lineNo)
					newLineNo++

					unpadded := lineNoStyled + " " + newLineStyle.Render(line.Type.String()+line.Text)
					addStr = maxWidthStyle.Render(unpadded)
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
				lineContent := line.Type.String() + line.Text
				oldContent := lineNoStyle.Render(strconv.Itoa(oldLineNo)) + " " + lineContent
				newContent := lineNoStyle.Render(strconv.Itoa(newLineNo)) + " " + lineContent

				old := maxWidthStyle.Render(oldContent)
				new := maxWidthStyle.Render(newContent)

				s.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, old, new))
				s.WriteString("\n")
				oldLineNo++
				newLineNo++

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

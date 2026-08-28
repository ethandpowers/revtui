package main

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type changeDetailsModel struct {
	backend        Backend
	change         Change
	patch          string
	patchLineCount int
	cursor         int
	width          int
	height         int
	lastTime       time.Time
	lastKey        string
}

func (m *changeDetailsModel) maxCursor() int {
	return m.patchLineCount - m.height + 3
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
		m.patch = msg.patch
		m.patchLineCount = len(strings.Split(msg.patch, "\n"))
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
	if len(m.patch) > 0 {
		addStyle := lipgloss.NewStyle().Foreground(lipgloss.Green)
		removeStyle := lipgloss.NewStyle().Foreground(lipgloss.Red)
		lines := strings.Split(m.patch, "\n")
		for index, line := range lines {
			if strings.HasPrefix(line, "+") {
				lines[index] = addStyle.Render(line)
			} else if strings.HasPrefix(line, "-") {
				lines[index] = removeStyle.Render(line)
			}
		}
		start := min(m.cursor, len(lines)-1)
		end := min(start+m.height-4, len(lines))
		content = strings.Join(lines[start:end], "\n")
	}
	return boxStyle.Render(content)
}

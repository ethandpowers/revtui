package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type changeDetailsModel struct {
	backend        Backend
	change         Change
	patch          string
	patchLineCount int
	cursor         int
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

		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < m.patchLineCount-1 {
				m.cursor++
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

func (m changeDetailsModel) View(width int, height int) string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Width(width).
		Height(max(0, height-1))

	content := ""
	if len(m.patch) > 0 {
		lines := strings.Split(m.patch, "\n")
		start := min(m.cursor, len(lines)-1)
		end := min(start+height-5, len(lines))
		content = strings.Join(lines[start:end], "\n")
	}
	return boxStyle.Render(content)
}

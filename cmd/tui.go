package main

import (
	"fmt"
	"os"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type columnConfig struct {
	longestChangeID int
	longestSubject  int
	longestOwner    int
}

type model struct {
	width  int
	height int

	backend      Backend
	changes      []Change
	cursor       int
	columnConfig columnConfig

	loading bool
	spinner spinner.Model

	err error
}

func initialModel(backend Backend) model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	// s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return model{
		backend: backend,
		changes: make([]Change, 0),
		cursor:  0,
		loading: true,
		spinner: s,
	}
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

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		loadChangesCmd(m.backend),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
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
		m.loading = false
		m.columnConfig = columnConfig{
			longestChangeID,
			longestSubject,
			longestOwner,
		}
		m.err = msg.err
		return m, nil

	case tea.KeyPressMsg:

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.changes)-1 {
				m.cursor++
			}

		}
	}

	if m.loading {
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) renderChangeRow(i int) string {
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

func (m model) View() tea.View {
	if m.loading {
		v := tea.NewView(m.spinner.View() + " Loading changes...\n")
		v.AltScreen = true
		return v
	}

	if m.err != nil {
		v := tea.NewView(fmt.Sprintf("Error: %s\n", m.err))
		v.AltScreen = true
		return v
	}

	s := fmt.Sprintf(
		"  %-*s %-*s %-*s\n",
		m.columnConfig.longestChangeID,
		ChangeIDField,
		m.columnConfig.longestSubject,
		SubjectField,
		m.columnConfig.longestOwner,
		OwnerField,
	)

	mainViewportSize := m.height - 2

	for i := 0; i < mainViewportSize; i++ {
		scrollOffset := 0

		if m.cursor >= mainViewportSize {
			scrollOffset = m.cursor - mainViewportSize + 1
		}

		changeIndex := i + scrollOffset
		if len(m.changes) > changeIndex {
			s += m.renderChangeRow(changeIndex)
		}
		s += "\n"
	}

	s += "q: quit\n"

	v := tea.NewView(s)
	v.AltScreen = true

	return v
}

func renderTUI(client Backend) {
	p := tea.NewProgram(initialModel(client))

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

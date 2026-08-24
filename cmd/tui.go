package main

import (
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type model struct {
	width  int
	height int

	backend Backend
	changes []Change

	loading      bool
	spinner      spinner.Model
	message      string
	showDetails  bool
	changesModel changeListModel
	detailsModel changeDetailsModel
	err          error
}

type startLoadingMsg struct {
	message string
}

func startLoading(message string) tea.Cmd {
	return func() tea.Msg {
		return startLoadingMsg{message: message}
	}
}

type stopLoadingMsg struct {
	message string
	err     error
}

func stopLoading(message string, err error) tea.Cmd {
	return func() tea.Msg {
		return stopLoadingMsg{message, err}
	}
}

type showDetailsMsg struct {
	change Change
}

func showDetails(change Change) tea.Cmd {
	return func() tea.Msg {
		return showDetailsMsg{change}
	}
}

func initialModel(backend Backend) model {
	s := spinner.New()
	s.Spinner = spinner.Dot

	return model{
		backend: backend,
		changes: make([]Change, 0),
		changesModel: changeListModel{
			backend: backend,
			changes: make([]Change, 0),
			cursor:  0,
		},
		loading:     true,
		spinner:     s,
		showDetails: false,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.changesModel.Init(),
	)
}

func (m model) updateChildren(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if m.showDetails {
		var cmd tea.Cmd
		m.detailsModel, cmd = m.detailsModel.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		var cmd tea.Cmd
		m.changesModel, cmd = m.changesModel.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:

		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case startLoadingMsg:
		m.loading = true
		m.message = msg.message
		return m, m.spinner.Tick

	case stopLoadingMsg:
		m.loading = false
		m.message = msg.message
		m.err = msg.err
		return m, nil

	case showDetailsMsg:
		m.showDetails = true
		m.detailsModel.change = msg.change
		return m, nil

	case tea.KeyPressMsg:

		switch msg.String() {

		case "ctrl+c", "q":
			return m, tea.Quit

		case "esc":
			m.showDetails = false
			return m, nil
		}
	}

	return m.updateChildren(msg)
}

func (m model) renderFooter() string {
	const shortcutHints = "c: checkout | w: checkout to worktree | p: cherry-pick | q: quit"
	var message string

	if m.loading {
		message = m.spinner.View() + " Loading ..."
	} else if m.err != nil {
		message = fmt.Sprintf("Error: %s", m.err.Error())
	} else {
		message = m.message
	}

	if m.width <= 0 {
		return strings.TrimSpace(message + " " + shortcutHints)
	}

	shortcutWidth := lipgloss.Width(shortcutHints)
	if shortcutWidth >= m.width {
		return truncateRunes(shortcutHints, m.width)
	}

	messageMaxWidth := m.width - shortcutWidth - 1
	message = truncateRunes(message, messageMaxWidth)
	spaces := m.width - lipgloss.Width(message) - shortcutWidth

	return message + strings.Repeat(" ", spaces) + shortcutHints
}

func truncateRunes(s string, width int) string {
	if width <= 0 {
		return ""
	}

	runes := []rune(s)
	if len(runes) <= width {
		return s
	}

	return string(runes[:width])
}

func (m model) View() tea.View {
	s := ""
	if m.showDetails {
		s = m.detailsModel.View(m.width, m.height)
	} else {
		s = m.changesModel.View(m.width, m.height)
	}

	s += "\n" + m.renderFooter()
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

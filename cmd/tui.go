package main

import (
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type changesViewMode int

const (
	changeList changesViewMode = iota
	changeGrid
)

type model struct {
	width  int
	height int

	backend Backend
	changes []Change

	loading         bool
	spinner         spinner.Model
	message         string
	showDetails     bool
	changesMode     changesViewMode
	changeListModel changeListModel
	changeGridModel changeGridModel
	detailsModel    changeDetailsModel
	err             error
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

	changes := make([]Change, 0)

	return model{
		backend:     backend,
		changes:     make([]Change, 0),
		changesMode: changeList,
		changeListModel: changeListModel{
			backend: backend,
			changes: changes,
			cursor:  0,
		},
		changeGridModel: changeGridModel{
			backend: backend,
			columns: []changeGridColModel{
				{ReviewStatusNotReady, make([]Change, 0)},
				{ReviewStatusReadyForReview, make([]Change, 0)},
				{ReviewStatusReviewed, make([]Change, 0)},
				{ReviewStatusVerified, make([]Change, 0)},
				{ReviewStatusBlocked, make([]Change, 0)},
				{ReviewStatusUnknown, make([]Change, 0)},
			},
			xCursor: 0,
			yCursor: 0,
		},
		loading:     true,
		spinner:     s,
		showDetails: false,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.changeListModel.Init(),
		m.changeGridModel.Init(),
		loadChangesCmd(m.backend),
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
		if m.changesMode == changeList {
			m.changeListModel, cmd = m.changeListModel.Update(msg)
		} else if m.changesMode == changeGrid {
			m.changeGridModel, cmd = m.changeGridModel.Update(msg)
		}
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

		m.detailsModel.width = msg.Width
		m.detailsModel.height = msg.Height - 1

		m.changeGridModel.width = msg.Width
		m.changeGridModel.height = msg.Height - 1
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

	case patchLoadedMsg:
		m.loading = false
		m.err = msg.err
		// fall through to updateChildren so detailsModel sets the patch

	case changesLoadedMsg:
		m.loading = false
		m.err = msg.err
		m.changes = msg.changes
		m.changeListModel.changes = msg.changes

		for _, change := range msg.changes {
			for i, col := range m.changeGridModel.columns {
				if col.status == change.Review.Primary {
					m.changeGridModel.columns[i].changes = append(m.changeGridModel.columns[i].changes, change)
				}
			}
		}
		// fall through so the list/grid can do any necessary setup

	case showDetailsMsg:
		m.showDetails = true
		m.detailsModel.backend = m.backend
		m.detailsModel.change = msg.change
		return m, tea.Batch(startLoading(""), fetchPatchCmd(m.backend, msg.change))

	case tea.KeyPressMsg:

		switch msg.String() {

		case "ctrl+c", "q":
			return m, tea.Quit

		case "esc":
			m.showDetails = false
			m.detailsModel.patch = ""
			return m, nil

		case "m":
			if m.changesMode == changeList {
				m.changesMode = changeGrid
			} else {
				m.changesMode = changeList
			}
		}
	}

	return m.updateChildren(msg)
}

func (m model) renderFooter() string {
	modeHint := ""
	if !m.showDetails {
		if m.changesMode == changeList {
			modeHint = "m: toggle grid | "
		} else {
			modeHint = "m: toggle list | "
		}
	}
	shortcutHints := modeHint + "c: checkout | w: checkout to worktree | p: cherry-pick | q: quit"
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
		s = m.detailsModel.View()
	} else if m.changesMode == changeList {
		s = m.changeListModel.View(m.width, m.height)
	} else if m.changesMode == changeGrid {
		s = m.changeGridModel.View()
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

package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/pkg/browser"
)

type model struct {
	width  int
	height int

	backend Backend
	changes []Change

	loading       bool
	spinner       spinner.Model
	message       string
	showDetails   bool
	detailsModel  changeDetailsModel
	listViewModel changeListViewModel
	err           error
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
		backend: backend,
		changes: make([]Change, 0),
		listViewModel: changeListViewModel{
			changesMode: changeList,
			changeListModel: changeListModel{
				changes: changes,
				cursor:  0,
			},
			changeGridModel: changeGridModel{
				columns: []changeGridColModel{
					{ReviewStatusNotReady, make([]Change, 0), 0},
					{ReviewStatusReadyForReview, make([]Change, 0), 0},
					{ReviewStatusReviewed, make([]Change, 0), 0},
					{ReviewStatusVerified, make([]Change, 0), 0},
					{ReviewStatusBlocked, make([]Change, 0), 0},
					{ReviewStatusUnknown, make([]Change, 0), 0},
				},
				xCursor: 0,
				yCursor: 0,
			},
		},
		loading:     true,
		spinner:     s,
		showDetails: false,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.listViewModel.Init(),
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
		m.listViewModel, cmd = m.listViewModel.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) getActiveChange() *Change {
	if m.showDetails {
		return &m.detailsModel.change
	} else {
		return m.listViewModel.getActiveChange()
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.detailsModel.width = msg.Width
		m.detailsModel.height = msg.Height - 1
		m.detailsModel.filesColWidth = min(80, m.width/4)
		m.detailsModel.renderedFileColWidth = m.width - m.detailsModel.filesColWidth
		if m.detailsModel.patch != nil {
			m.detailsModel.renderActiveFile()
		}
		// fall through so the list/grid can adjust layout

	case startLoadingMsg:
		m.loading = true
		m.message = msg.message
		return m, m.spinner.Tick

	case stopLoadingMsg:
		m.loading = false
		m.message = msg.message
		m.err = msg.err
		return m, nil

	case checkoutMsg:
		return m, stopLoading(msg.message, msg.err)

	case patchLoadedMsg:
		m.loading = false
		m.err = msg.err
		// fall through to updateChildren so detailsModel sets the patch

	case changesLoadedMsg:
		m.loading = false
		m.err = msg.err
		m.changes = msg.changes
		// fall through so the list/grid can do any necessary setup

	case showDetailsMsg:
		m.showDetails = true
		m.detailsModel.backend = m.backend
		m.detailsModel.change = msg.change
		m.detailsModel.filesCursor = 0
		m.detailsModel.filesScrollOffset = 0
		m.detailsModel.isFileFocused = false
		return m, tea.Sequence(startLoading(""), fetchPatchCmd(m.backend, msg.change))

	case tea.KeyPressMsg:

		switch msg.String() {

		case "ctrl+c", "q":
			return m, tea.Quit

		case "esc":
			if m.detailsModel.isFileFocused {
				break
			}
			m.showDetails = false
			m.detailsModel.patch = nil
			m.detailsModel.err = nil
			return m, nil

		case "c":
			if len(m.changes) == 0 {
				return m, nil
			}

			change := m.getActiveChange()
			return m, tea.Sequence(startLoading(fmt.Sprintf("Checking out %s", change.Title)), checkoutChangeCmd(*m.getActiveChange(), m.backend))

		case "o":
			if len(m.changes) == 0 {
				return m, nil
			}

			change := m.getActiveChange()
			url, err := m.backend.GetChangeUrl(*change)
			if err != nil {
				m.err = err
				return m, nil
			}

			browser.Stdout = io.Discard
			browser.Stderr = io.Discard
			browser.OpenURL(url)

		case "r":
			cmds := []tea.Cmd{startLoading("")}
			cmds = append(cmds, loadChangesCmd(m.backend))
			if m.showDetails {
				cmds = append(cmds, fetchPatchCmd(m.backend, *m.getActiveChange()))
			}

			return m, tea.Batch(cmds...)

		case "enter":
			if !m.showDetails {
				if len(m.changes) == 0 {
					return m, nil
				}

				return m, showDetails(*m.getActiveChange())
			}
		}
	}

	return m.updateChildren(msg)
}

func (m model) renderFooter() string {
	modeHint := ""
	if !m.showDetails {
		// TODO: The keycombos hint should be handled more elegantly.  Maybe by storing a list of combo -> hint mappings?
		if m.listViewModel.changesMode == changeList {
			modeHint = "m: toggle grid | "
		} else {
			modeHint = "m: toggle list | "
		}
	}
	shortcutHints := modeHint + "c: checkout | o: open in browser | r: refresh | q: quit"
	var message string

	if m.loading {
		text := "Loading ..."
		if len(m.message) > 0 {
			text = m.message
		}

		message = m.spinner.View() + text
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
	} else {
		s = m.listViewModel.View()
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

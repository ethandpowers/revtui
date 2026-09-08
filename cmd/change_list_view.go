package main

import (
	tea "charm.land/bubbletea/v2"
)

type changesViewMode int

const (
	changeList changesViewMode = iota
	changeGrid
)

type changeListView interface {
	SelectedChange() *Change
}

type changeListViewModel struct {
	width  int
	height int

	changesMode     changesViewMode
	changeListModel changeListModel
	changeGridModel changeGridModel
}

func (m changeListViewModel) getChangeListView() changeListView {
	if m.changesMode == changeList {
		return m.changeListModel
	} else if m.changesMode == changeGrid {
		return m.changeGridModel
	}

	return nil
}

func (m changeListViewModel) getActiveChange() *Change {
	listView := m.getChangeListView()
	if listView == nil {
		return nil
	}
	return listView.SelectedChange()
}

func (m changeListViewModel) Init() tea.Cmd {
	return tea.Batch(m.changeGridModel.Init(), m.changeListModel.Init())
}

func (m changeListViewModel) Update(msg tea.Msg) (changeListViewModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.changeGridModel.width = msg.Width
		m.changeGridModel.height = msg.Height - 1

		m.changeListModel.width = msg.Width
		m.changeListModel.height = msg.Height - 1
		return m, nil

	case changesLoadedMsg:
		m.changeListModel.changes = msg.changes

		for i, col := range m.changeGridModel.columns {
			m.changeGridModel.columns[i].changes = m.changeGridModel.columns[i].changes[:0]
			for _, change := range msg.changes {
				if col.status == change.Review.Primary {
					m.changeGridModel.columns[i].changes = append(m.changeGridModel.columns[i].changes, change)
				}
			}
		}
		// fall through so the list/grid can do any necessary setup

	case tea.KeyPressMsg:

		switch msg.String() {
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

func (m changeListViewModel) View() string {
	s := ""
	if m.changesMode == changeList {
		s = m.changeListModel.View()
	} else if m.changesMode == changeGrid {
		s = m.changeGridModel.View()
	}

	return s
}

func (m changeListViewModel) updateChildren(msg tea.Msg) (changeListViewModel, tea.Cmd) {
	var cmd tea.Cmd
	if m.changesMode == changeGrid {
		m.changeGridModel, cmd = m.changeGridModel.Update(msg)
	} else if m.changesMode == changeList {
		m.changeListModel, cmd = m.changeListModel.Update(msg)
	}

	return m, cmd
}

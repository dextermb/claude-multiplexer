package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func newSearchInput() textinput.Model {
	in := textinput.New()
	in.Placeholder = "filter sessions"
	in.Prompt = "/ "
	in.CharLimit = 64
	return in
}

// searchActive reports whether the search box narrows the list: it is focused,
// or it holds a needle.
func (m Model) searchActive() bool {
	return m.searchOn || m.search.Value() != ""
}

// searchRows is the number of sidebar lines the search box takes: one when it is
// active, none when it is not.
func (m Model) searchRows() int {
	if m.searchActive() {
		return 1
	}
	return 0
}

func (m Model) searchNeedle() string {
	return strings.ToLower(strings.TrimSpace(m.search.Value()))
}

// rowMatches reports whether a row passes the search needle. An empty needle
// matches every row.
func rowMatches(item row, needle string) bool {
	if needle == "" {
		return true
	}
	return strings.Contains(strings.ToLower(item.displayName()), needle) ||
		strings.Contains(strings.ToLower(item.name), needle) ||
		strings.Contains(strings.ToLower(item.dir), needle)
}

func (m Model) focusSearch() (tea.Model, tea.Cmd) {
	m.searchOn = true
	m.focus = focusSidebar
	m.prompt.Blur()
	m.search.Focus()
	return m, textinput.Blink
}

// clearSearch drops the needle and hides the box, then restores the full list.
func (m *Model) clearSearch() {
	m.searchOn = false
	m.search.Blur()
	m.search.SetValue("")
	m.listOffset = 0
	m.refresh()
}

// searchKey handles a key while the search box is focused. The first esc blurs
// the box but keeps the needle, so the list stays narrowed; enter or an arrow
// key steps the focus into the results.
func (m Model) searchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searchOn = false
		m.search.Blur()
		return m, nil
	case "enter", "down", "up", "ctrl+j", "ctrl+k":
		m.searchOn = false
		m.search.Blur()
		m.focus = focusSidebar
		return m, nil
	}
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	m.listOffset = 0
	m.refresh()
	return m, cmd
}

// searchView draws the search box at the top of the sidebar.
func (m Model) searchView() string {
	width := m.sidebarInnerCols()
	m.search.Width = width - 2
	return searchStyle.Width(width).Render(m.search.View())
}

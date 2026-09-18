package tui

import tea "github.com/charmbracelet/bubbletea"

// modalRegion says where a modal draws, and whether it blocks the mouse wheel.
// See docs/tui.md "Where a dialog draws".
type modalRegion int

const (
	modalBody modalRegion = iota // covers the sidebar and the pane together
	modalPane                    // draws in the pane, under the session bar
)

// modal is the one active dialog. The key router, the view, and the mouse guard
// each go through this seam, so a new dialog is one adapter, not a branch in
// each. The per-session question dialog is not a modal; see docs/tui.md.
type modal interface {
	// update handles a message and returns the modal that is active next:
	// itself to stay open, or nil to close. It applies its own result to the
	// Model through the pointer.
	update(m *Model, msg tea.Msg) (modal, tea.Cmd)
	view(width, height int) string
	region() modalRegion
}

// routeModal hands a message to the active modal and stores the modal it names
// as active next. See docs/tui.md.
func (m Model) routeModal(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.modal.update(&m, msg)
	m.modal = next
	return m, cmd
}

// applyForm maps a widget's formResult to the next active modal: self stays open
// on formOpen, submit runs on formSubmitted, and nil closes on formCancelled.
func applyForm(self modal, m *Model, result formResult, cmd tea.Cmd, submit func(*Model) (modal, tea.Cmd)) (modal, tea.Cmd) {
	switch result {
	case formCancelled:
		return nil, nil
	case formSubmitted:
		return submit(m)
	}
	return self, cmd
}

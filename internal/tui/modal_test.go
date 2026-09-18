package tui

import "testing"

func jobsOf(m Model) *jobsModal      { d, _ := m.modal.(*jobsModal); return d }
func choiceOf(m Model) *choiceDialog { d, _ := m.modal.(*choiceDialog); return d }
func renameOf(m Model) *renameDialog { d, _ := m.modal.(*renameDialog); return d }
func pickerOf(m Model) *picker       { d, _ := m.modal.(*picker); return d }
func fieldsOf(m Model) *fieldForm    { d, _ := m.modal.(*fieldForm); return d }

// TestTheModalSeamClosesEveryDialogOnEsc opens each dialog through its own open
// method and feeds one esc through the router, so the one seam closes them all.
func TestTheModalSeamClosesEveryDialogOnEsc(t *testing.T) {
	opens := map[string]func(Model) (Model, bool){
		"choice": func(m Model) (Model, bool) { next, _ := m.openChoice(settingModel); return next.(Model), true },
		"rename": func(m Model) (Model, bool) { next, _ := m.openRename(); return next.(Model), true },
		"jobs":   func(m Model) (Model, bool) { next, _ := m.openJobs(); return next.(Model), true },
		"layout": func(m Model) (Model, bool) { next, _ := m.openLayoutSwitcher(); return next.(Model), true },
		"picker": func(m Model) (Model, bool) { next, _ := m.openPicker(); return next.(Model), true },
	}
	for name, open := range opens {
		t.Run(name, func(t *testing.T) {
			m, mgr := newTestModel(t, "")
			m = start(t, m, 100, 24)
			m, _ = step(t, m, key("esc"))
			m = spawn(t, m, mgr, "alpha", t.TempDir())
			m.focus = focusSidebar
			m.prompt.Blur()

			m, ok := open(m)
			if !ok {
				return
			}
			if m.modal == nil {
				t.Fatalf("%s did not open a modal", name)
			}
			m, _ = step(t, m, key("esc"))
			if m.modal != nil {
				t.Fatalf("esc did not close the %s modal", name)
			}
		})
	}
}

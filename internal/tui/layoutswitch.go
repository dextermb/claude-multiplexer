package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// layoutSwitch is the dialog that activates a layout for the selected session or
// for all sessions. The first row is "default", which clears the layout and
// takes the built-in dimensions. It does not create or edit a layout; see
// docs/tui/layouts.md.
type layoutSwitch struct {
	session       string
	names         []string
	cursor        int
	allSessions   bool
	activeSession string
	activeGlobal  string
}

func newLayoutSwitch(session string, names []string, activeSession, activeGlobal string) *layoutSwitch {
	d := &layoutSwitch{
		session:       session,
		names:         names,
		activeSession: activeSession,
		activeGlobal:  activeGlobal,
	}
	if session == "" {
		d.allSessions = true
	}
	for i, name := range names {
		if name == activeSession {
			d.cursor = i + 1
		}
	}
	return d
}

// count is the number of rows: the default row, then one per named layout.
func (d *layoutSwitch) count() int { return len(d.names) + 1 }

func (d *layoutSwitch) Update(msg tea.Msg) (formResult, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return formOpen, nil
	}
	switch key.String() {
	case "esc":
		return formCancelled, nil
	case "up", "k":
		d.cursor = (d.cursor - 1 + d.count()) % d.count()
	case "down", "j":
		d.cursor = (d.cursor + 1) % d.count()
	case "tab":
		d.allSessions = !d.allSessions
	case "enter":
		return formSubmitted, nil
	}
	return formOpen, nil
}

// choice reports the layout the cursor names, and whether it is the default row.
// The default row clears the layout, so its name is empty.
func (d *layoutSwitch) choice() (name string, isDefault bool) {
	if d.cursor <= 0 {
		return "", true
	}
	return d.names[d.cursor-1], false
}

func (d *layoutSwitch) active() string {
	if d.allSessions {
		return d.activeGlobal
	}
	return d.activeSession
}

func (d *layoutSwitch) scopeLabel() string {
	if d.allSessions {
		return "all sessions"
	}
	return "this session"
}

func (d *layoutSwitch) View(width int) string {
	inner := modalInner(width)
	rowWidth := inner - 4

	var b strings.Builder
	b.WriteString(titleStyle.Render("Layouts"))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("scope: " + d.scopeLabel()))
	b.WriteString("\n\n")

	active := d.active()
	labels := append([]string{"default"}, d.names...)
	for i, label := range labels {
		mark := "○"
		isActive := (i == 0 && active == "") || (i > 0 && d.names[i-1] == active)
		if isActive {
			mark = "◉"
		}
		rowText := mark + " " + label
		if i == d.cursor {
			b.WriteString(selectedRowStyle.Width(rowWidth).Render("▸ " + rowText))
		} else {
			b.WriteString(rowStyle.Width(rowWidth).Render("  " + rowText))
		}
		b.WriteString("\n")
	}

	if len(d.names) == 0 {
		b.WriteString("\n" + hintStyle.Render(truncate("Use the save_layout tool to make a layout.", rowWidth)))
	}
	b.WriteString("\n" + hintStyle.Render("↑↓ move · tab scope · enter apply · esc cancel"))
	return modalStyle.Width(inner).Render(b.String())
}

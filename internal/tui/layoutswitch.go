package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// layoutSwitch is the dialog that activates a layout for the selected session or
// for all sessions. It does not create or edit a layout; see docs/tui/layouts.md.
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
			d.cursor = i
		}
	}
	return d
}

func (d *layoutSwitch) Update(msg tea.Msg) (formResult, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return formOpen, nil
	}
	switch key.String() {
	case "esc":
		return formCancelled, nil
	case "up", "k":
		if len(d.names) > 0 {
			d.cursor = (d.cursor - 1 + len(d.names)) % len(d.names)
		}
	case "down", "j":
		if len(d.names) > 0 {
			d.cursor = (d.cursor + 1) % len(d.names)
		}
	case "tab":
		d.allSessions = !d.allSessions
	case "enter":
		if len(d.names) == 0 {
			return formCancelled, nil
		}
		return formSubmitted, nil
	}
	return formOpen, nil
}

func (d *layoutSwitch) chosen() string {
	if len(d.names) == 0 {
		return ""
	}
	return d.names[d.cursor]
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

	if len(d.names) == 0 {
		b.WriteString(hintStyle.Render(truncate("No layouts yet. Use the save_layout tool to make one.", rowWidth)))
		b.WriteString("\n\n" + hintStyle.Render("tab scope · esc close"))
		return modalStyle.Width(inner).Render(b.String())
	}

	active := d.active()
	for i, name := range d.names {
		mark := "○"
		if name == active {
			mark = "◉"
		}
		rowText := mark + " " + name
		if i == d.cursor {
			b.WriteString(selectedRowStyle.Width(rowWidth).Render("▸ " + rowText))
		} else {
			b.WriteString(rowStyle.Width(rowWidth).Render("  " + rowText))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n" + hintStyle.Render("↑↓ move · tab scope · enter apply · esc cancel"))
	return modalStyle.Width(inner).Render(b.String())
}

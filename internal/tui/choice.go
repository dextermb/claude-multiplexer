package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dextermb/claude-multiplexer/internal/session"
)

type settingKind int

const (
	settingModel settingKind = iota
	settingMode
	settingEffort
)

var (
	modelChoices = []string{"opus", "sonnet", "haiku"}
	modeChoices  = []string{"acceptEdits", "auto", "bypassPermissions", "default", "dontAsk", "plan"}
)

type choiceDialog struct {
	kind    settingKind
	session string
	title   string
	note    string
	options []string
	cursor  int
	current string
}

func newChoiceDialog(kind settingKind, name, current string) *choiceDialog {
	d := &choiceDialog{kind: kind, session: name, current: current}
	switch kind {
	case settingModel:
		d.title, d.options = "Model", modelChoices
	case settingMode:
		d.title, d.options = "Permission mode", modeChoices
	case settingEffort:
		d.title, d.options = "Effort", session.EffortLevels
		d.note = "resumes the session — Claude Code has no live effort switch"
	}
	for i, option := range d.options {
		if option == current {
			d.cursor = i
		}
	}
	return d
}

func (d *choiceDialog) Update(msg tea.Msg) (formResult, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return formOpen, nil
	}
	switch key.String() {
	case "esc":
		return formCancelled, nil
	case "up", "k":
		d.cursor = (d.cursor - 1 + len(d.options)) % len(d.options)
	case "down", "j":
		d.cursor = (d.cursor + 1) % len(d.options)
	case "enter":
		return formSubmitted, nil
	}
	return formOpen, nil
}

func (d *choiceDialog) chosen() string { return d.options[d.cursor] }

func (d *choiceDialog) View(width int) string {
	inner := width - 8
	if inner < 40 {
		inner = 40
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(d.title))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("for " + d.session))
	b.WriteString("\n\n")

	rowWidth := inner - 4
	for i, option := range d.options {
		mark := "○"
		if option == d.current {
			mark = "◉"
		}
		row := mark + " " + option
		if i == d.cursor {
			b.WriteString(selectedRowStyle.Width(rowWidth).Render("▸ " + row))
		} else {
			b.WriteString(rowStyle.Width(rowWidth).Render("  " + row))
		}
		b.WriteString("\n")
	}

	if d.note != "" {
		b.WriteString("\n" + hintStyle.Render(truncate(d.note, rowWidth)))
	}
	b.WriteString("\n\n" + hintStyle.Render("↑↓ move · enter apply · esc cancel"))
	return modalStyle.Width(inner).Render(b.String())
}

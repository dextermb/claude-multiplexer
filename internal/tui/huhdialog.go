package tui

import (
	"strings"

	bkey "charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

// blacklineForm is the huh theme of the interface: square, grey at rest, the
// focused field marked by a white rule, the option under the cursor inverted,
// and the focused button inverted. See docs/tui/theme.md.
func blacklineForm(bool) *huh.Styles {
	t := huh.ThemeBase(true)
	t.Form.Base = lipgloss.NewStyle()
	t.Group.Base = lipgloss.NewStyle()
	t.Group.Title = headingLabelStyle
	t.Group.Description = fgStyle(colMuted)
	t.FieldSeparator = lipgloss.NewStyle().SetString("\n\n")

	button := lipgloss.NewStyle().Padding(0, 2).MarginRight(1)
	f := &t.Focused
	f.Base = lipgloss.NewStyle().PaddingLeft(1).BorderStyle(lipgloss.NormalBorder()).BorderLeft(true).BorderForeground(colAccent)
	f.Card = f.Base
	f.Title = labelStyle.Foreground(colFg)
	f.NoteTitle = f.Title
	f.Description = fgStyle(colMuted)
	f.ErrorIndicator = fgStyle(colDanger).SetString(" ×")
	f.ErrorMessage = fgStyle(colDanger)
	f.SelectSelector = fgStyle(colFg).SetString("› ")
	f.Option = fgStyle(colSecondary)
	f.NextIndicator = fgStyle(colDimmed).MarginLeft(1).SetString("›")
	f.PrevIndicator = fgStyle(colDimmed).MarginRight(1).SetString("‹")
	f.MultiSelectSelector = fgStyle(colFg).SetString("› ")
	f.SelectedOption = invertStyle.Padding(0, 1)
	f.UnselectedOption = fgStyle(colSecondary).Padding(0, 1)
	f.SelectedPrefix = fgStyle(colFg).SetString("[■] ")
	f.UnselectedPrefix = fgStyle(colDimmed).SetString("[ ] ")
	f.FocusedButton = button.Foreground(colAccentFg).Background(colAccent)
	f.BlurredButton = button.Foreground(colSecondary)
	f.Next = f.FocusedButton
	f.TextInput.Cursor = fgStyle(colFg)
	f.TextInput.CursorText = invertStyle
	f.TextInput.Placeholder = fgStyle(colDimmed)
	f.TextInput.Prompt = fgStyle(colDimmed)
	f.TextInput.Text = fgStyle(colFg)
	f.Directory = fgStyle(colFg)
	f.File = fgStyle(colSecondary)

	t.Blurred = t.Focused
	t.Blurred.Base = t.Focused.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Card = t.Blurred.Base
	t.Blurred.Title = labelStyle
	t.Blurred.SelectedOption = fgStyle(colFg).Padding(0, 1)
	t.Blurred.MultiSelectSelector = lipgloss.NewStyle().SetString("  ")
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()

	t.Help.ShortKey = keyStyle
	t.Help.ShortDesc = fgStyle(colMuted)
	t.Help.ShortSeparator = fgStyle(colDimmed)
	return t
}

// huhKeys is the huh key map of the interface: esc closes a form, as every
// other dialog does.
func huhKeys() *huh.KeyMap {
	keys := huh.NewDefaultKeyMap()
	keys.Quit = bkey.NewBinding(bkey.WithKeys("esc"))
	return keys
}

// newHuhForm builds a huh form in the theme of the interface. The dialog draws its
// own hint line, so the form draws no help.
func newHuhForm(groups ...*huh.Group) *huh.Form {
	return huh.NewForm(groups...).
		WithTheme(huh.ThemeFunc(blacklineForm)).
		WithKeyMap(huhKeys()).
		WithShowHelp(false)
}

// runForm hands a message to a form and reports how it ended: open while it
// runs, submitted when the last field is done, cancelled on esc.
func runForm(form *huh.Form, msg tea.Msg) (*huh.Form, formResult, tea.Cmd) {
	next, cmd := form.Update(msg)
	if f, ok := next.(*huh.Form); ok {
		form = f
	}
	switch form.State {
	case huh.StateCompleted:
		return form, formSubmitted, nil
	case huh.StateAborted:
		return form, formCancelled, nil
	}
	return form, formOpen, cmd
}

// formBox draws a form in a dialog box: the title, a line that names what it
// acts on, the form, and a hint.
func formBox(width int, title, subject string, form *huh.Form, hint string) string {
	inner := modalInner(width)
	form.WithWidth(inner - 4)
	var b strings.Builder
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render(subject))
	b.WriteString("\n\n")
	b.WriteString(form.View())
	b.WriteString("\n\n" + hintStyle.Render(hint))
	return modalStyle.Width(inner + 2).Render(b.String())
}

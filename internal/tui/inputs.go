package tui

import (
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
)

// newTextArea is a Bubbles text area in the Blackline greys: your text in the
// heading grey, the placeholder and the prompt mark dimmed, and no band on the
// cursor line. See docs/tui/theme.md.
func newTextArea() textarea.Model {
	area := textarea.New()
	styles := textarea.DefaultStyles(true)
	focused := textarea.StyleState{
		Base:             lipgloss.NewStyle(),
		Text:             fgStyle(colHeading),
		LineNumber:       fgStyle(colDimmed),
		CursorLineNumber: fgStyle(colMuted),
		CursorLine:       lipgloss.NewStyle(),
		EndOfBuffer:      fgStyle(colDimmed),
		Placeholder:      fgStyle(colDimmed),
		Prompt:           fgStyle(colDimmed),
		Selection:        invertStyle,
	}
	blurred := focused
	blurred.Text = fgStyle(colSecondary)
	styles.Focused, styles.Blurred = focused, blurred
	styles.Cursor.Color = colFg
	area.SetStyles(styles)
	return area
}

// newTextInput is a Bubbles text input in the Blackline greys.
func newTextInput() textinput.Model {
	input := textinput.New()
	styles := textinput.DefaultStyles(true)
	focused := textinput.StyleState{
		Text:        fgStyle(colFg),
		Placeholder: fgStyle(colDimmed),
		Suggestion:  fgStyle(colDimmed),
		Prompt:      fgStyle(colDimmed),
	}
	blurred := focused
	blurred.Text = fgStyle(colSecondary)
	styles.Focused, styles.Blurred = focused, blurred
	styles.Cursor.Color = colFg
	input.SetStyles(styles)
	return input
}

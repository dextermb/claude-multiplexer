package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/protocol"
)

type questionDialog struct {
	session   string
	questions []protocol.Question
	step      int
	cursor    int
	chosen    []map[int]bool
	text      []textinput.Model
	err       string
}

func newQuestionDialog(session string, questions []protocol.Question) *questionDialog {
	d := &questionDialog{session: session, questions: questions}
	for range questions {
		d.chosen = append(d.chosen, make(map[int]bool))
		input := newTextInput()
		input.Placeholder = "or type an answer"
		input.Prompt = ""
		input.CharLimit = 512
		input.SetWidth(40)
		d.text = append(d.text, input)
	}
	d.syncFocus()
	return d
}

func (d *questionDialog) current() protocol.Question { return d.questions[d.step] }

func (d *questionDialog) textRow() int { return len(d.current().Options) }

func (d *questionDialog) onText() bool { return d.cursor == d.textRow() }

func (d *questionDialog) syncFocus() {
	for i := range d.text {
		d.text[i].Blur()
	}
	if d.onText() {
		d.text[d.step].Focus()
		d.text[d.step].CursorEnd()
	}
}

func (d *questionDialog) Update(msg tea.Msg) (formResult, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return formOpen, nil
	}
	switch key.String() {
	case "esc":
		return formCancelled, nil
	case "up":
		d.move(-1)
		return formOpen, nil
	case "down":
		d.move(1)
		return formOpen, nil
	case "enter":
		if d.confirmStep() {
			return formSubmitted, nil
		}
		return formOpen, nil
	case "space":
		if !d.onText() {
			d.toggle(d.cursor)
			return formOpen, nil
		}
	}
	if d.onText() {
		var cmd tea.Cmd
		d.text[d.step], cmd = d.text[d.step].Update(msg)
		return formOpen, cmd
	}
	return formOpen, nil
}

func (d *questionDialog) paste(text string) {
	if !d.onText() {
		return
	}
	input := &d.text[d.step]
	runes := []rune(input.Value())
	at := input.Position()
	if at < 0 {
		at = 0
	}
	if at > len(runes) {
		at = len(runes)
	}
	input.SetValue(string(runes[:at]) + text + string(runes[at:]))
	input.SetCursor(at + len([]rune(text)))
}

func (d *questionDialog) move(delta int) {
	rows := d.textRow() + 1
	d.cursor = (d.cursor + delta + rows) % rows
	d.syncFocus()
}

func (d *questionDialog) toggle(option int) {
	chosen := d.chosen[d.step]
	if chosen[option] {
		delete(chosen, option)
		return
	}
	if !d.current().MultiSelect {
		for key := range chosen {
			delete(chosen, key)
		}
	}
	chosen[option] = true
}

func (d *questionDialog) confirmStep() bool {
	if len(d.chosen[d.step]) == 0 && strings.TrimSpace(d.text[d.step].Value()) == "" {
		d.err = "choose an option or type an answer"
		return false
	}
	d.err = ""
	if d.step < len(d.questions)-1 {
		d.step++
		d.cursor = 0
		d.syncFocus()
		return false
	}
	return true
}

func (d *questionDialog) answer() string {
	var lines []string
	for i, question := range d.questions {
		var labels []string
		for j, option := range question.Options {
			if d.chosen[i][j] {
				labels = append(labels, option.Label)
			}
		}
		value := strings.Join(labels, ", ")
		if note := strings.TrimSpace(d.text[i].Value()); note != "" {
			if value == "" {
				value = note
			} else {
				value = fmt.Sprintf("%s (%s)", value, note)
			}
		}
		key := question.Header
		if key == "" {
			key = question.Question
		}
		lines = append(lines, fmt.Sprintf("%s: %s", key, value))
	}
	return strings.Join(lines, "\n")
}

func questionCap(caps map[string]int, bucket string) int {
	if cap, ok := caps[bucket]; ok {
		return cap
	}
	return config.DefaultQuestionCap
}

// capLines keeps at most cap lines and reports the rest as hidden. A focused
// option draws in full, and a cap below zero never caps. See docs/tui/input.md.
func capLines(lines []string, cap int, focused bool) ([]string, int) {
	if focused || cap < 0 || len(lines) <= cap {
		return lines, 0
	}
	return lines[:cap:cap], len(lines) - cap
}

// View draws the dialog inline, at the top of the output pane: a label set in a
// rule, the question, the options, and the answer field. See docs/tui/input.md.
func (d *questionDialog) View(width int, caps map[string]int) string {
	inner := width
	optionCap := questionCap(caps, config.BucketQuestionOption)
	descriptionCap := questionCap(caps, config.BucketQuestionDescription)

	question := d.current()
	var b strings.Builder
	head := "a question for you"
	if len(d.questions) > 1 {
		head = fmt.Sprintf("question %d of %d", d.step+1, len(d.questions))
	}
	b.WriteString(ruleLabel(head, inner, true))
	b.WriteString("\n\n")
	b.WriteString(questionTextStyle.Width(inner - 2).Render(" " + question.Question))
	b.WriteString("\n")
	choose := "choose one"
	if question.MultiSelect {
		choose = "choose one or more"
	}
	b.WriteString(hintStyle.Width(inner - 2).Render(" " + choose + " · the answer goes to " + d.session + " as the next prompt"))
	b.WriteString("\n\n")

	rowWidth := inner - 2
	textWidth := rowWidth - 6
	labelCol := optionLabelColumn(question.Options, textWidth)
	for i, option := range question.Options {
		mark := "( )"
		if d.chosen[d.step][i] {
			mark = "(●)"
		}
		focused := i == d.cursor

		quiet := hintStyle
		if focused {
			quiet = lipgloss.NewStyle()
		}
		var content []string
		if line, ok := inlineOption(option, labelCol, textWidth, quiet); ok {
			content = append(content, line)
		}
		labelLines, labelHidden := capLines(wrapText(option.Label, textWidth), optionCap, focused)
		if len(content) > 0 {
			labelLines, labelHidden = nil, 0
		}
		content = append(content, labelLines...)
		if labelHidden > 0 {
			content = append(content, quiet.Render(markerText(labelHidden)))
		}
		if option.Description != "" && len(labelLines) > 0 {
			descLines, descHidden := capLines(wrapText(option.Description, textWidth), descriptionCap, focused)
			for _, line := range descLines {
				content = append(content, quiet.Render(line))
			}
			if descHidden > 0 {
				content = append(content, quiet.Render(markerText(descHidden)))
			}
		}
		if len(content) == 0 {
			content = append(content, "")
		}

		var block strings.Builder
		for j, line := range content {
			if j == 0 {
				block.WriteString(mark + " " + line)
			} else {
				block.WriteString("\n      " + line)
			}
		}
		if focused {
			b.WriteString(" " + selectedRowStyle.Width(rowWidth).Render("▸ "+block.String()))
		} else {
			b.WriteString(" " + rowStyle.Width(rowWidth).Render("  "+block.String()))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(d.answerView(inner))
	b.WriteString("\n")

	if d.err != "" {
		b.WriteString("\n" + errorStyle.Render(d.err))
	}
	b.WriteString("\n\n " + paneHints("↑/↓ move · space choose · enter send · esc dismiss"))
	return b.String()
}

// optionLabelColumn is the width of the label column when every short label
// sits on one row with its description, and 0 when the options are too wide.
func optionLabelColumn(options []protocol.Option, textWidth int) int {
	widest := 0
	for _, option := range options {
		widest = maxInt(widest, lipgloss.Width(option.Label))
	}
	if widest == 0 || widest+3 > textWidth/2 {
		return 0
	}
	return widest + 3
}

// inlineOption draws a short option on one row: the label in its column, and
// the description after it, muted.
func inlineOption(option protocol.Option, labelCol, textWidth int, quiet lipgloss.Style) (string, bool) {
	if labelCol == 0 || lipgloss.Width(option.Description) > textWidth-labelCol {
		return "", false
	}
	pad := strings.Repeat(" ", labelCol-lipgloss.Width(option.Label))
	return option.Label + pad + quiet.Render(option.Description), true
}

// answerView is the free-text answer in brackets, white when it has the focus.
func (d *questionDialog) answerView(width int) string {
	label, bracket := fieldLabelStyle, fgStyle(colStrong)
	if d.onText() {
		label, bracket = labelStyle.Foreground(colFg), keyStyle
	}
	w := maxInt(8, minInt(64, width-14))
	d.text[d.step].SetWidth(w - 1)
	body := lipgloss.NewStyle().Width(w).MaxWidth(w).Render(d.text[d.step].View())
	return label.Render(" answer   ") + bracket.Render("[ ") + body + bracket.Render(" ]")
}

package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

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
		input := textinput.New()
		input.Placeholder = "or type an answer"
		input.CharLimit = 512
		input.Width = 40
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
	key, ok := msg.(tea.KeyMsg)
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
	case " ":
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

func (d *questionDialog) View(width int, caps map[string]int) string {
	inner := modalInner(width)
	optionCap := questionCap(caps, config.BucketQuestionOption)
	descriptionCap := questionCap(caps, config.BucketQuestionDescription)

	question := d.current()
	var b strings.Builder
	if len(d.questions) > 1 {
		b.WriteString(titleStyle.Render(fmt.Sprintf("Question %d of %d", d.step+1, len(d.questions))))
	} else {
		b.WriteString(titleStyle.Render("A question for you"))
	}
	b.WriteString("\n\n")
	b.WriteString(questionTextStyle.Width(inner - 2).Render(question.Question))
	b.WriteString("\n")
	if question.MultiSelect {
		b.WriteString(hintStyle.Render("choose one or more"))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	rowWidth := inner - 4
	textWidth := rowWidth - 4
	for i, option := range question.Options {
		mark := "○"
		if d.chosen[d.step][i] {
			mark = "◉"
		}
		focused := i == d.cursor

		var content []string
		labelLines, labelHidden := capLines(wrapText(option.Label, textWidth), optionCap, focused)
		content = append(content, labelLines...)
		if labelHidden > 0 {
			content = append(content, hintStyle.Render(markerText(labelHidden, false)))
		}
		if option.Description != "" {
			descLines, descHidden := capLines(wrapText(option.Description, textWidth), descriptionCap, focused)
			for _, line := range descLines {
				content = append(content, hintStyle.Render(line))
			}
			if descHidden > 0 {
				content = append(content, hintStyle.Render(markerText(descHidden, false)))
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
				block.WriteString("\n    " + line)
			}
		}
		if focused {
			b.WriteString(selectedRowStyle.Width(rowWidth).Render("▸ " + block.String()))
		} else {
			b.WriteString(rowStyle.Width(rowWidth).Render("  " + block.String()))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(fieldLabelStyle.Render("  "))
	b.WriteString(d.text[d.step].View())
	b.WriteString("\n")

	if d.err != "" {
		b.WriteString("\n" + errorStyle.Render(d.err))
	}
	b.WriteString("\n\n" + hintStyle.Render("↑↓ move · space choose · enter send · esc cancel"))
	return modalStyle.Width(inner).Render(b.String())
}

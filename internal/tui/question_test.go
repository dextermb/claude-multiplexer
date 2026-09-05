package tui

import (
	"strings"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/manager"
	"github.com/dextermb/claude-multiplexer/internal/protocol"
)

func colourQuestion(multi bool) []protocol.Question {
	return []protocol.Question{{
		Question:    "Which colour do you prefer?",
		Header:      "Colour",
		Options:     []protocol.Option{{Label: "Red"}, {Label: "Blue"}},
		MultiSelect: multi,
	}}
}

func TestQuestionSingleSelectFormatsTheAnswer(t *testing.T) {
	d := newQuestionDialog("alpha", colourQuestion(false))
	d.Update(key(" "))
	res, _ := d.Update(key("enter"))
	if res != formSubmitted {
		t.Fatalf("enter with a choice must submit, got %v", res)
	}
	if got := d.answer(); got != "Colour: Red" {
		t.Fatalf("answer = %q, want %q", got, "Colour: Red")
	}
}

func TestQuestionSingleSelectIsExclusive(t *testing.T) {
	d := newQuestionDialog("alpha", colourQuestion(false))
	d.Update(key(" "))
	d.Update(key("down"))
	d.Update(key(" "))
	if got := d.answer(); got != "Colour: Blue" {
		t.Fatalf("answer = %q, want only Blue", got)
	}
}

func TestQuestionMultiSelectKeepsEveryChoice(t *testing.T) {
	d := newQuestionDialog("alpha", colourQuestion(true))
	d.Update(key(" "))
	d.Update(key("down"))
	d.Update(key(" "))
	if got := d.answer(); got != "Colour: Red, Blue" {
		t.Fatalf("answer = %q, want both", got)
	}
}

func TestQuestionFreeTextAnswer(t *testing.T) {
	d := newQuestionDialog("alpha", colourQuestion(false))
	d.Update(key("down"))
	d.Update(key("down"))
	d.Update(key("teal"))
	res, _ := d.Update(key("enter"))
	if res != formSubmitted {
		t.Fatalf("enter with typed text must submit, got %v", res)
	}
	if got := d.answer(); got != "Colour: teal" {
		t.Fatalf("answer = %q, want the typed value", got)
	}
}

func TestQuestionChoiceWithNoteCombines(t *testing.T) {
	d := newQuestionDialog("alpha", colourQuestion(false))
	d.Update(key(" "))
	d.Update(key("down"))
	d.Update(key("down"))
	d.Update(key("warmer"))
	d.Update(key("enter"))
	if got := d.answer(); got != "Colour: Red (warmer)" {
		t.Fatalf("answer = %q, want the label and note", got)
	}
}

func TestQuestionWrapsLongOptionText(t *testing.T) {
	label := "This is a very long option label that must wrap across more than one line and not get cut off"
	desc := "and this description is also long enough that it needs to wrap onto several lines inside the modal"
	q := []protocol.Question{{
		Question: "Pick one",
		Header:   "Pick",
		Options:  []protocol.Option{{Label: label, Description: desc}},
	}}
	d := newQuestionDialog("alpha", q)
	view := visible(d.View(60))
	if strings.Contains(view, "…") {
		t.Fatalf("a long option must wrap, not truncate:\n%s", view)
	}
	flat := strings.Join(strings.Fields(strings.ReplaceAll(view, "│", " ")), " ")
	for _, want := range []string{label, desc} {
		if !strings.Contains(flat, want) {
			t.Fatalf("the view dropped text %q:\n%s", want, view)
		}
	}
}

func TestQuestionEmptyStepWillNotSubmit(t *testing.T) {
	d := newQuestionDialog("alpha", colourQuestion(false))
	res, _ := d.Update(key("enter"))
	if res != formOpen {
		t.Fatalf("enter with no choice must stay open, got %v", res)
	}
	if d.err == "" {
		t.Fatal("an empty step must set an error")
	}
}

func TestQuestionWalksEveryQuestion(t *testing.T) {
	questions := []protocol.Question{
		{Header: "Colour", Options: []protocol.Option{{Label: "Red"}}},
		{Header: "Size", Options: []protocol.Option{{Label: "Large"}}},
	}
	d := newQuestionDialog("alpha", questions)
	d.Update(key(" "))
	if res, _ := d.Update(key("enter")); res != formOpen {
		t.Fatal("the first answer must advance, not submit")
	}
	if d.step != 1 {
		t.Fatalf("step = %d, want the second question", d.step)
	}
	d.Update(key(" "))
	if res, _ := d.Update(key("enter")); res != formSubmitted {
		t.Fatal("the last answer must submit")
	}
	if got := d.answer(); got != "Colour: Red\nSize: Large" {
		t.Fatalf("answer = %q, want a line for each question", got)
	}
}

func TestEventOpensTheQuestionDialogAndSubmitSends(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m, _ = step(t, m, eventMsg(manager.Event{Session: "alpha", Questions: colourQuestion(false)}))
	if m.questions["alpha"] == nil {
		t.Fatal("a question event must open the dialog for its session")
	}
	if view := m.View(); !strings.Contains(visible(view), "Which colour do you prefer?") {
		t.Fatalf("the view does not show the question:\n%s", visible(view))
	}

	m.focus = focusOutput
	m, _ = step(t, m, key(" "))
	m, _ = step(t, m, key("enter"))
	if m.questions["alpha"] != nil {
		t.Fatal("enter must close the dialog")
	}
	if got := m.queued["alpha"]; len(got) != 1 || got[0] != "Colour: Red" {
		t.Fatalf("the answer must be sent, queued = %v", got)
	}
	if m.focus != focusPrompt {
		t.Fatalf("the answer must return the focus to the prompt, focus = %v", m.focus)
	}
}

func TestSecondQuestionWhileOneIsOpenIsNoted(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m, _ = step(t, m, eventMsg(manager.Event{Session: "alpha", Questions: colourQuestion(false)}))
	first := m.questions["alpha"]
	m, _ = step(t, m, eventMsg(manager.Event{Session: "alpha", Questions: colourQuestion(true)}))
	if m.questions["alpha"] != first {
		t.Fatal("a second question must not replace the open one")
	}
	if !strings.Contains(m.status, "another question") {
		t.Fatalf("status = %q, want a note about the waiting question", m.status)
	}
}

func TestQuestionDoesNotStealSelection(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())
	m = spawn(t, m, mgr, "beta", t.TempDir())
	m.sel = "alpha"

	m, _ = step(t, m, eventMsg(manager.Event{Session: "beta", Questions: colourQuestion(false)}))
	if m.sel != "alpha" {
		t.Fatalf("a question must not steal the selection, sel = %q", m.sel)
	}
	if m.questions["beta"] == nil {
		t.Fatal("the question must attach to the session that asked")
	}
	if view := m.View(); strings.Contains(visible(view), "Which colour do you prefer?") {
		t.Fatal("the pane of another session must not show the question")
	}
}

func TestQuestionIgnoresKeysUntilItsPaneHasFocus(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m, _ = step(t, m, eventMsg(manager.Event{Session: "alpha", Questions: colourQuestion(false)}))
	m.focus = focusSidebar
	m, _ = step(t, m, key(" "))
	if got := m.questions["alpha"].chosen[0]; len(got) != 0 {
		t.Fatal("a key must not reach the dialog while the pane is not focused")
	}
}

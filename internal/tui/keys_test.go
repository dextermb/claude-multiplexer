package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func chord(t *testing.T, m Model, target, action string) (Model, tea.Cmd) {
	t.Helper()
	next, _ := step(t, m, key(target))
	if next.seq == nil {
		t.Fatalf("%q must start a sequence", target)
	}
	return step(t, next, key(action))
}

func sequenceModel(t *testing.T) (Model, string) {
	t.Helper()
	m, mgr := newTestModel(t, "")
	m = start(t, m, 160, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())
	m.focus = focusSidebar
	m.prompt.Blur()
	return m, "alpha"
}

func TestATargetKeyWaitsForTheActionKey(t *testing.T) {
	m, _ := sequenceModel(t)

	m, cmd := step(t, m, key("s"))
	if m.seq == nil || m.seq.target != "s" {
		t.Fatalf("seq = %+v, want the s target", m.seq)
	}
	if cmd == nil {
		t.Fatal("a target key must return the timeout command")
	}
	if m.jobsModal != nil || m.form != nil || m.choice != nil {
		t.Fatal("a target key alone must open nothing")
	}
}

func TestTheSequenceOpensTheJobsDialog(t *testing.T) {
	m, name := sequenceModel(t)

	m, _ = chord(t, m, "s", "j")
	if m.seq != nil {
		t.Fatal("the action key must end the sequence")
	}
	if m.jobsModal == nil {
		t.Fatal("s j must open the jobs dialog")
	}
	if !strings.Contains(m.View(), name) {
		t.Fatalf("the dialog does not name the session:\n%s", m.View())
	}
}

func TestTheOldSingleKeysDoNothing(t *testing.T) {
	for _, name := range []string{"J", "M", "e", "p", "z", "Z", "R", "A", "a", "r", "x", "m"} {
		m, _ := sequenceModel(t)
		before := m.View()
		m, _ = step(t, m, key(name))
		if m.jobsModal != nil || m.choice != nil || m.rename != nil || m.confirm != "" {
			t.Errorf("%q must no longer open a dialog", name)
		}
		if got := m.View(); got != before {
			t.Errorf("%q changed the screen:\n%s", name, got)
		}
	}
}

func TestEscCancelsTheSequence(t *testing.T) {
	m, _ := sequenceModel(t)
	m = spawn(t, m, m.mgr, "beta", t.TempDir())
	m.sel = "alpha"

	m, _ = step(t, m, key("s"))
	m, _ = step(t, m, key("esc"))
	if m.seq != nil {
		t.Fatal("esc must cancel the sequence")
	}
	if m.jobsModal != nil {
		t.Fatal("esc must not run an action")
	}

	m, _ = step(t, m, key("j"))
	if m.sel == "alpha" {
		t.Fatal("the next key must move the list again")
	}
}

func TestAnUnknownActionKeyCancelsTheSequence(t *testing.T) {
	m, _ := sequenceModel(t)

	m, _ = step(t, m, key("s"))
	m, _ = step(t, m, key("w"))
	if m.seq != nil {
		t.Fatal("an unknown key must end the sequence")
	}
	if m.status != "no key s w" {
		t.Fatalf("status = %q, want the notice", m.status)
	}
}

func TestTheSequenceTimesOut(t *testing.T) {
	m, _ := sequenceModel(t)

	m, _ = step(t, m, key("s"))
	gen := m.seq.gen
	m, _ = step(t, m, sequenceTimeoutMsg{gen: gen})
	if m.seq != nil {
		t.Fatal("the timeout must clear the target")
	}

	m, _ = step(t, m, key("j"))
	if m.seq != nil {
		t.Fatal("j alone must not start a sequence")
	}
}

func TestAStaleTimeoutLeavesANewerSequence(t *testing.T) {
	m, _ := sequenceModel(t)

	m, _ = step(t, m, key("s"))
	stale := m.seq.gen
	m, _ = step(t, m, key("esc"))
	m, _ = step(t, m, key("l"))
	m, _ = step(t, m, sequenceTimeoutMsg{gen: stale})
	if m.seq == nil || m.seq.target != "l" {
		t.Fatalf("seq = %+v, want the newer l sequence", m.seq)
	}
}

func TestTheControlFormWorksInThePrompt(t *testing.T) {
	m, _ := sequenceModel(t)
	m.focus = focusPrompt
	m.prompt.Focus()
	m.prompt.SetValue("hello")

	m, _ = step(t, m, key("s"))
	if m.seq != nil {
		t.Fatal("a plain s in the prompt must be text")
	}
	if m.prompt.Value() != "hellos" {
		t.Fatalf("prompt = %q, want the s typed into it", m.prompt.Value())
	}

	m, _ = chord(t, m, "ctrl+s", "j")
	if m.jobsModal == nil {
		t.Fatal("ctrl+s j must open the jobs dialog from the prompt")
	}
	if m.prompt.Value() != "hellos" {
		t.Fatalf("prompt = %q, want the text left alone", m.prompt.Value())
	}
}

func TestTheStatusBarShowsTheFollowUpKeys(t *testing.T) {
	m, _ := sequenceModel(t)

	view := visible(m.View())
	for _, want := range []string{"n new", "s session", "l list", "o output"} {
		if !strings.Contains(view, want) {
			t.Errorf("the status bar is missing %q:\n%s", want, view)
		}
	}

	m, _ = step(t, m, key("s"))
	view = visible(m.View())
	for _, want := range []string{"j jobs", "m model", "x stop"} {
		if !strings.Contains(view, want) {
			t.Errorf("the pending bar is missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "? keys") {
		t.Errorf("the pending bar must show the target only:\n%s", view)
	}
}

func TestEveryBindingHasAnAction(t *testing.T) {
	for _, item := range bindings {
		if item.target == "" {
			continue
		}
		if _, ok := sequenceActions[item.keys]; !ok {
			t.Errorf("the key list names %q, which runs nothing", item.keys)
		}
		if item.brief == "" {
			t.Errorf("%q has no short form for the status bar", item.keys)
		}
	}
	for keys := range sequenceActions {
		found := false
		for _, item := range bindings {
			if item.keys == keys {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%q runs, but the key list does not name it", keys)
		}
	}
}

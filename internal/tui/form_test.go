package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dextermb/claude-multiplexer/internal/config"
)

func TestTheFormSelectsOpenOnTheirDefaults(t *testing.T) {
	f := newForm("/tmp", newSessionDefaults{model: "sonnet", mode: "plan", effort: "high", control: true})
	cases := []struct {
		field int
		want  string
	}{
		{fieldModel, "sonnet"},
		{fieldMode, "plan"},
		{fieldEffort, "high"},
		{fieldControl, "yes"},
	}
	for _, tc := range cases {
		if got := f.selects[tc.field].value(); got != tc.want {
			t.Errorf("field %d opens on %q, want %q", tc.field, got, tc.want)
		}
	}
}

func TestADefaultThatIsNotAnOptionFallsBackToTheFirst(t *testing.T) {
	f := newForm("/tmp", newSessionDefaults{model: "gpt", mode: "auto"})
	if got := f.selects[fieldModel].value(); got != "" {
		t.Errorf("an unknown model opens on %q, want the first (default) option", got)
	}
}

func TestADefaultModelSendsNothing(t *testing.T) {
	f := newForm("/tmp", newSessionDefaults{model: "", mode: "auto", effort: ""})
	if f.selects[fieldModel].label() != "default" {
		t.Fatalf("model label = %q, want default", f.selects[fieldModel].label())
	}
	spec := f.spec()
	if spec.Model != "" {
		t.Errorf("spec model = %q, want empty so Claude Code takes its own default", spec.Model)
	}
	if spec.Effort != "" {
		t.Errorf("spec effort = %q, want empty", spec.Effort)
	}
}

func TestTheFormSpecReadsTheSelects(t *testing.T) {
	f := newForm("/tmp", newSessionDefaults{model: "opus", mode: "acceptEdits", effort: "max", control: true})
	spec := f.spec()
	if spec.Model != "opus" || spec.PermissionMode != "acceptEdits" || spec.Effort != "max" || !spec.Control {
		t.Errorf("spec = %+v, want the selected options", spec)
	}
}

func TestLeftAndRightCycleAFocusedSelect(t *testing.T) {
	f := newForm("/tmp", newSessionDefaults{mode: "auto"})
	f.focus = fieldModel
	first := f.selects[fieldModel].value()
	f.Update(tea.KeyMsg{Type: tea.KeyLeft})
	last := f.selects[fieldModel].value()
	if last == first {
		t.Fatalf("left did not cycle: still %q", last)
	}
	if last != modelOptions[len(modelOptions)-1] {
		t.Errorf("left from the first option = %q, want the last %q", last, modelOptions[len(modelOptions)-1])
	}
	f.Update(tea.KeyMsg{Type: tea.KeyRight})
	if got := f.selects[fieldModel].value(); got != first {
		t.Errorf("right did not wrap back: %q, want %q", got, first)
	}
}

func TestTypingDoesNotChangeASelect(t *testing.T) {
	f := newForm("/tmp", newSessionDefaults{mode: "auto"})
	f.focus = fieldEffort
	before := f.selects[fieldEffort].value()
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if got := f.selects[fieldEffort].value(); got != before {
		t.Errorf("a letter changed the select to %q, want %q", got, before)
	}
}

func TestResolveSessionDefaultsPrefersTheFlagThenTheFile(t *testing.T) {
	yes := true
	file := config.Config{
		DefaultModel:          "sonnet",
		DefaultPermissionMode: "plan",
		DefaultEffort:         "high",
		DefaultControl:        &yes,
	}
	got := resolveSessionDefaults(Options{DefaultModel: "opus"}, file)
	if got.model != "opus" {
		t.Errorf("model = %q, want the flag to win", got.model)
	}
	if got.mode != "plan" {
		t.Errorf("mode = %q, want the file value", got.mode)
	}
	if got.effort != "high" {
		t.Errorf("effort = %q, want the file value", got.effort)
	}
	if !got.control {
		t.Error("control = false, want the file value true")
	}
}

func TestResolveSessionDefaultsFallsBackToTheBuiltInMode(t *testing.T) {
	got := resolveSessionDefaults(Options{}, config.Config{})
	if got.mode != "auto" {
		t.Errorf("mode = %q, want the built-in auto", got.mode)
	}
	if got.model != "" || got.effort != "" || got.control {
		t.Errorf("defaults = %+v, want empty model and effort and no control", got)
	}
}

package commands

import (
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/keys"
)

func keymap(t *testing.T) keys.Keymap {
	t.Helper()
	km, _, errs := keys.LoadKeymap(nil)
	if len(errs) > 0 {
		t.Fatalf("LoadKeymap: %v", errs)
	}
	return km
}

func TestResolveBindsASingleKeyAndASequence(t *testing.T) {
	r, errs := Resolve([]config.Command{
		{Keys: "ctrl+b", Label: "gitui", Script: "g.sh"},
		{Keys: "b o", Label: "browser", Script: "o.sh"},
	}, keymap(t))
	if len(errs) > 0 {
		t.Fatalf("Resolve: %v", errs)
	}
	if c, ok := r.Single("ctrl+b"); !ok || c.Label != "gitui" {
		t.Fatalf("Single(ctrl+b) = %+v %v, want the gitui command", c, ok)
	}
	if !r.IsLeader("b") {
		t.Fatal("b is not a leader, want it to start a sequence")
	}
	if c, ok := r.Second("b", "o"); !ok || c.Label != "browser" {
		t.Fatalf("Second(b, o) = %+v %v, want the browser command", c, ok)
	}
	if len(r.List()) != 2 {
		t.Fatalf("List = %d, want 2", len(r.List()))
	}
}

func TestResolveRefusesAReservedKey(t *testing.T) {
	_, errs := Resolve([]config.Command{
		{Keys: "esc o", Label: "x", Script: "x.sh"},
	}, keymap(t))
	if len(errs) != 1 {
		t.Fatalf("errs = %v, want one refusal of a reserved key", errs)
	}
}

func TestResolveRefusesALeaderThatClashesWithABuiltin(t *testing.T) {
	// s is the session target, so it cannot lead a command sequence.
	_, errs := Resolve([]config.Command{
		{Keys: "s o", Label: "x", Script: "x.sh"},
	}, keymap(t))
	if len(errs) != 1 {
		t.Fatalf("errs = %v, want one refusal of the s leader", errs)
	}
}

func TestResolveRefusesASingleKeyThatClashesWithAGlobal(t *testing.T) {
	// q quits, so it cannot be a standalone command.
	_, errs := Resolve([]config.Command{
		{Keys: "q", Label: "x", Script: "x.sh"},
	}, keymap(t))
	if len(errs) != 1 {
		t.Fatalf("errs = %v, want one refusal of the q key", errs)
	}
}

func TestResolveRefusesADuplicateLabelAndTrigger(t *testing.T) {
	_, errs := Resolve([]config.Command{
		{Keys: "ctrl+b", Label: "same", Script: "a.sh"},
		{Keys: "ctrl+y", Label: "same", Script: "b.sh"},
		{Keys: "ctrl+b", Label: "other", Script: "c.sh"},
	}, keymap(t))
	if len(errs) != 2 {
		t.Fatalf("errs = %v, want two refusals (a duplicate label and a duplicate trigger)", errs)
	}
}

func TestResolveRefusesAnUnknownScriptType(t *testing.T) {
	_, errs := Resolve([]config.Command{
		{Keys: "ctrl+b", Label: "x", Script: "x.rb"},
	}, keymap(t))
	if len(errs) != 1 {
		t.Fatalf("errs = %v, want one refusal of the unknown extension", errs)
	}
}

func TestResolveKeepsTheGoodWhenOneIsBad(t *testing.T) {
	r, errs := Resolve([]config.Command{
		{Keys: "q", Label: "bad", Script: "x.sh"},
		{Keys: "b o", Label: "good", Script: "o.sh"},
	}, keymap(t))
	if len(errs) != 1 {
		t.Fatalf("errs = %v, want one refusal", errs)
	}
	if _, ok := r.Second("b", "o"); !ok {
		t.Fatal("the good command was lost, want it kept")
	}
}

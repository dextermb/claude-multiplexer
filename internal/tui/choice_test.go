package tui

import "testing"

func TestChoiceStartsAtTheCurrentValue(t *testing.T) {
	d := newChoiceDialog(settingMode, "alpha", "plan")
	if d.chosen() != "plan" {
		t.Fatalf("the cursor must start on the current value, got %q", d.chosen())
	}
}

func TestChoiceMovesAndApplies(t *testing.T) {
	d := newChoiceDialog(settingEffort, "alpha", "low")
	d.Update(key("down"))
	res, _ := d.Update(key("enter"))
	if res != formSubmitted {
		t.Fatalf("enter must submit, got %v", res)
	}
	if d.chosen() != "medium" {
		t.Fatalf("chosen = %q, want the next level medium", d.chosen())
	}
}

func TestChoiceWrapsAtTheTop(t *testing.T) {
	d := newChoiceDialog(settingModel, "alpha", modelChoices[0])
	d.Update(key("up"))
	if d.chosen() != modelChoices[len(modelChoices)-1] {
		t.Fatalf("up from the first must wrap to the last, got %q", d.chosen())
	}
}

func TestChoiceEscCancels(t *testing.T) {
	d := newChoiceDialog(settingModel, "alpha", "sonnet")
	res, _ := d.Update(key("esc"))
	if res != formCancelled {
		t.Fatalf("esc must cancel, got %v", res)
	}
}

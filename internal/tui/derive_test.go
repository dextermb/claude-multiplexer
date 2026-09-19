package tui

import "testing"

// TestFollowSelectionDerivesTheDiffAndTaskScroll checks the selection-follow
// derivation directly: a moved selection resets the task scroll and returns a
// diff refresh for the new session, and a still selection derives nothing.
func TestFollowSelectionDerivesTheDiffAndTaskScroll(t *testing.T) {
	m := Model{
		sel:        "b",
		taskScroll: 7,
		rows:       []row{{name: "a"}, {name: "b"}},
	}

	cmd := m.followSelection("a")
	if cmd == nil {
		t.Fatal("a moved selection must return a diff refresh command")
	}
	if m.taskScroll != 0 {
		t.Fatalf("a moved selection must reset the task scroll, got %d", m.taskScroll)
	}
	msg, ok := cmd().(diffMsg)
	if !ok {
		t.Fatalf("the command must produce a diffMsg, got %T", cmd())
	}
	if msg.name != "b" {
		t.Fatalf("the diff must refresh the new selection, got %q", msg.name)
	}

	m.taskScroll = 7
	if cmd := m.followSelection("b"); cmd != nil {
		t.Fatal("a still selection must return no command")
	}
	if m.taskScroll != 7 {
		t.Fatalf("a still selection must leave the task scroll, got %d", m.taskScroll)
	}
}

// TestClampTaskFocusRetreatsWhenTheDiffPanelOpens checks the focus derivation
// directly: focusTask holds only while the diff panel is closed, so opening the
// diff panel derives the focus back to the output.
func TestClampTaskFocusRetreatsWhenTheDiffPanelOpens(t *testing.T) {
	m := Model{focus: focusTask, diffPanel: true}
	m.clampTaskFocus()
	if m.focus != focusOutput {
		t.Fatalf("an open diff panel must derive the focus to the output, focus = %d", m.focus)
	}

	m = Model{focus: focusDiff, diffPanel: true}
	m.clampTaskFocus()
	if m.focus != focusDiff {
		t.Fatalf("the clamp must touch only focusTask, focus = %d", m.focus)
	}
}

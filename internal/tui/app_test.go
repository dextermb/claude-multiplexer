package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/dextermb/claude-multiplexer/internal/manager"
	"github.com/dextermb/claude-multiplexer/internal/render"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

var fakeClaude string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "fakeclaude")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fakeClaude = filepath.Join(dir, "fakeclaude")
	build := exec.Command("go", "build", "-o", fakeClaude,
		"github.com/dextermb/claude-multiplexer/internal/testutil/fakeclaude")
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build fakeclaude: %v\n%s", err, out)
		os.RemoveAll(dir)
		os.Exit(1)
	}
	lipgloss.SetColorProfile(termenv.ANSI256)

	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func newTestModel(t *testing.T, initialDir string) (Model, *manager.Manager) {
	t.Helper()
	mgr, err := manager.New(manager.Options{
		Root:       t.TempDir(),
		ClaudePath: fakeClaude,
		Renderer:   render.Renderer{},
	})
	if err != nil {
		t.Fatalf("manager.New: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		mgr.Shutdown(ctx)
	})
	m := New(Options{Manager: mgr, DefaultDir: t.TempDir(), InitialDir: initialDir})
	return m, mgr
}

func step(t *testing.T, m Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(msg)
	model, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T", next)
	}
	return model, cmd
}

func key(name string) tea.KeyMsg {
	switch name {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "ctrl+s":
		return tea.KeyMsg{Type: tea.KeyCtrlS}
	case "ctrl+l":
		return tea.KeyMsg{Type: tea.KeyCtrlL}
	case "ctrl+o":
		return tea.KeyMsg{Type: tea.KeyCtrlO}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(name)}
}

var ansiCodes = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func visible(text string) string {
	return ansiCodes.ReplaceAllString(text, "")
}

func start(t *testing.T, m Model, width, height int) Model {
	t.Helper()
	m, _ = step(t, m, tea.WindowSizeMsg{Width: width, Height: height})
	m, _ = step(t, m, storedMsg{metas: m.mgr.Stored()})
	return m
}

func spawn(t *testing.T, m Model, mgr *manager.Manager, name, dir string) Model {
	t.Helper()
	sessionName, err := mgr.Spawn(context.Background(), manager.Spec{Name: name, Dir: dir})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	m, _ = step(t, m, spawnedMsg{name: sessionName})
	return m
}

func TestTheFormOpensWhenThereAreNoSessions(t *testing.T) {
	m, _ := newTestModel(t, "")
	m = start(t, m, 100, 30)
	if m.form == nil {
		t.Fatal("the new session form must open when the list is empty")
	}
	if view := m.View(); !strings.Contains(view, "New session") {
		t.Fatalf("view does not show the form:\n%s", view)
	}
	m, _ = step(t, m, key("esc"))
	if m.form != nil {
		t.Fatal("esc must close the form")
	}
	if view := m.View(); !strings.Contains(view, "No sessions yet") {
		t.Fatalf("view does not show the empty state:\n%s", view)
	}
}

func TestTheFormRejectsADirectoryThatDoesNotExist(t *testing.T) {
	m, _ := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m.form.inputs[fieldDir].SetValue(filepath.Join(t.TempDir(), "nope"))
	m, _ = step(t, m, key("enter"))
	if m.form == nil {
		t.Fatal("the form must stay open when the directory is wrong")
	}
	if !strings.Contains(m.View(), "no such directory") {
		t.Fatalf("view does not show the reason:\n%s", m.View())
	}
}

func TestTheSidebarListsEverySessionAndMarksTheSelectedOne(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m, _ = step(t, m, key("esc"))

	dir := t.TempDir()
	m = spawn(t, m, mgr, "alpha", dir)
	m = spawn(t, m, mgr, "beta", dir)

	view := m.View()
	for _, want := range []string{"alpha", "beta"} {
		if !strings.Contains(view, want) {
			t.Errorf("view has no %q:\n%s", want, view)
		}
	}
	if m.sel != "beta" {
		t.Fatalf("selected = %q, want the newest session", m.sel)
	}
}

func TestAClickInTheSidebarSelectsASession(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m, _ = step(t, m, key("esc"))

	dir := t.TempDir()
	m = spawn(t, m, mgr, "alpha", dir)
	m = spawn(t, m, mgr, "beta", dir)

	// The first line of the sidebar is the group header, so the rows follow it.
	m, _ = step(t, m, tea.MouseMsg{
		X: 3, Y: titleHeight + 1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
	})
	if m.sel != "alpha" {
		t.Fatalf("selected = %q, want alpha", m.sel)
	}
	if m.focus != focusSidebar {
		t.Fatal("a click in the sidebar must move the focus there")
	}

	m, _ = step(t, m, tea.MouseMsg{
		X: 3, Y: titleHeight + 2, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
	})
	if m.sel != "beta" {
		t.Fatalf("selected = %q, want beta", m.sel)
	}
}

func TestAClickBelowTheListChangesNothing(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m, _ = step(t, m, tea.MouseMsg{
		X: 3, Y: titleHeight + 5, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
	})
	if m.sel != "alpha" {
		t.Fatalf("selected = %q, want alpha", m.sel)
	}
}

func TestTheKeysMoveThroughTheList(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m, _ = step(t, m, key("esc"))

	dir := t.TempDir()
	m = spawn(t, m, mgr, "alpha", dir)
	m = spawn(t, m, mgr, "beta", dir)
	m = spawn(t, m, mgr, "gamma", dir)

	m.focus = focusSidebar
	m.prompt.Blur()
	m, _ = step(t, m, key("k"))
	if m.sel != "beta" {
		t.Fatalf("after k, selected = %q, want beta", m.sel)
	}
	m, _ = step(t, m, key("up"))
	if m.sel != "alpha" {
		t.Fatalf("after up, selected = %q, want alpha", m.sel)
	}
	m, _ = step(t, m, key("up"))
	if m.sel != "alpha" {
		t.Fatalf("the list must stop at the top, got %q", m.sel)
	}
	m, _ = step(t, m, key("j"))
	if m.sel != "beta" {
		t.Fatalf("after j, selected = %q, want beta", m.sel)
	}
}

func fill(m *Model, count int) {
	lines := make([]render.Line, 0, count)
	for i := 0; i < count; i++ {
		lines = append(lines, render.Line{Class: render.ClassText, Text: fmt.Sprintf("line %d", i)})
	}
	m.appendOutput(lines)
}

func TestTabCyclesTheThreePanes(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.focus = focusSidebar
	m.prompt.Blur()
	for _, want := range []focusArea{focusPrompt, focusOutput, focusSidebar} {
		m, _ = step(t, m, key("tab"))
		if m.focus != want {
			t.Fatalf("focus = %v, want %v", m.focus, want)
		}
	}
}

func TestTheOutputPaneScrollsWithTheKeys(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())
	fill(&m, 200)

	m.focus = focusOutput
	m.prompt.Blur()
	if !m.output.AtBottom() {
		t.Fatal("the pane must start at the bottom")
	}
	if m.scrollIndicator() != "" {
		t.Fatalf("no indicator belongs at the bottom, got %q", m.scrollIndicator())
	}

	bottom := m.output.YOffset
	m, _ = step(t, m, key("k"))
	if m.output.YOffset != bottom-1 {
		t.Fatalf("k moved the offset to %d, want %d", m.output.YOffset, bottom-1)
	}
	if m.scrollIndicator() == "" {
		t.Fatal("the bar must show the position once you scroll up")
	}
	if !strings.Contains(m.barView(), "↑") {
		t.Fatalf("the bar hides the scroll position:\n%s", m.barView())
	}

	m, _ = step(t, m, key("g"))
	if !m.output.AtTop() {
		t.Fatalf("g must go to the top, offset %d", m.output.YOffset)
	}
	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	if !m.output.AtBottom() {
		t.Fatal("G must go back to the bottom")
	}
	m, _ = step(t, m, key("u"))
	if m.output.AtBottom() {
		t.Fatal("u must scroll up half a pane")
	}
	m, _ = step(t, m, key("d"))
	if !m.output.AtBottom() {
		t.Fatal("d must scroll back down half a pane")
	}
}

func TestNewOutputDoesNotStealYourPlace(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())
	fill(&m, 200)

	m.output.GotoTop()
	before := m.output.YOffset
	m.lastSeq = 5
	m, _ = step(t, m, eventMsg(manager.Event{
		Seq:     6,
		Session: m.sel,
		Lines:   []render.Line{{Class: render.ClassText, Text: "something new"}},
	}))
	if m.output.YOffset != before {
		t.Fatalf("the pane jumped from %d to %d", before, m.output.YOffset)
	}

	m.output.GotoBottom()
	m, _ = step(t, m, eventMsg(manager.Event{
		Seq:     7,
		Session: m.sel,
		Lines:   []render.Line{{Class: render.ClassText, Text: "newer still"}},
	}))
	if !m.output.AtBottom() {
		t.Fatal("the pane must keep following when you are at the bottom")
	}
}

func TestAClickInTheOutputFocusesIt(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m, _ = step(t, m, tea.MouseMsg{
		X: sidebarWidth + 5, Y: 3, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
	})
	if m.focus != focusOutput {
		t.Fatalf("focus = %v, want the output", m.focus)
	}
}

func TestThePromptSendsToTheSelectedSession(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())
	if m.focus != focusPrompt {
		t.Fatal("a new session must put the focus in the prompt")
	}

	m.focus = focusSidebar
	m.prompt.Blur()
	m, _ = step(t, m, key("tab"))
	if m.focus != focusPrompt {
		t.Fatal("tab must move the focus to the prompt")
	}
	for _, r := range "hello" {
		m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m, _ = step(t, m, key("enter"))
	if m.prompt.Value() != "" {
		t.Fatalf("the prompt was not cleared, it holds %q", m.prompt.Value())
	}

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		snap, err := mgr.Snapshot("alpha")
		if err == nil && snap.Turns >= 1 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("the session never finished the turn")
}

func TestSendWithoutASessionReportsTheReason(t *testing.T) {
	m, _ := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m, _ = step(t, m, key("esc"))

	m.focus = focusPrompt
	m.prompt.SetValue("hello")
	m, _ = step(t, m, key("enter"))
	if m.errText != "no session is selected" {
		t.Fatalf("errText = %q", m.errText)
	}
	if !strings.Contains(m.View(), "no session is selected") {
		t.Fatalf("the status bar hides the error:\n%s", m.View())
	}
}

func TestAGapInTheSequenceRebuildsTheOutput(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	if err := mgr.Send("alpha", "hello"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if snap, err := mgr.Snapshot("alpha"); err == nil && snap.Turns >= 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	m.lastSeq = 1
	m, _ = step(t, m, eventMsg(manager.Event{Seq: 99, Session: "alpha"}))
	if !strings.Contains(visible(m.outputText), "echo: hello") {
		t.Fatalf("the output was not rebuilt after a gap:\n%s", m.outputText)
	}
}

func TestStopAsksForConfirmationFirst(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 30)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m.focus = focusSidebar
	m.prompt.Blur()
	m, _ = chord(t, m, "s", "x")
	if m.confirm != "alpha" {
		t.Fatalf("confirm = %q, want alpha", m.confirm)
	}
	if !strings.Contains(m.View(), "Stop session") {
		t.Fatalf("the confirmation is not shown:\n%s", m.View())
	}
	m, _ = step(t, m, key("n"))
	if m.confirm != "" {
		t.Fatal("any other key must cancel the confirmation")
	}

	m, _ = chord(t, m, "s", "x")
	m, cmd := step(t, m, key("y"))
	if m.confirm != "" {
		t.Fatal("y must clear the confirmation")
	}
	if cmd == nil {
		t.Fatal("y must return a stop command")
	}
	msg, ok := cmd().(stoppedMsg)
	if !ok || msg.err != nil {
		t.Fatalf("stop message = %+v", msg)
	}
	snap, err := mgr.Snapshot("alpha")
	if err != nil || snap.State.Live() {
		t.Fatalf("the session is still live: %+v, %v", snap, err)
	}
}

func TestTheViewFillsTheWindowExactly(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 96, 22)
	m, _ = step(t, m, key("esc"))

	dir := t.TempDir()
	m = spawn(t, m, mgr, "alpha", dir)
	m = spawn(t, m, mgr, "beta", dir)

	for _, state := range []struct {
		name string
		view string
	}{
		{"normal", m.View()},
	} {
		lines := strings.Split(state.view, "\n")
		if len(lines) != m.height {
			t.Fatalf("%s: the view has %d lines, want %d", state.name, len(lines), m.height)
		}
		for i, line := range lines {
			if width := lipgloss.Width(line); width > m.width {
				t.Fatalf("%s: line %d is %d wide, want at most %d", state.name, i, width, m.width)
			}
		}
		for i := 0; i < m.bodyHeight(); i++ {
			if width := lipgloss.Width(lines[i]); width != m.width {
				t.Errorf("%s: body line %d is %d wide, want exactly %d", state.name, i, width, m.width)
			}
		}
	}
}

func TestTheSessionBarShowsTheNumbers(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 110, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	if err := mgr.Send("alpha", "hello"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if snap, err := mgr.Snapshot("alpha"); err == nil && snap.Turns >= 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	m.refresh()

	bar := m.barView()
	for _, want := range []string{"alpha", "fake-model", "auto", "idle", "$0.2500"} {
		if !strings.Contains(bar, want) {
			t.Errorf("the bar has no %q:\n%s", want, bar)
		}
	}
	for _, gone := range []string{"turn", "ms"} {
		if strings.Contains(bar, gone) {
			t.Errorf("the bar still shows %q:\n%s", gone, bar)
		}
	}
	if lipgloss.Width(bar) != m.outputWidth() {
		t.Errorf("the bar is %d wide, want %d", lipgloss.Width(bar), m.outputWidth())
	}
	if strings.Contains(bar, "\n") {
		t.Error("the bar must stay on one line")
	}
}

func TestTheSessionBarDropsDetailsBeforeTheName(t *testing.T) {
	widths := []int{140, 74, 60, 44}
	for _, width := range widths {
		m, mgr := newTestModel(t, "")
		m, _ = step(t, m, tea.WindowSizeMsg{Width: width, Height: 24})
		m, _ = step(t, m, key("esc"))
		m = spawn(t, m, mgr, "alpha", t.TempDir())
		if err := mgr.Send("alpha", "hello"); err != nil {
			t.Fatalf("Send: %v", err)
		}
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			if snap, err := mgr.Snapshot("alpha"); err == nil && snap.Turns >= 1 {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		m.refresh()

		bar := m.barView()
		if lipgloss.Width(bar) != m.outputWidth() {
			t.Errorf("width %d: the bar is %d wide, want %d:\n%s",
				width, lipgloss.Width(bar), m.outputWidth(), bar)
		}
		if strings.Contains(bar, "\n") {
			t.Errorf("width %d: the bar must stay on one line", width)
		}
		if !strings.Contains(bar, "alpha") {
			t.Errorf("width %d: the name must survive:\n%s", width, bar)
		}
		if width == 140 {
			if !strings.Contains(bar, "fake-model") {
				t.Errorf("a wide bar must name the model:\n%s", bar)
			}
			if !strings.Contains(bar, "$0.2500") {
				t.Errorf("a wide bar must show the cost:\n%s", bar)
			}
		}
		if width == 44 && strings.Contains(bar, "$0.2500") {
			t.Errorf("a narrow bar must drop the cost first:\n%s", bar)
		}
	}
}

func TestTheBarLadderShedsBothSidesInTurn(t *testing.T) {
	got := barLadder(3, 6)
	want := [][2]int{{0, 0}, {1, 0}, {1, 1}, {2, 1}, {2, 2}, {2, 3}, {2, 4}, {2, 5}}
	if len(got) != len(want) {
		t.Fatalf("ladder = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ladder = %v, want %v", got, want)
		}
	}
	if last := barLadder(2, 2); last[len(last)-1] != [2]int{1, 1} {
		t.Fatalf("the ladder must end at the smallest pair, got %v", last)
	}
}

func TestTheSessionBarSaysWhenThereIsNoSession(t *testing.T) {
	m, _ := newTestModel(t, "")
	m = start(t, m, 90, 24)
	m, _ = step(t, m, key("esc"))
	if !strings.Contains(m.barView(), "no session") {
		t.Fatalf("bar = %q", m.barView())
	}
}

func TestMetaLinesAreMutedAndTextIsNot(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 90, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	meta := m.wrap([]render.Line{{Class: render.ClassMeta, Text: "● init"}})
	text := m.wrap([]render.Line{{Class: render.ClassText, Text: "● init"}})
	tool := m.wrap([]render.Line{{Class: render.ClassToolUse, Text: "● init"}})
	if meta == text {
		t.Fatal("a meta line must not render like assistant text")
	}
	if tool == text || tool == meta {
		t.Fatal("a tool line must have its own colour")
	}
	if !strings.Contains(meta, "\x1b[") {
		t.Fatalf("the meta line carries no colour: %q", meta)
	}
}

func TestTheStatusBarCountsTheSessions(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 120, 30)
	m, _ = step(t, m, key("esc"))
	dir := t.TempDir()
	m = spawn(t, m, mgr, "alpha", dir)
	m = spawn(t, m, mgr, "beta", dir)

	status := m.statusView()
	for _, want := range []string{"2 sessions", "$0.0000", "q quit"} {
		if !strings.Contains(status, want) {
			t.Errorf("status %q has no %q", status, want)
		}
	}
	if strings.Contains(status, "busy") {
		t.Errorf("status %q shows a busy count when no session is busy", status)
	}

	for i := range m.rows {
		if m.rows[i].live {
			m.rows[i].state = session.StateBusy
			break
		}
	}
	if busy := m.statusView(); !strings.Contains(busy, "1 busy") {
		t.Errorf("status %q has no %q with one busy session", busy, "1 busy")
	}
}

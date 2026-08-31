package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dextermb/claude-multiplexer/internal/manager"
	"github.com/dextermb/claude-multiplexer/internal/render"
)

func waitUntil(t *testing.T, limit time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("timed out")
}

func storeSession(t *testing.T, mgr *manager.Manager, name, dir, prompt string) {
	t.Helper()
	spawned, err := mgr.Spawn(context.Background(), manager.Spec{Name: name, Dir: dir})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if err := mgr.Send(spawned, prompt); err != nil {
		t.Fatalf("Send: %v", err)
	}
	waitUntil(t, 10*time.Second, func() bool {
		snap, err := mgr.Snapshot(spawned)
		return err == nil && snap.Turns >= 1
	})
	waitUntil(t, 10*time.Second, func() bool {
		for _, meta := range append(mgr.Stored(), managerMeta(mgr, spawned)...) {
			if meta.Name == spawned && meta.Turns >= 1 {
				return true
			}
		}
		return false
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := mgr.Stop(ctx, spawned); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	waitUntil(t, 10*time.Second, func() bool { return mgr.Remove(spawned) == nil })
	waitUntil(t, 10*time.Second, func() bool {
		for _, meta := range mgr.Stored() {
			if meta.Name == spawned {
				return true
			}
		}
		return false
	})
}

func managerMeta(mgr *manager.Manager, name string) []manager.Meta {
	meta, err := manager.ReadMeta(mgr.Root() + "/sessions/" + name + "/meta.json")
	if err != nil {
		return nil
	}
	return []manager.Meta{meta}
}

func TestStoredSessionsFillTheSidebar(t *testing.T) {
	m, mgr := newTestModel(t, "")
	dir := t.TempDir()
	storeSession(t, mgr, "yesterday", dir, "what did we do")
	m = start(t, m, 100, 24)

	view := m.View()
	if !strings.Contains(view, "yesterday") {
		t.Fatalf("the stored session is missing from the sidebar:\n%s", view)
	}
	if g := rowGlyph(m.rows[0], 0); g != "○" {
		t.Fatalf("the stored glyph is %q, want ○", g)
	}
	if m.form != nil {
		t.Fatal("the form must not open when there is stored work to show")
	}
	if m.sel != "yesterday" {
		t.Fatalf("selected = %q", m.sel)
	}
	if !strings.Contains(visible(m.outputText), "› what did we do") {
		t.Fatalf("the past prompt was not replayed:\n%s", m.outputText)
	}
	if !strings.Contains(visible(m.outputText), "echo: what did we do") {
		t.Fatalf("the past reply was not replayed:\n%s", m.outputText)
	}
}

func TestAStoredSessionRefusesAPromptAndOffersToResume(t *testing.T) {
	m, mgr := newTestModel(t, "")
	storeSession(t, mgr, "old", t.TempDir(), "hello")
	m = start(t, m, 100, 24)

	m.focus = focusPrompt
	m.prompt.SetValue("carry on")
	m, _ = step(t, m, key("enter"))
	if !strings.Contains(m.errText, "not running") {
		t.Fatalf("errText = %q", m.errText)
	}
	if m.prompt.Value() != "carry on" {
		t.Fatal("the prompt must keep what you typed")
	}
}

func TestEnterResumesAStoredSession(t *testing.T) {
	m, mgr := newTestModel(t, "")
	storeSession(t, mgr, "old", t.TempDir(), "hello")
	m = start(t, m, 100, 24)

	m.focus = focusSidebar
	m.prompt.Blur()
	m, cmd := step(t, m, key("enter"))
	if cmd == nil {
		t.Fatal("enter on a stored session must return a resume command")
	}
	msg, ok := cmd().(spawnedMsg)
	if !ok {
		t.Fatalf("msg = %T", msg)
	}
	if msg.err != nil || msg.name != "old" {
		t.Fatalf("resume gave %+v", msg)
	}

	m, _ = step(t, m, msg)
	m, _ = step(t, m, storedMsg{metas: mgr.Stored()})
	item, ok := m.selectedRow()
	if !ok || !item.live {
		t.Fatalf("the resumed session is not live: %+v", item)
	}
	if !strings.Contains(m.View(), "old") {
		t.Fatal("the resumed session left the sidebar")
	}
}

func TestArchivingHidesASessionAndAShowsItAgain(t *testing.T) {
	m, mgr := newTestModel(t, "")
	dir := t.TempDir()
	storeSession(t, mgr, "done", dir, "hello")
	m = start(t, m, 100, 24)

	m.focus = focusSidebar
	m.prompt.Blur()
	m, cmd := step(t, m, key("a"))
	if cmd == nil {
		t.Fatal("a must return an archive command")
	}
	m, _ = step(t, m, cmd())
	m, _ = step(t, m, storedMsg{metas: mgr.Stored()})

	if len(m.rows) != 0 {
		t.Fatalf("rows = %+v, want none after archiving", m.rows)
	}
	if !strings.Contains(m.View(), "Press A to show them") {
		t.Fatalf("the empty state does not mention the archive:\n%s", m.View())
	}

	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("A")})
	if len(m.rows) != 1 || !m.rows[0].archived {
		t.Fatalf("rows = %+v, want the archived session back", m.rows)
	}
	if g := rowGlyph(m.rows[0], 0); g != "·" {
		t.Fatalf("the archived glyph is %q, want ·", g)
	}

	m, cmd = step(t, m, key("a"))
	if cmd == nil {
		t.Fatal("a must restore an archived session")
	}
	m, _ = step(t, m, cmd())
	m, _ = step(t, m, storedMsg{metas: mgr.Stored()})
	if len(m.rows) != 1 || m.rows[0].archived {
		t.Fatalf("rows = %+v, want the session restored", m.rows)
	}
}

func TestARunningSessionCannotBeArchived(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "live", t.TempDir())

	m.focus = focusSidebar
	m.prompt.Blur()
	m, cmd := step(t, m, key("a"))
	if cmd != nil {
		t.Fatal("a live session must not be archived")
	}
	if !strings.Contains(m.errText, "stop the session") {
		t.Fatalf("errText = %q", m.errText)
	}
}

func TestStoppingASessionKeepsItInTheList(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "brief", t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := mgr.Stop(ctx, "brief"); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	m.refresh()
	item, ok := m.selectedRow()
	if !ok || !item.live || item.running() {
		t.Fatalf("a stopped session must stay in the list as exited: %+v", item)
	}
}

func TestAnExitedSessionResumesAndKeepsItsHistory(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "work", t.TempDir())

	if err := mgr.Send("work", "the first question"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	waitUntil(t, 10*time.Second, func() bool {
		snap, err := mgr.Snapshot("work")
		return err == nil && snap.Turns >= 1
	})
	waitUntil(t, 10*time.Second, func() bool {
		meta, err := mgr.Meta("work")
		return err == nil && meta.ClaudeSessionID != ""
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := mgr.Stop(ctx, "work"); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	m.refresh()

	if !strings.Contains(m.promptView(), "press Enter to resume") {
		t.Fatalf("the prompt does not offer a resume:\n%s", m.promptView())
	}

	m.focus = focusSidebar
	m.prompt.Blur()
	m, cmd := step(t, m, key("enter"))
	if cmd == nil {
		t.Fatal("enter on an exited session must resume it")
	}
	msg, ok := cmd().(spawnedMsg)
	if !ok || msg.err != nil {
		t.Fatalf("resume gave %+v", msg)
	}
	if msg.name != "work" {
		t.Fatalf("resumed as %q, want the same name", msg.name)
	}

	m, _ = step(t, m, msg)
	if !strings.Contains(visible(m.outputText), "the first question") {
		t.Fatalf("the resumed session lost its history:\n%s", m.outputText)
	}
	if !strings.Contains(visible(m.outputText), "— resumed —") {
		t.Fatalf("the resume is not marked:\n%s", m.outputText)
	}
	item, _ := m.selectedRow()
	if !item.running() {
		t.Fatalf("the session is not running: %+v", item)
	}
}

func TestTheTextStreamsIntoThePaneAndSettles(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m, _ = step(t, m, eventMsg(manager.Event{
		Seq: m.lastSeq + 1, Session: "alpha", Partial: "The loader has",
	}))
	if got := visible(m.output.View()); !strings.Contains(got, "The loader has") {
		t.Fatalf("the first piece is not shown:\n%s", got)
	}
	if !strings.Contains(visible(m.output.View()), cursorMark) {
		t.Error("a streaming line must carry the cursor")
	}

	m, _ = step(t, m, eventMsg(manager.Event{
		Seq: m.lastSeq + 1, Session: "alpha", Partial: "The loader has three problems",
	}))
	view := visible(m.output.View())
	if strings.Count(view, "The loader has") != 1 {
		t.Fatalf("the pieces were repeated instead of replaced:\n%s", view)
	}
	if !strings.Contains(view, "three problems") {
		t.Fatalf("the second piece is missing:\n%s", view)
	}

	m, _ = step(t, m, eventMsg(manager.Event{
		Seq:     m.lastSeq + 1,
		Session: "alpha",
		Lines:   []render.Line{{Class: render.ClassText, Text: "The loader has three problems."}},
	}))
	view = visible(m.output.View())
	if strings.Contains(view, cursorMark) {
		t.Errorf("the cursor must go when the message lands:\n%s", view)
	}
	if got := strings.Count(view, "The loader has three problems"); got != 1 {
		t.Fatalf("the settled text appears %d times, want once:\n%s", got, view)
	}
}

func TestAPartialBelongsToItsOwnSession(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	dir := t.TempDir()
	m = spawn(t, m, mgr, "alpha", dir)
	m = spawn(t, m, mgr, "beta", dir)

	m, _ = step(t, m, eventMsg(manager.Event{
		Seq: m.lastSeq + 1, Session: "alpha", Partial: "words for alpha",
	}))
	if strings.Contains(visible(m.output.View()), "words for alpha") {
		t.Fatal("beta is selected, so alpha's stream must not show")
	}

	m.sel = "alpha"
	m.rebuildOutput()
	if !strings.Contains(visible(m.output.View()), "words for alpha") {
		t.Fatal("selecting alpha must show its stream")
	}
}

func TestADroppedPartialSelfHeals(t *testing.T) {
	m, mgr := newTestModel(t, "")
	m = start(t, m, 100, 24)
	m, _ = step(t, m, key("esc"))
	m = spawn(t, m, mgr, "alpha", t.TempDir())

	m, _ = step(t, m, eventMsg(manager.Event{Seq: 10, Session: "alpha", Partial: "one two"}))
	m, _ = step(t, m, eventMsg(manager.Event{Seq: 40, Session: "alpha", Partial: "one two three four"}))

	view := visible(m.output.View())
	if !strings.Contains(view, "one two three four") {
		t.Fatalf("the newest partial must survive a gap:\n%s", view)
	}
	if strings.Count(view, "one two") != 1 {
		t.Fatalf("the gap duplicated the text:\n%s", view)
	}
}

package manager

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/render"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

func governorManager(t *testing.T, warn, act int, action string) *Manager {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), config.FileName)
	cfg := config.Config{ContextAction: action}
	if warn > 0 {
		cfg.ContextWarnPercent = &warn
	}
	if act > 0 {
		cfg.ContextActPercent = &act
	}
	if err := config.Write(configPath, cfg); err != nil {
		t.Fatalf("write config: %v", err)
	}
	m, err := New(Options{
		Root:        t.TempDir(),
		ClaudePath:  fakeClaude,
		ConfigPaths: []string{configPath},
		Renderer:    render.Renderer{},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return m
}

// fill returns a snapshot whose context is the given percent of the 200k window
// of a Claude model.
func fill(name string, percent int) session.Snapshot {
	return session.Snapshot{
		Name:          name,
		Model:         "claude-sonnet-4-6",
		ContextTokens: 200_000 * percent / 100,
	}
}

func drainNotices(sub *Subscription) []string {
	var out []string
	for {
		select {
		case ev := <-sub.C:
			if ev.Notice != "" {
				out = append(out, ev.Notice)
			}
		default:
			return out
		}
	}
}

func TestTheGovernorWarnsOncePerCrossing(t *testing.T) {
	m := governorManager(t, 60, 0, ContextActionNotify)
	sub := m.Subscribe(16)
	defer sub.Close()
	item := &entry{}

	m.maybeContextNotice(item, fill("alpha", 50))
	if got := drainNotices(sub); len(got) != 0 {
		t.Fatalf("notices below the threshold = %v, want none", got)
	}

	m.maybeContextNotice(item, fill("alpha", 62))
	first := drainNotices(sub)
	if len(first) != 1 || !strings.Contains(first[0], "62%") {
		t.Fatalf("notices = %v, want one naming 62%%", first)
	}

	m.maybeContextNotice(item, fill("alpha", 70))
	if got := drainNotices(sub); len(got) != 0 {
		t.Fatalf("notices after the first = %v, want none", got)
	}
}

func TestTheGovernorHoldsAtTheActThreshold(t *testing.T) {
	m := governorManager(t, 60, 80, ContextActionHold)
	sub := m.Subscribe(16)
	defer sub.Close()
	item := &entry{}

	m.maybeContextNotice(item, fill("alpha", 85))
	notices := drainNotices(sub)
	if len(notices) != 1 || !strings.Contains(notices[0], "held") {
		t.Fatalf("notices = %v, want one that says held", notices)
	}
	item.ctxMu.Lock()
	held := item.ctxHeld
	item.ctxMu.Unlock()
	if !held {
		t.Fatal("the governor did not hold the session")
	}
}

func TestTheNotifyActionNeverHolds(t *testing.T) {
	m := governorManager(t, 60, 80, ContextActionNotify)
	sub := m.Subscribe(16)
	defer sub.Close()
	item := &entry{}

	m.maybeContextNotice(item, fill("alpha", 90))
	if got := drainNotices(sub); len(got) != 1 {
		t.Fatalf("notices = %v, want one", got)
	}
	item.ctxMu.Lock()
	held := item.ctxHeld
	item.ctxMu.Unlock()
	if held {
		t.Fatal("the notify action held the session")
	}
}

func TestTheGovernorIsOffByDefault(t *testing.T) {
	m := newTestManager(t)
	sub := m.Subscribe(16)
	defer sub.Close()
	item := &entry{}

	m.maybeContextNotice(item, fill("alpha", 99))
	if got := drainNotices(sub); len(got) != 0 {
		t.Fatalf("notices with no setting = %v, want none", got)
	}
}

func TestTheGovernorSkipsAnUnknownWindow(t *testing.T) {
	m := governorManager(t, 60, 80, ContextActionHold)
	sub := m.Subscribe(16)
	defer sub.Close()
	item := &entry{}

	m.maybeContextNotice(item, session.Snapshot{
		Name: "alpha", Model: "some-other-model", ContextTokens: 900_000,
	})
	if got := drainNotices(sub); len(got) != 0 {
		t.Fatalf("notices for an unknown window = %v, want none", got)
	}
}

func TestTheActThresholdSkipsTheWarnNotice(t *testing.T) {
	m := governorManager(t, 60, 80, ContextActionHold)
	sub := m.Subscribe(16)
	defer sub.Close()
	item := &entry{}

	m.maybeContextNotice(item, fill("alpha", 95))
	if got := drainNotices(sub); len(got) != 1 {
		t.Fatalf("notices = %v, want only the act notice", got)
	}
}

func TestContextFillReadsThePercentOfTheWindow(t *testing.T) {
	got, ok := contextFill(fill("alpha", 25))
	if !ok || got != 25 {
		t.Fatalf("contextFill = %d, %v, want 25 and true", got, ok)
	}
	if _, ok := contextFill(session.Snapshot{Model: "claude-sonnet-4-6"}); ok {
		t.Fatal("contextFill reported a fill for an empty context")
	}
}

func TestAHeldSessionRefusesAPrompt(t *testing.T) {
	m := newTestManager(t)
	name, err := m.Spawn(context.Background(), Spec{Name: "alpha", Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	item, err := m.entry(name)
	if err != nil {
		t.Fatalf("entry: %v", err)
	}
	item.ctxMu.Lock()
	item.ctxHeld = true
	item.ctxMu.Unlock()

	if !m.ContextHeld(name) {
		t.Fatal("ContextHeld did not report the hold")
	}
	if err := m.Send(name, "hello"); !errors.Is(err, ErrContextHeld) {
		t.Fatalf("Send = %v, want ErrContextHeld", err)
	}
	if _, err := m.SendFrom(name, "boss", "hello"); !errors.Is(err, ErrContextHeld) {
		t.Fatalf("SendFrom = %v, want ErrContextHeld", err)
	}

	if err := m.ClearContextHold(name); err != nil {
		t.Fatalf("ClearContextHold: %v", err)
	}
	if m.ContextHeld(name) {
		t.Fatal("the hold survived the clear")
	}
	if err := m.Send(name, "hello"); errors.Is(err, ErrContextHeld) {
		t.Fatal("Send still refused after the clear")
	}
}

func TestHeldSessionsNamesTheHeldOnes(t *testing.T) {
	m := newTestManager(t)
	held, err := m.Spawn(context.Background(), Spec{Name: "held", Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if _, err := m.Spawn(context.Background(), Spec{Name: "free", Dir: t.TempDir()}); err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	item, _ := m.entry(held)
	item.ctxMu.Lock()
	item.ctxHeld = true
	item.ctxMu.Unlock()

	names := m.HeldSessions()
	if !names[held] || names["free"] {
		t.Fatalf("HeldSessions = %v, want only %q", names, held)
	}
}

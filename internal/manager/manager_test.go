package manager

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

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
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	m, err := New(Options{
		Root:       t.TempDir(),
		ClaudePath: fakeClaude,
		Renderer:   render.Renderer{},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		m.Shutdown(ctx)
	})
	return m
}

func TestBusDropsTheOldestEventAndCountsIt(t *testing.T) {
	bus := NewBus()
	sub := bus.Subscribe(4)
	for i := 0; i < 10; i++ {
		bus.Publish(Event{Session: fmt.Sprintf("s%d", i)})
	}
	var received int
	for {
		select {
		case <-sub.C:
			received++
			continue
		default:
		}
		break
	}
	if got := int(sub.Dropped()) + received; got != 10 {
		t.Fatalf("received %d and dropped %d, want 10 in total", received, sub.Dropped())
	}
	if received != 4 {
		t.Fatalf("received %d, want the buffer size of 4", received)
	}
}

func TestBusKeepsTheNewestEventWhenItDrops(t *testing.T) {
	bus := NewBus()
	sub := bus.Subscribe(2)
	for _, name := range []string{"a", "b", "c"} {
		bus.Publish(Event{Session: name})
	}
	var got []string
	for len(sub.C) > 0 {
		got = append(got, (<-sub.C).Session)
	}
	if strings.Join(got, ",") != "b,c" {
		t.Fatalf("got %v, want the two newest events", got)
	}
}

func TestBusStopsSendingAfterClose(t *testing.T) {
	bus := NewBus()
	sub := bus.Subscribe(1)
	sub.Close()
	sub.Close()
	bus.Publish(Event{Session: "a"})
	if bus.Subscribers() != 0 {
		t.Fatal("the subscriber was not removed")
	}
}

func TestManagerRunsManySessionsAtOnce(t *testing.T) {
	const count = 20
	m := newTestManager(t)
	sub := m.Subscribe(4096)
	defer sub.Close()

	var (
		mu      sync.Mutex
		seqs    []uint64
		results = make(map[string]int)
		done    = make(chan struct{})
	)
	go func() {
		defer close(done)
		for ev := range sub.C {
			mu.Lock()
			seqs = append(seqs, ev.Seq)
			for _, line := range ev.Lines {
				if strings.HasPrefix(line.Text, "✓ ") {
					results[ev.Session]++
				}
			}
			finished := len(results)
			mu.Unlock()
			if finished == count {
				return
			}
		}
	}()

	dir := t.TempDir()
	names := make([]string, 0, count)
	for i := 0; i < count; i++ {
		name, err := m.Spawn(context.Background(), Spec{Name: fmt.Sprintf("s%d", i), Dir: dir})
		if err != nil {
			t.Fatalf("Spawn %d: %v", i, err)
		}
		names = append(names, name)
	}
	for _, name := range names {
		if err := m.Send(name, "hello"); err != nil {
			t.Fatalf("Send %s: %v", name, err)
		}
	}

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		mu.Lock()
		t.Fatalf("only %d of %d sessions finished a turn", len(results), count)
	}

	mu.Lock()
	defer mu.Unlock()
	if sub.Dropped() != 0 {
		t.Errorf("a subscriber with a large buffer must not drop, got %d", sub.Dropped())
	}
	for i := 1; i < len(seqs); i++ {
		if seqs[i] != seqs[i-1]+1 {
			t.Fatalf("the sequence jumped from %d to %d without a drop", seqs[i-1], seqs[i])
		}
	}
	for _, name := range names {
		if results[name] != 1 {
			t.Errorf("session %s finished %d turns, want 1", name, results[name])
		}
		lines := strings.Join(render.Text(m.Lines(name)), "\n")
		if !strings.Contains(lines, "echo: hello") {
			t.Errorf("session %s has no assistant line:\n%s", name, lines)
		}
	}
}

func TestManagerSurvivesASlowSubscriber(t *testing.T) {
	m := newTestManager(t)
	slow := m.Subscribe(1)
	defer slow.Close()

	dir := t.TempDir()
	name, err := m.Spawn(context.Background(), Spec{Dir: dir})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	for i := 0; i < 5; i++ {
		if err := m.Send(name, fmt.Sprintf("prompt %d", i)); err != nil {
			t.Fatalf("Send: %v", err)
		}
	}
	waitFor(t, 20*time.Second, func() bool {
		snap, err := m.Snapshot(name)
		return err == nil && snap.Turns >= 5
	})
	if slow.Dropped() == 0 {
		t.Error("a subscriber with a buffer of one must drop events")
	}
	if got := len(m.Lines(name)); got < 5 {
		t.Errorf("the manager buffer holds %d lines, so it lost output", got)
	}
}

func TestManagerShowsThePromptInTheOutput(t *testing.T) {
	m := newTestManager(t)
	name, err := m.Spawn(context.Background(), Spec{Name: "echo", Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if err := m.Send(name, "what is in this directory"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	waitFor(t, 10*time.Second, func() bool {
		snap, err := m.Snapshot(name)
		return err == nil && snap.Turns >= 1
	})

	lines := m.Lines(name)
	var prompt *render.Line
	for i := range lines {
		if lines[i].Class == render.ClassPrompt {
			prompt = &lines[i]
			break
		}
	}
	if prompt == nil {
		t.Fatalf("no prompt line in the output:\n%s", strings.Join(render.Text(lines), "\n"))
	}
	if prompt.Text != "› what is in this directory" {
		t.Fatalf("prompt line = %q", prompt.Text)
	}
	order := render.Text(lines)
	promptAt, replyAt := -1, -1
	for i, line := range order {
		if strings.HasPrefix(line, "› ") && promptAt < 0 {
			promptAt = i
		}
		if strings.HasPrefix(line, "echo: ") && replyAt < 0 {
			replyAt = i
		}
	}
	if promptAt < 0 || replyAt < 0 || promptAt > replyAt {
		t.Fatalf("the prompt must come before the reply:\n%s", strings.Join(order, "\n"))
	}
}

func TestManagerMakesEveryNameUnique(t *testing.T) {
	m := newTestManager(t)
	dir := t.TempDir()
	var names []string
	for i := 0; i < 3; i++ {
		name, err := m.Spawn(context.Background(), Spec{Name: "api", Dir: dir})
		if err != nil {
			t.Fatalf("Spawn %d: %v", i, err)
		}
		names = append(names, name)
	}
	want := []string{"api", "api-2", "api-3"}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("names = %v, want %v", names, want)
		}
	}
}

func TestManagerNamesASessionAfterItsDirectory(t *testing.T) {
	m := newTestManager(t)
	dir := filepath.Join(t.TempDir(), "widgets")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	name, err := m.Spawn(context.Background(), Spec{Dir: dir})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if name != "widgets" {
		t.Fatalf("name = %q, want widgets", name)
	}
}

func TestManagerWritesMeta(t *testing.T) {
	m := newTestManager(t)
	dir := t.TempDir()
	name, err := m.Spawn(context.Background(), Spec{Name: "meta", Dir: dir, Model: "fake-model"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if err := m.Send(name, "hello"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	waitFor(t, 10*time.Second, func() bool {
		snap, err := m.Snapshot(name)
		return err == nil && snap.Turns >= 1
	})

	path := filepath.Join(m.Root(), "sessions", name, "meta.json")
	waitFor(t, 5*time.Second, func() bool {
		meta, err := ReadMeta(path)
		return err == nil && meta.ClaudeSessionID != ""
	})
	meta, err := ReadMeta(path)
	if err != nil {
		t.Fatalf("ReadMeta: %v", err)
	}
	if meta.Name != "meta" || meta.Model != "fake-model" || meta.Dir != dir {
		t.Fatalf("meta = %+v", meta)
	}
}

func TestManagerReportsAnUnknownSession(t *testing.T) {
	m := newTestManager(t)
	if err := m.Send("nope", "hello"); err == nil {
		t.Fatal("Send to an unknown session must fail")
	}
	if lines := m.Lines("nope"); lines != nil {
		t.Fatalf("Lines = %v, want none", lines)
	}
}

func TestManagerPublishesAClosedEvent(t *testing.T) {
	m := newTestManager(t)
	sub := m.Subscribe(256)
	defer sub.Close()

	name, err := m.Spawn(context.Background(), Spec{Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := m.Stop(ctx, name); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	deadline := time.After(10 * time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Closed {
				if ev.Snapshot.State != session.StateExited {
					t.Fatalf("final state = %v, want exited", ev.Snapshot.State)
				}
				return
			}
		case <-deadline:
			t.Fatal("no closed event arrived")
		}
	}
}

func waitForMeta(t *testing.T, m *Manager, name string) Meta {
	t.Helper()
	var meta Meta
	waitFor(t, 10*time.Second, func() bool {
		got, err := ReadMeta(filepath.Join(m.Root(), "sessions", name, "meta.json"))
		if err != nil {
			return false
		}
		meta = got
		return true
	})
	return meta
}

func retire(t *testing.T, m *Manager, name string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := m.Stop(ctx, name); err != nil {
		t.Fatalf("Stop %s: %v", name, err)
	}
	waitFor(t, 10*time.Second, func() bool { return m.Remove(name) == nil })
}

func runOneTurn(t *testing.T, m *Manager, name, prompt string) {
	t.Helper()
	if err := m.Send(name, prompt); err != nil {
		t.Fatalf("Send: %v", err)
	}
	waitFor(t, 10*time.Second, func() bool {
		snap, err := m.Snapshot(name)
		return err == nil && snap.Turns >= 1
	})
}

func waitFor(t *testing.T, limit time.Duration, cond func() bool) {
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

func TestStoredKeepsOnlySessionsThatDidWork(t *testing.T) {
	m := newTestManager(t)
	dir := t.TempDir()

	worked, err := m.Spawn(context.Background(), Spec{Name: "worked", Dir: dir})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	runOneTurn(t, m, worked, "hello")

	idle, err := m.Spawn(context.Background(), Spec{Name: "idle", Dir: dir})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	waitForMeta(t, m, worked)
	retire(t, m, worked)
	retire(t, m, idle)
	waitFor(t, 10*time.Second, func() bool {
		_, err := os.Stat(filepath.Join(m.Root(), "sessions", "idle"))
		return os.IsNotExist(err)
	})

	stored := m.Stored()
	if len(stored) != 1 {
		t.Fatalf("stored = %+v, want only the session that finished a turn", stored)
	}
	meta := stored[0]
	if meta.Name != "worked" || meta.Turns != 1 || meta.Cost != 0.25 {
		t.Fatalf("meta = %+v", meta)
	}
	if meta.ClaudeSessionID == "" || meta.Dir != dir {
		t.Fatalf("meta = %+v", meta)
	}
	if _, err := os.Stat(filepath.Join(m.Root(), "sessions", "idle")); !os.IsNotExist(err) {
		t.Error("a session with no turns must leave nothing behind")
	}
}

func TestStoredHidesTheLiveSessions(t *testing.T) {
	m := newTestManager(t)
	name, err := m.Spawn(context.Background(), Spec{Name: "live", Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	runOneTurn(t, m, name, "hello")
	if stored := m.Stored(); len(stored) != 0 {
		t.Fatalf("stored = %+v, want none while the session runs", stored)
	}
}

func TestReplayRebuildsThePastOutput(t *testing.T) {
	m := newTestManager(t)
	name, err := m.Spawn(context.Background(), Spec{Name: "past", Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	runOneTurn(t, m, name, "remember this")
	retire(t, m, name)

	replay := strings.Join(render.Text(m.Replay(name)), "\n")
	for _, want := range []string{"› remember this", "echo: remember this", "✓ success"} {
		if !strings.Contains(replay, want) {
			t.Errorf("the replay has no %q:\n%s", want, replay)
		}
	}
	if m.Replay("nope") != nil {
		t.Error("an unknown session must replay nothing")
	}
}

func TestResumeKeepsTheNameAndAddsToTheSameTranscript(t *testing.T) {
	m := newTestManager(t)
	dir := t.TempDir()
	name, err := m.Spawn(context.Background(), Spec{Name: "again", Dir: dir})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	runOneTurn(t, m, name, "first")
	waitForMeta(t, m, name)
	retire(t, m, name)
	waitFor(t, 10*time.Second, func() bool { return len(m.Stored()) == 1 })

	resumed, err := m.Resume(context.Background(), m.Stored()[0])
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if resumed != "again" {
		t.Fatalf("resumed as %q, want the same name", resumed)
	}
	runOneTurn(t, m, resumed, "second")
	waitFor(t, 10*time.Second, func() bool {
		meta, err := ReadMeta(filepath.Join(m.Root(), "sessions", "again", "meta.json"))
		return err == nil && meta.Turns == 2
	})

	replay := strings.Join(render.Text(m.Replay("again")), "\n")
	if !strings.Contains(replay, "› first") || !strings.Contains(replay, "› second") {
		t.Fatalf("the transcript lost a turn:\n%s", replay)
	}
	meta, err := ReadMeta(filepath.Join(m.Root(), "sessions", "again", "meta.json"))
	if err != nil {
		t.Fatal(err)
	}
	if meta.Turns != 2 || meta.Cost != 0.5 {
		t.Fatalf("the totals did not carry over: %+v", meta)
	}
}

func TestResumeWithEffortKeepsTheNameAndStoresTheLevel(t *testing.T) {
	m := newTestManager(t)
	name, err := m.Spawn(context.Background(), Spec{Name: "grind", Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	runOneTurn(t, m, name, "first")
	waitForMeta(t, m, name)

	resumed, err := m.ResumeWithEffort(context.Background(), name, "high")
	if err != nil {
		t.Fatalf("ResumeWithEffort: %v", err)
	}
	if resumed != "grind" {
		t.Fatalf("resumed as %q, want the same name", resumed)
	}
	runOneTurn(t, m, resumed, "second")
	waitFor(t, 10*time.Second, func() bool {
		meta, err := ReadMeta(filepath.Join(m.Root(), "sessions", "grind", "meta.json"))
		return err == nil && meta.Effort == "high" && meta.Turns == 2
	})
	retire(t, m, resumed)
}

func TestResumeWithEffortNeedsASessionThatHasRun(t *testing.T) {
	m := newTestManager(t)
	name, err := m.Spawn(context.Background(), Spec{Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	defer retire(t, m, name)
	if _, err := m.ResumeWithEffort(context.Background(), name, "high"); err == nil {
		t.Fatal("a session with no turn must not resume for effort")
	}
}

func TestANewSessionNeverTakesAStoredName(t *testing.T) {
	m := newTestManager(t)
	dir := t.TempDir()
	name, err := m.Spawn(context.Background(), Spec{Name: "shared", Dir: dir})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	runOneTurn(t, m, name, "hello")
	waitForMeta(t, m, name)
	retire(t, m, name)

	fresh, err := m.Spawn(context.Background(), Spec{Name: "shared", Dir: dir})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if fresh == "shared" {
		t.Fatal("a new session must not write over a stored transcript")
	}
}

func TestArchiveHidesASessionAndGivesItBack(t *testing.T) {
	m := newTestManager(t)
	name, err := m.Spawn(context.Background(), Spec{Name: "old", Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	runOneTurn(t, m, name, "hello")

	if err := m.Archive(name, true); !errors.Is(err, ErrStillLive) {
		t.Fatalf("archiving a live session must fail, got %v", err)
	}

	waitForMeta(t, m, name)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := m.Stop(ctx, name); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if err := m.Archive(name, true); err != nil {
		t.Fatalf("Archive: %v", err)
	}
	stored := m.Stored()
	if len(stored) != 1 || !stored[0].Archived {
		t.Fatalf("stored = %+v, want one archived session", stored)
	}
	if stored[0].ArchivedAt.IsZero() {
		t.Error("the archive time was not recorded")
	}
	if len(m.Names()) != 0 {
		t.Errorf("an archived session must leave the live list, got %v", m.Names())
	}

	if err := m.Archive(name, false); err != nil {
		t.Fatalf("un-archive: %v", err)
	}
	if stored = m.Stored(); len(stored) != 1 || stored[0].Archived {
		t.Fatalf("stored = %+v, want it back", stored)
	}
}

func TestSetTitleRenamesALiveSessionAndPersistsIt(t *testing.T) {
	m := newTestManager(t)
	name, err := m.Spawn(context.Background(), Spec{Name: "old", Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	runOneTurn(t, m, name, "hello")

	if err := m.SetTitle(name, "My work"); err != nil {
		t.Fatalf("SetTitle: %v", err)
	}
	snap, err := m.Snapshot(name)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if snap.Title != "My work" {
		t.Fatalf("snapshot title = %q, want %q", snap.Title, "My work")
	}

	waitFor(t, 10*time.Second, func() bool {
		meta, err := ReadMeta(filepath.Join(m.Root(), "sessions", name, "meta.json"))
		return err == nil && meta.Title == "My work"
	})

	if err := m.SetTitle(name, ""); err != nil {
		t.Fatalf("SetTitle clear: %v", err)
	}
	snap, _ = m.Snapshot(name)
	if snap.Title != "" {
		t.Fatalf("cleared title = %q, want empty", snap.Title)
	}
}

func TestSetTitleRenamesAStoredSession(t *testing.T) {
	m := newTestManager(t)
	name, err := m.Spawn(context.Background(), Spec{Name: "kept", Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	runOneTurn(t, m, name, "hello")
	waitForMeta(t, m, name)
	retire(t, m, name)

	if err := m.SetTitle(name, "Archived work"); err != nil {
		t.Fatalf("SetTitle: %v", err)
	}
	meta, err := ReadMeta(filepath.Join(m.Root(), "sessions", name, "meta.json"))
	if err != nil {
		t.Fatalf("ReadMeta: %v", err)
	}
	if meta.Title != "Archived work" {
		t.Fatalf("stored title = %q, want %q", meta.Title, "Archived work")
	}

	if err := m.SetTitle("nope", "x"); err == nil {
		t.Fatal("SetTitle on an unknown session must fail")
	}
}

func TestPartialTextGrowsAndThenClears(t *testing.T) {
	m := newTestManager(t)
	sub := m.Subscribe(4096)
	defer sub.Close()

	name, err := m.Spawn(context.Background(), Spec{Name: "stream", Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if err := m.Send(name, "hello there friend"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	waitFor(t, 10*time.Second, func() bool {
		snap, err := m.Snapshot(name)
		return err == nil && snap.Turns >= 1
	})

	var partials []string
	var afterReply string
	for len(sub.C) > 0 {
		ev := <-sub.C
		if ev.Partial != "" {
			partials = append(partials, ev.Partial)
		}
		for _, line := range ev.Lines {
			if strings.HasPrefix(line.Text, "echo: ") {
				afterReply = ev.Partial
			}
		}
	}

	if len(partials) < 2 {
		t.Fatalf("got %d partial events, want the text to arrive in pieces", len(partials))
	}
	for i := 1; i < len(partials); i++ {
		if !strings.HasPrefix(partials[i], partials[i-1]) {
			t.Fatalf("partial %d is not a continuation of %d:\n%q\n%q",
				i, i-1, partials[i-1], partials[i])
		}
	}
	if last := partials[len(partials)-1]; last != "echo: hello there friend" {
		t.Fatalf("the last partial = %q", last)
	}
	if afterReply != "" {
		t.Fatalf("the partial must clear when the whole message lands, got %q", afterReply)
	}
}

func TestTheTranscriptHoldsNoPartialEvents(t *testing.T) {
	m := newTestManager(t)
	name, err := m.Spawn(context.Background(), Spec{Name: "stream", Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	runOneTurn(t, m, name, "hello there friend")
	retire(t, m, name)

	data, err := os.ReadFile(filepath.Join(m.Root(), "sessions", name, "transcript.jsonl"))
	if err != nil {
		t.Fatalf("read transcript: %v", err)
	}
	if strings.Contains(string(data), "stream_event") {
		t.Fatalf("the transcript holds partial events:\n%s", data)
	}
	replay := strings.Join(render.Text(m.Replay(name)), "\n")
	if got := strings.Count(replay, "echo: hello there friend"); got != 1 {
		t.Fatalf("the reply appears %d times in the replay, want once:\n%s", got, replay)
	}
}

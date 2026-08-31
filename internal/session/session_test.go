package session

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/protocol"
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

type collector struct {
	events []Event
	done   chan struct{}
}

func collect(s *Session) *collector {
	c := &collector{done: make(chan struct{})}
	go func() {
		defer close(c.done)
		for ev := range s.Events() {
			c.events = append(c.events, ev)
		}
	}()
	return c
}

func (c *collector) wait(t *testing.T) []Event {
	t.Helper()
	select {
	case <-c.done:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for the event channel to close")
	}
	return c.events
}

func (c *collector) states() []State {
	var out []State
	for _, ev := range c.events {
		if ev.Kind == KindState {
			out = append(out, ev.State)
		}
	}
	return out
}

func (c *collector) results() []*protocol.Result {
	var out []*protocol.Result
	for _, ev := range c.events {
		if ev.Kind == KindProtocol && ev.Protocol.Result != nil {
			out = append(out, ev.Protocol.Result)
		}
	}
	return out
}

func newTestSession(t *testing.T, cfg Config) *Session {
	t.Helper()
	if cfg.ClaudePath == "" {
		cfg.ClaudePath = fakeClaude
	}
	if cfg.Dir == "" {
		cfg.Dir = t.TempDir()
	}
	s, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s
}

func TestConfigArgs(t *testing.T) {
	cfg := Config{
		Model:          "claude-opus-5",
		PermissionMode: "plan",
		Effort:         "high",
		AllowedTools:   []string{"Read", "Bash(git *)"},
		ResumeID:       "abc",
		IncludePartial: true,
		ExtraArgs:      []string{"--add-dir", "/tmp"},
	}
	cfg.applyDefaults()
	got := strings.Join(cfg.Args(), " ")

	for _, want := range []string{
		"-p",
		"--output-format stream-json",
		"--input-format stream-json",
		"--verbose",
		"--permission-mode plan",
		"--model claude-opus-5",
		"--effort high",
		"--allowedTools Read,Bash(git *)",
		"--resume abc",
		"--include-partial-messages",
		"--add-dir /tmp",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("args %q do not contain %q", got, want)
		}
	}
}

func TestConfigArgsOmitsEffortWhenEmpty(t *testing.T) {
	var cfg Config
	cfg.applyDefaults()
	if strings.Contains(strings.Join(cfg.Args(), " "), "--effort") {
		t.Fatalf("no effort must give no --effort flag, got %v", cfg.Args())
	}
}

func TestConfigArgsDefaultPermissionMode(t *testing.T) {
	var cfg Config
	cfg.applyDefaults()
	if !strings.Contains(strings.Join(cfg.Args(), " "), "--permission-mode auto") {
		t.Fatalf("the default permission mode must be auto, got %v", cfg.Args())
	}
}

func TestSessionRunsOneTurn(t *testing.T) {
	s := newTestSession(t, Config{Name: "one", Model: "fake-model"})
	c := collect(s)
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := s.Send("hello"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	waitForState(t, s, StateIdle, 5*time.Second)
	waitForTurns(t, s, 1, 5*time.Second)

	stop, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Stop(stop); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	c.wait(t)

	snap := s.Snapshot()
	if snap.State != StateExited {
		t.Errorf("state = %v, want exited", snap.State)
	}
	if snap.ClaudeSessionID == "" {
		t.Error("the session id from the init event was not stored")
	}
	if snap.Turns != 1 || snap.Cost != 0.25 {
		t.Errorf("turns = %d, cost = %v, want 1 and 0.25", snap.Turns, snap.Cost)
	}

	wantStates := []State{StateBusy, StateIdle, StateExited}
	if got := c.states(); !containsInOrder(got, wantStates) {
		t.Errorf("states = %v, want %v in that order", got, wantStates)
	}

	var echoed bool
	for _, ev := range c.events {
		if ev.Kind == KindProtocol && ev.Protocol.Type == protocol.TypeAssistant {
			if ev.Protocol.Text() == "echo: hello" {
				echoed = true
			}
		}
	}
	if !echoed {
		t.Error("the assistant text did not reach the consumer")
	}
}

func TestSessionSendsTheFirstPromptBeforeTheInitEvent(t *testing.T) {
	s := newTestSession(t, Config{
		Name: "lazy",
		Env:  []string{"FAKECLAUDE_MODE=lazyinit"},
	})
	c := collect(s)
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	for _, text := range []string{"one", "two"} {
		if err := s.Send(text); err != nil {
			t.Fatalf("Send %q: %v", text, err)
		}
	}
	waitForTurns(t, s, 2, 10*time.Second)

	stop, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.Stop(stop)
	c.wait(t)

	if snap := s.Snapshot(); snap.ClaudeSessionID == "" {
		t.Error("the late init event was not applied")
	}
	results := c.results()
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	for i, want := range []string{"echo: one", "echo: two"} {
		if results[i].Result != want {
			t.Errorf("result %d = %q, want %q", i, results[i].Result, want)
		}
	}
}

func TestSessionQueuesPromptsWhileBusy(t *testing.T) {
	s := newTestSession(t, Config{Name: "queue"})
	c := collect(s)
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	for _, text := range []string{"one", "two", "three"} {
		if err := s.Send(text); err != nil {
			t.Fatalf("Send %q: %v", text, err)
		}
	}
	waitForTurns(t, s, 3, 10*time.Second)

	stop, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.Stop(stop)
	c.wait(t)

	results := c.results()
	if len(results) != 3 {
		t.Fatalf("got %d results, want 3", len(results))
	}
	for i, want := range []string{"echo: one", "echo: two", "echo: three"} {
		if results[i].Result != want {
			t.Errorf("result %d = %q, want %q", i, results[i].Result, want)
		}
	}
	if snap := s.Snapshot(); snap.Queued != 0 {
		t.Errorf("queue length = %d, want 0", snap.Queued)
	}
}

func TestSessionRecordsTranscript(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions", "tr", "transcript.jsonl")
	s := newTestSession(t, Config{Name: "tr", TranscriptPath: path})
	c := collect(s)
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := s.Send("hello"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	waitForTurns(t, s, 1, 5*time.Second)

	stop, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.Stop(stop)
	c.wait(t)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read transcript: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Fatalf("transcript has %d lines, want 3:\n%s", len(lines), data)
	}
	if !strings.Contains(lines[0], `"subtype":"init"`) {
		t.Errorf("the first transcript line is not the init event: %s", lines[0])
	}
}

func TestSessionFailsWhenChildCrashes(t *testing.T) {
	s := newTestSession(t, Config{
		Name: "crash",
		Env:  []string{"FAKECLAUDE_MODE=crash"},
	})
	c := collect(s)
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := s.Wait(); err == nil {
		t.Fatal("Wait must report the non-zero exit code")
	}
	c.wait(t)

	if state := s.State(); state != StateFailed {
		t.Errorf("state = %v, want failed", state)
	}
	stderr := strings.Join(s.Stderr(), "\n")
	if !strings.Contains(stderr, "exploded on purpose") {
		t.Errorf("stderr was not captured: %q", stderr)
	}
}

func TestSessionTakesTheModelAndModeFromTheInitEvent(t *testing.T) {
	s := newTestSession(t, Config{Name: "init", PermissionMode: "plan"})
	c := collect(s)
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := s.Send("hello"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	waitForTurns(t, s, 1, 5*time.Second)

	stop, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.Stop(stop)
	c.wait(t)

	snap := s.Snapshot()
	if snap.Model != "fake-model" {
		t.Errorf("model = %q, want the model from the init event", snap.Model)
	}
	if snap.PermissionMode != "auto" {
		t.Errorf("permission mode = %q, want the mode the child reports", snap.PermissionMode)
	}
}

func TestSessionRejectsSendAfterExit(t *testing.T) {
	s := newTestSession(t, Config{
		Name: "gone",
		Env:  []string{"FAKECLAUDE_MODE=exit-after-init"},
	})
	c := collect(s)
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	_ = s.Wait()
	c.wait(t)

	if err := s.Send("hello"); err != ErrNotLive {
		t.Fatalf("Send after exit = %v, want ErrNotLive", err)
	}
}

func TestSessionKeepsGoingAfterANonJSONLine(t *testing.T) {
	s := newTestSession(t, Config{
		Name: "garbage",
		Env:  []string{"FAKECLAUDE_MODE=garbage"},
	})
	c := collect(s)
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := s.Send("hello"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	waitForTurns(t, s, 1, 5*time.Second)

	stop, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.Stop(stop)
	c.wait(t)

	var sawError bool
	for _, ev := range c.events {
		if ev.Kind == KindError && strings.Contains(ev.Line, "not JSON") {
			sawError = true
		}
	}
	if !sawError {
		t.Error("the non-JSON line was not reported")
	}
	if len(c.results()) != 1 {
		t.Errorf("got %d results, want 1", len(c.results()))
	}
}

func TestSessionPassesDirectoryAndFlagsToTheChild(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	s := newTestSession(t, Config{
		Name:           "args",
		Dir:            dir,
		Model:          "claude-opus-5",
		PermissionMode: "plan",
		Env:            []string{"FAKECLAUDE_ARGS_FILE=" + argsFile, "FAKECLAUDE_MODE=exit-after-init"},
	})
	c := collect(s)
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	_ = s.Wait()
	c.wait(t)

	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read args file: %v", err)
	}
	recorded := string(data)
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	if !strings.Contains(recorded, "cwd="+resolved) {
		t.Errorf("the child did not run in %s:\n%s", resolved, recorded)
	}
	for _, want := range []string{"--permission-mode", "plan", "--model", "claude-opus-5"} {
		if !strings.Contains(recorded, want) {
			t.Errorf("the child did not receive %q:\n%s", want, recorded)
		}
	}
}

func TestNewRejectsAMissingDirectory(t *testing.T) {
	_, err := New(Config{Name: "bad", Dir: filepath.Join(t.TempDir(), "nope")})
	if err == nil {
		t.Fatal("New must reject a directory that does not exist")
	}
}

func TestSessionInterruptEndsTheTurn(t *testing.T) {
	s := newTestSession(t, Config{Name: "int", Env: []string{"FAKECLAUDE_MODE=interruptible"}})
	c := collect(s)
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := s.Send("one"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	waitForState(t, s, StateBusy, 5*time.Second)
	if err := s.Interrupt(); err != nil {
		t.Fatalf("Interrupt: %v", err)
	}
	waitForState(t, s, StateIdle, 5*time.Second)

	stop, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.Stop(stop)
	c.wait(t)

	results := c.results()
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Result != "interrupted: one" || !results[0].IsError {
		t.Errorf("result = %q (is_error=%v), want the interrupted turn", results[0].Result, results[0].IsError)
	}
}

func TestSessionInterruptFlushesTheQueue(t *testing.T) {
	s := newTestSession(t, Config{Name: "flush", Env: []string{"FAKECLAUDE_MODE=interruptible"}})
	c := collect(s)
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := s.Send("one"); err != nil {
		t.Fatalf("Send one: %v", err)
	}
	waitForState(t, s, StateBusy, 5*time.Second)
	if err := s.Send("two"); err != nil {
		t.Fatalf("Send two: %v", err)
	}
	if snap := s.Snapshot(); snap.Queued != 1 {
		t.Fatalf("queue length = %d, want 1 while busy", snap.Queued)
	}

	if err := s.Interrupt(); err != nil {
		t.Fatalf("Interrupt one: %v", err)
	}
	waitForTurns(t, s, 1, 5*time.Second)
	if err := s.Interrupt(); err != nil {
		t.Fatalf("Interrupt two: %v", err)
	}
	waitForTurns(t, s, 2, 5*time.Second)

	stop, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.Stop(stop)
	c.wait(t)

	results := c.results()
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	for i, want := range []string{"interrupted: one", "interrupted: two"} {
		if results[i].Result != want {
			t.Errorf("result %d = %q, want %q", i, results[i].Result, want)
		}
	}
	if snap := s.Snapshot(); snap.Queued != 0 {
		t.Errorf("queue length = %d, want 0", snap.Queued)
	}
}

func TestSessionDiscardQueuedStopsTheQueuedPrompt(t *testing.T) {
	s := newTestSession(t, Config{Name: "drop", Env: []string{"FAKECLAUDE_MODE=interruptible"}})
	c := collect(s)
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := s.Send("one"); err != nil {
		t.Fatalf("Send one: %v", err)
	}
	waitForState(t, s, StateBusy, 5*time.Second)
	if err := s.Send("two"); err != nil {
		t.Fatalf("Send two: %v", err)
	}
	s.DiscardQueued()
	if snap := s.Snapshot(); snap.Queued != 0 {
		t.Fatalf("queue length = %d, want 0 after discard", snap.Queued)
	}
	if err := s.Interrupt(); err != nil {
		t.Fatalf("Interrupt: %v", err)
	}
	waitForState(t, s, StateIdle, 5*time.Second)
	time.Sleep(50 * time.Millisecond)
	if snap := s.Snapshot(); snap.State != StateIdle {
		t.Errorf("state = %v, want idle — the dropped prompt must not run", snap.State)
	}

	stop, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.Stop(stop)
	c.wait(t)

	results := c.results()
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1 — the queued prompt was not dropped", len(results))
	}
	if results[0].Result != "interrupted: one" {
		t.Errorf("result = %q, want the first turn only", results[0].Result)
	}
}

func waitForState(t *testing.T, s *Session, want State, limit time.Duration) {
	t.Helper()
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		if s.State() == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for state %v, now %v", want, s.State())
}

func waitForTurns(t *testing.T, s *Session, want int, limit time.Duration) {
	t.Helper()
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		if s.Snapshot().Turns >= want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d turns, now %d", want, s.Snapshot().Turns)
}

func containsInOrder(got, want []State) bool {
	next := 0
	for _, state := range got {
		if next < len(want) && state == want[next] {
			next++
		}
	}
	return next == len(want)
}

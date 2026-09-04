package session

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/protocol"
)

const (
	DefaultPermissionMode = "auto"
	DefaultClaudePath     = "claude"
	DefaultEventBuffer    = 256
	DefaultStderrLines    = 20
	DefaultStopGrace      = 5 * time.Second
)

var (
	ErrNotStarted = errors.New("session: not started")
	ErrNotLive    = errors.New("session: not live")
)

// EffortLevels are the values --effort accepts, from least to most.
var EffortLevels = []string{"low", "medium", "high", "xhigh", "max"}

type Config struct {
	Name            string
	Dir             string
	Model           string
	PermissionMode  string
	Effort          string
	AllowedTools    []string
	DisallowedTools []string
	ResumeID        string
	SessionID       string
	IncludePartial  bool
	ReplayPrompts   bool
	ClaudePath      string
	ExtraArgs       []string
	Env             []string
	TranscriptPath  string
	EventBuffer     int
	StderrLines     int
}

func (c *Config) applyDefaults() {
	if c.ClaudePath == "" {
		c.ClaudePath = DefaultClaudePath
	}
	if c.PermissionMode == "" {
		c.PermissionMode = DefaultPermissionMode
	}
	if c.EventBuffer <= 0 {
		c.EventBuffer = DefaultEventBuffer
	}
	if c.StderrLines <= 0 {
		c.StderrLines = DefaultStderrLines
	}
	if c.Name == "" {
		c.Name = "session"
	}
}

func (c Config) Args() []string {
	args := []string{
		"-p",
		"--output-format", "stream-json",
		"--input-format", "stream-json",
		"--verbose",
		"--permission-mode", c.PermissionMode,
	}
	if c.Model != "" {
		args = append(args, "--model", c.Model)
	}
	if c.Effort != "" {
		args = append(args, "--effort", c.Effort)
	}
	if len(c.AllowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(c.AllowedTools, ","))
	}
	if len(c.DisallowedTools) > 0 {
		args = append(args, "--disallowedTools", strings.Join(c.DisallowedTools, ","))
	}
	if c.SessionID != "" {
		args = append(args, "--session-id", c.SessionID)
	}
	if c.ResumeID != "" {
		args = append(args, "--resume", c.ResumeID)
	}
	if c.IncludePartial {
		args = append(args, "--include-partial-messages")
	}
	if c.ReplayPrompts {
		args = append(args, "--replay-user-messages")
	}
	return append(args, c.ExtraArgs...)
}

type EventKind int

const (
	KindProtocol EventKind = iota
	KindState
	KindStderr
	KindError
)

type Event struct {
	Session  string
	Kind     EventKind
	At       time.Time
	Protocol protocol.Event
	State    State
	Prev     State
	Line     string
	Err      error
	Snapshot Snapshot
}

type Snapshot struct {
	Name            string
	Title           string
	Dir             string
	Model           string
	PermissionMode  string
	Effort          string
	State           State
	ClaudeSessionID string
	Cost            float64
	Turns           int
	Queued          int
	LastDuration    time.Duration
	InputTokens     int
	OutputTokens    int
	ContextTokens   int
	StartedAt       time.Time
	EndedAt         time.Time
	Err             error
	Jobs            []Job
}

// RunningJobs is the count of background jobs that still run.
func (s Snapshot) RunningJobs() int {
	n := 0
	for _, job := range s.Jobs {
		if job.Status.Running() {
			n++
		}
	}
	return n
}

type Session struct {
	cfg    Config
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	writer *protocol.Writer
	events chan Event
	q      *queue
	tr     *Transcript
	stderr *ring

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	readers   sync.WaitGroup
	writeDone chan struct{}
	closeIn   sync.Once

	mu              sync.Mutex
	state           State
	title           string
	model           string
	permissionMode  string
	effort          string
	claudeSessionID string
	cost            float64
	turns           int
	lastDuration    time.Duration
	inputTokens     int
	outputTokens    int
	contextTokens   int
	startedAt       time.Time
	endedAt         time.Time
	sent            bool
	askedQuestion   bool
	interruptSeq    int
	controlSeq      int
	err             error
	jobs            map[string]*Job
	jobOrder        []string

	idleSig chan struct{}
}

func New(cfg Config) (*Session, error) {
	cfg.applyDefaults()
	if cfg.Dir != "" {
		info, err := os.Stat(cfg.Dir)
		if err != nil {
			return nil, fmt.Errorf("session %q: %w", cfg.Name, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("session %q: %s is not a directory", cfg.Name, cfg.Dir)
		}
	}
	return &Session{
		cfg:            cfg,
		events:         make(chan Event, cfg.EventBuffer),
		q:              newQueue(),
		stderr:         newRing(cfg.StderrLines),
		done:           make(chan struct{}),
		writeDone:      make(chan struct{}),
		idleSig:        make(chan struct{}, 1),
		state:          StateStarting,
		model:          cfg.Model,
		permissionMode: cfg.PermissionMode,
		effort:         cfg.Effort,
	}, nil
}

func (s *Session) Name() string { return s.cfg.Name }

func (s *Session) Events() <-chan Event { return s.events }

func (s *Session) Start(parent context.Context) error {
	s.ctx, s.cancel = context.WithCancel(parent)

	cmd := exec.Command(s.cfg.ClaudePath, s.cfg.Args()...)
	cmd.Dir = s.cfg.Dir
	if len(s.cfg.Env) > 0 {
		cmd.Env = append(os.Environ(), s.cfg.Env...)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		s.cancel()
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.cancel()
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		s.cancel()
		return err
	}

	if s.cfg.TranscriptPath != "" {
		tr, err := OpenTranscript(s.cfg.TranscriptPath)
		if err != nil {
			s.cancel()
			return err
		}
		s.tr = tr
	}

	if err := cmd.Start(); err != nil {
		s.cancel()
		s.tr.Close()
		return fmt.Errorf("session %q: start %s: %w", s.cfg.Name, s.cfg.ClaudePath, err)
	}

	s.cmd = cmd
	s.stdin = stdin
	s.writer = protocol.NewWriter(stdin)
	s.mu.Lock()
	s.startedAt = time.Now()
	s.mu.Unlock()

	s.readers.Add(2)
	go s.readStdout(stdout)
	go s.readStderr(stderr)
	go s.writeLoop()
	go s.supervise()
	return nil
}

func (s *Session) Send(text string) error {
	if s.cmd == nil {
		return ErrNotStarted
	}
	if !s.State().Live() {
		return ErrNotLive
	}
	s.q.push(text)
	return nil
}

func (s *Session) Interrupt() error {
	if s.cmd == nil {
		return ErrNotStarted
	}
	if s.State() != StateBusy {
		return nil
	}
	s.mu.Lock()
	s.interruptSeq++
	n := s.interruptSeq
	s.mu.Unlock()
	return s.writer.SendInterrupt(fmt.Sprintf("int-%d", n))
}

func (s *Session) DiscardQueued() {
	s.q.clear()
}

func (s *Session) controlID(kind string) string {
	s.mu.Lock()
	s.controlSeq++
	n := s.controlSeq
	s.mu.Unlock()
	return fmt.Sprintf("%s-%d", kind, n)
}

// SetModel switches the model of the running child; see docs/protocol.md.
func (s *Session) SetModel(model string) error {
	if s.cmd == nil {
		return ErrNotStarted
	}
	if !s.State().Live() {
		return ErrNotLive
	}
	if err := s.writer.SetModel(s.controlID("model"), model); err != nil {
		return err
	}
	s.mu.Lock()
	s.model = model
	s.mu.Unlock()
	return nil
}

// SetPermissionMode switches the permission mode of the running child; see
// docs/protocol.md.
func (s *Session) SetPermissionMode(mode string) error {
	if s.cmd == nil {
		return ErrNotStarted
	}
	if !s.State().Live() {
		return ErrNotLive
	}
	if err := s.writer.SetPermissionMode(s.controlID("mode"), mode); err != nil {
		return err
	}
	s.mu.Lock()
	s.permissionMode = mode
	s.mu.Unlock()
	return nil
}

// SetTitle changes the display title of the session. The title is local
// metadata and does not touch the child, so it works whatever the state.
func (s *Session) SetTitle(title string) {
	s.mu.Lock()
	s.title = title
	state := s.state
	s.mu.Unlock()
	s.emit(Event{Kind: KindState, State: state, Prev: state})
}

func (s *Session) State() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *Session) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Snapshot{
		Name:            s.cfg.Name,
		Title:           s.title,
		Dir:             s.cfg.Dir,
		Model:           s.model,
		PermissionMode:  s.permissionMode,
		Effort:          s.effort,
		State:           s.state,
		ClaudeSessionID: s.claudeSessionID,
		Cost:            s.cost,
		Turns:           s.turns,
		Queued:          s.q.len(),
		LastDuration:    s.lastDuration,
		InputTokens:     s.inputTokens,
		OutputTokens:    s.outputTokens,
		ContextTokens:   s.contextTokens,
		StartedAt:       s.startedAt,
		EndedAt:         s.endedAt,
		Err:             s.err,
		Jobs:            s.jobList(),
	}
}

func (s *Session) jobList() []Job {
	if len(s.jobOrder) == 0 {
		return nil
	}
	out := make([]Job, 0, len(s.jobOrder))
	for _, id := range s.jobOrder {
		out = append(out, *s.jobs[id])
	}
	return out
}

func (s *Session) Stderr() []string { return s.stderr.all() }

func (s *Session) Wait() error {
	<-s.done
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

func (s *Session) Stop(ctx context.Context) error {
	if s.cmd == nil {
		return ErrNotStarted
	}
	s.closeStdin()
	select {
	case <-s.done:
		return s.Wait()
	case <-ctx.Done():
	}
	_ = s.cmd.Process.Kill()
	s.cancel()
	<-s.done
	return s.Wait()
}

func (s *Session) closeStdin() {
	s.closeIn.Do(func() {
		if s.stdin != nil {
			_ = s.stdin.Close()
		}
	})
}

func (s *Session) readStdout(r io.ReadCloser) {
	defer s.readers.Done()
	defer r.Close()
	reader := protocol.NewReader(r)
	for {
		ev, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil && !errors.Is(err, protocol.ErrNotJSON) {
			s.fail(err)
			return
		}
		if ev.Type != protocol.TypeStreamEvent {
			_ = s.tr.WriteLine(ev.Raw)
		}
		if errors.Is(err, protocol.ErrNotJSON) {
			s.emit(Event{Kind: KindError, Line: string(ev.Raw), Err: err})
			continue
		}
		s.apply(ev)
		s.emit(Event{Kind: KindProtocol, Protocol: ev})
	}
}

func (s *Session) readStderr(r io.ReadCloser) {
	defer s.readers.Done()
	defer r.Close()
	reader := protocol.NewReader(r)
	for {
		ev, err := reader.Next()
		if err != nil && !errors.Is(err, protocol.ErrNotJSON) {
			return
		}
		line := string(ev.Raw)
		s.stderr.add(line)
		s.emit(Event{Kind: KindStderr, Line: line})
	}
}

func (s *Session) writeLoop() {
	defer close(s.writeDone)
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.q.sig:
		}
		for {
			if err := s.waitIdle(); err != nil {
				return
			}
			text, ok := s.q.pop()
			if !ok {
				break
			}
			s.setState(StateBusy)
			s.markSent()
			if err := s.writer.SendUser(s.ClaudeSessionID(), text); err != nil {
				s.fail(err)
				return
			}
		}
	}
}

func (s *Session) waitIdle() error {
	for {
		s.mu.Lock()
		state, sent := s.state, s.sent
		s.mu.Unlock()
		switch {
		case state == StateIdle || state == StateWaiting:
			return nil
		case state == StateStarting && !sent:
			// Claude Code emits init only after it reads input; see docs/protocol.md.
			return nil
		case !state.Live():
			return ErrNotLive
		}
		select {
		case <-s.idleSig:
		case <-s.ctx.Done():
			return s.ctx.Err()
		}
	}
}

func (s *Session) ClaudeSessionID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.claudeSessionID
}

func (s *Session) apply(ev protocol.Event) {
	if _, _, ok := ev.AskUserQuestion(); ok {
		s.mu.Lock()
		s.askedQuestion = true
		s.mu.Unlock()
		_ = s.Interrupt()
	}
	switch {
	case ev.Type == protocol.TypeSystem && ev.Task != nil:
		s.applyTask(ev.Subtype, ev.Task)
	case ev.IsInit() && ev.Init != nil:
		s.mu.Lock()
		s.claudeSessionID = ev.Init.SessionID
		if s.model == "" {
			s.model = ev.Init.Model
		}
		if ev.Init.PermissionMode != "" {
			s.permissionMode = ev.Init.PermissionMode
		}
		s.mu.Unlock()
		s.setStateIf(StateStarting, StateIdle)
	case ev.Type == protocol.TypeAssistant && ev.Message != nil && ev.Message.Usage != nil:
		u := ev.Message.Usage
		s.mu.Lock()
		s.contextTokens = u.InputTokens + u.CacheReadInputTokens + u.CacheCreationInputTokens + u.OutputTokens
		s.mu.Unlock()
	case ev.Type == protocol.TypeResult && ev.Result != nil:
		s.mu.Lock()
		s.cost += ev.Result.TotalCostUSD
		s.turns += ev.Result.NumTurns
		s.lastDuration = time.Duration(ev.Result.DurationMS) * time.Millisecond
		if usage := ev.Result.Usage; usage != nil {
			s.inputTokens += usage.InputTokens + usage.CacheReadInputTokens + usage.CacheCreationInputTokens
			s.outputTokens += usage.OutputTokens
		}
		if ev.Result.SessionID != "" {
			s.claudeSessionID = ev.Result.SessionID
		}
		asked := s.askedQuestion
		s.askedQuestion = false
		s.mu.Unlock()
		if asked {
			s.setState(StateWaiting)
		} else {
			s.setState(StateIdle)
		}
	}
}

func (s *Session) applyTask(subtype string, task *protocol.Task) {
	if task.TaskID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	switch subtype {
	case protocol.SubtypeTaskStarted:
		if s.jobs == nil {
			s.jobs = make(map[string]*Job)
		}
		if _, ok := s.jobs[task.TaskID]; ok {
			return
		}
		s.jobs[task.TaskID] = &Job{
			ID:          task.TaskID,
			Description: task.Description,
			TaskType:    task.TaskType,
			Status:      JobRunning,
			StartedAt:   time.Now(),
		}
		s.jobOrder = append(s.jobOrder, task.TaskID)
	case protocol.SubtypeTaskUpdated:
		job := s.jobs[task.TaskID]
		if job == nil || task.Patch == nil {
			return
		}
		s.setJobStatus(job, task.Patch.Status)
	case protocol.SubtypeTaskNotification:
		job := s.jobs[task.TaskID]
		if job == nil {
			return
		}
		s.setJobStatus(job, task.Status)
	}
}

func (s *Session) setJobStatus(job *Job, status string) {
	if !job.Status.Running() {
		return
	}
	next, terminal := classifyStatus(status)
	if !terminal {
		return
	}
	job.Status = next
	job.EndedAt = time.Now()
}

func (s *Session) setState(next State) {
	s.mu.Lock()
	prev := s.state
	if prev == next || !prev.Live() {
		s.mu.Unlock()
		return
	}
	s.state = next
	s.mu.Unlock()
	s.afterState(prev, next)
}

func (s *Session) setStateIf(from, next State) {
	s.mu.Lock()
	if s.state != from || from == next {
		s.mu.Unlock()
		return
	}
	s.state = next
	s.mu.Unlock()
	s.afterState(from, next)
}

func (s *Session) markSent() {
	s.mu.Lock()
	s.sent = true
	s.mu.Unlock()
}

func (s *Session) afterState(prev, next State) {
	if next == StateIdle || next == StateWaiting {
		select {
		case s.idleSig <- struct{}{}:
		default:
		}
	}
	s.emit(Event{Kind: KindState, State: next, Prev: prev})
}

func (s *Session) fail(err error) {
	s.mu.Lock()
	if s.err == nil {
		s.err = err
	}
	s.mu.Unlock()
	s.emit(Event{Kind: KindError, Err: err})
	s.setState(StateFailed)
}

func (s *Session) supervise() {
	s.readers.Wait()
	waitErr := s.cmd.Wait()

	s.mu.Lock()
	s.endedAt = time.Now()
	if waitErr != nil && s.err == nil {
		s.err = waitErr
	}
	failed := s.err != nil
	s.mu.Unlock()

	if failed {
		s.setState(StateFailed)
	} else {
		s.setState(StateExited)
	}

	s.cancel()
	<-s.writeDone
	s.tr.Close()
	close(s.events)
	close(s.done)
}

func (s *Session) emit(ev Event) {
	ev.Session = s.cfg.Name
	if ev.At.IsZero() {
		ev.At = time.Now()
	}
	ev.Snapshot = s.Snapshot()
	select {
	case s.events <- ev:
	case <-s.ctx.Done():
	}
}

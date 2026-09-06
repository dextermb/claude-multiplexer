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
	pendingBash     map[string]string
	pendingPath     map[string]string
	jobByToolUse    map[string]string

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

func (s *Session) ClaudeSessionID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.claudeSessionID
}

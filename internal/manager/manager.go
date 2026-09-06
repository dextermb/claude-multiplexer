package manager

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/mcp"
	"github.com/dextermb/claude-multiplexer/internal/protocol"
	"github.com/dextermb/claude-multiplexer/internal/render"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

const DefaultMaxLines = 5000

var (
	ErrUnknownSession = errors.New("manager: unknown session")
	ErrNoDirectory    = errors.New("manager: a session needs a directory")
	ErrNotDirectory   = errors.New("manager: the path is not a directory")
	ErrStillLive      = errors.New("manager: the session is still live")
	ErrUnknownJob     = errors.New("manager: unknown background job")
	ErrJobNotRunning  = errors.New("manager: the background job is not running")
)

type Options struct {
	Root                  string
	ConfigPaths           []string
	Renderer              render.Renderer
	MaxLines              int
	ClaudePath            string
	DefaultModel          string
	DefaultPermissionMode string
}

type Spec struct {
	Name           string
	Dir            string
	Model          string
	PermissionMode string
	Effort         string
	AllowedTools   []string
	ResumeID       string
	Control        bool
	Parent         string
}

type Event struct {
	Seq        uint64
	Session    string
	Kind       session.EventKind
	Lines      []render.Line
	Partial    string
	Snapshot   session.Snapshot
	Closed     bool
	Questions  []protocol.Question
	QuestionID string
	Todos      []protocol.Todo
	// Notice describes a change made outside the session stream, for example by
	// an MCP tool. Reload says the stored list changed. See docs/mcp/notices.md.
	Notice string
	Reload bool
}

type Manager struct {
	opts Options
	bus  *Bus
	mcp  *mcp.Server

	mu      sync.Mutex
	entries map[string]*entry
	order   []string
	pumps   sync.WaitGroup
}

func New(opts Options) (*Manager, error) {
	if opts.Root == "" {
		root, err := config.DefaultRoot()
		if err != nil {
			return nil, err
		}
		opts.Root = root
	}
	if opts.MaxLines <= 0 {
		opts.MaxLines = DefaultMaxLines
	}
	if opts.ClaudePath == "" {
		opts.ClaudePath = session.DefaultClaudePath
	}
	if opts.DefaultPermissionMode == "" {
		opts.DefaultPermissionMode = session.DefaultPermissionMode
	}
	return &Manager{
		opts:    opts,
		bus:     NewBus(),
		entries: make(map[string]*entry),
	}, nil
}

func (m *Manager) Subscribe(buffer int) *Subscription { return m.bus.Subscribe(buffer) }

func (m *Manager) Root() string { return m.opts.Root }

func (m *Manager) pump(item *entry) {
	defer m.pumps.Done()
	for ev := range item.sess.Events() {
		lines := m.opts.Renderer.Lines(ev)
		item.lines.append(lines)
		partial := trackPartial(item, ev)
		todos := trackTodos(item, ev)
		snap := ev.Snapshot
		item.setSnapshot(snap)
		m.rememberSession(item, snap)
		qid, questions, _ := ev.Protocol.AskUserQuestion()
		m.bus.Publish(Event{
			Session:    ev.Session,
			Kind:       ev.Kind,
			Lines:      lines,
			Partial:    partial,
			Snapshot:   snap,
			Questions:  questions,
			QuestionID: qid,
			Todos:      todos,
		})
	}
	m.releaseTools(item.token)
	final := item.sess.Snapshot()
	item.setSnapshot(final)
	name := item.metaCopy().Name
	if final.Turns == 0 {
		_ = os.RemoveAll(sessionDir(m.opts.Root, name))
	}
	m.bus.Publish(Event{
		Session:  name,
		Snapshot: final,
		Closed:   true,
	})
}

func trackPartial(item *entry, ev session.Event) string {
	item.partialMu.Lock()
	defer item.partialMu.Unlock()
	if ev.Kind != session.KindProtocol {
		return item.partial.String()
	}
	switch {
	case ev.Protocol.Type == protocol.TypeStreamEvent && ev.Protocol.Delta != nil:
		item.partial.WriteString(ev.Protocol.Delta.Text)
	case ev.Protocol.Type == protocol.TypeAssistant, ev.Protocol.Type == protocol.TypeResult:
		item.partial.Reset()
	}
	return item.partial.String()
}

func trackTodos(item *entry, ev session.Event) []protocol.Todo {
	item.todoMu.Lock()
	defer item.todoMu.Unlock()
	if ev.Kind == session.KindProtocol {
		item.tasks.Apply(ev.Protocol)
	}
	return item.tasks.List()
}

func (m *Manager) uniqueName(want, dir string, keep bool) string {
	base := want
	if base == "" {
		base = filepath.Base(dir)
	}
	if base == "" || base == "." || base == string(filepath.Separator) {
		base = "session"
	}
	name := base
	for i := 2; ; i++ {
		_, live := m.entries[name]
		if !live && (keep || !m.remembered(name)) {
			return name
		}
		name = fmt.Sprintf("%s-%d", base, i)
	}
}

func (m *Manager) remembered(name string) bool {
	_, err := os.Stat(metaPath(m.opts.Root, name))
	return err == nil
}

func removeName(names []string, name string) []string {
	out := names[:0]
	for _, item := range names {
		if item != name {
			out = append(out, item)
		}
	}
	return out
}

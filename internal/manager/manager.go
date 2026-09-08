package manager

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/dextermb/claude-multiplexer/internal/api"
	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/mcp"
	"github.com/dextermb/claude-multiplexer/internal/protocol"
	"github.com/dextermb/claude-multiplexer/internal/render"
	"github.com/dextermb/claude-multiplexer/internal/session"
	"github.com/dextermb/claude-multiplexer/internal/usage"
)

const DefaultMaxLines = 5000

var (
	ErrUnknownSession  = errors.New("manager: unknown session")
	ErrNoDirectory     = errors.New("manager: a session needs a directory")
	ErrNotDirectory    = errors.New("manager: the path is not a directory")
	ErrStillLive       = errors.New("manager: the session is still live")
	ErrUnknownJob      = errors.New("manager: unknown background job")
	ErrJobNotRunning   = errors.New("manager: the background job is not running")
	ErrNoCron          = errors.New("manager: a schedule needs a cron expression")
	ErrBadCron         = errors.New("manager: the cron expression is not valid")
	ErrNoPrompt        = errors.New("manager: a schedule needs a prompt")
	ErrUnknownSchedule = errors.New("manager: unknown schedule")
	ErrNoAPIStore      = errors.New("manager: the api is not started")
)

type Options struct {
	Root                  string
	ConfigPaths           []string
	Renderer              render.Renderer
	MaxLines              int
	ClaudePath            string
	DefaultModel          string
	DefaultPermissionMode string
	APIPortStart          int
	APIPortEnd            int
	// UsageFetch reads the Claude usage-limit headers for the poll. It is the one
	// integration seam left open: the exact endpoint and credential are settled
	// against a real account. A nil fetch keeps the poll off, so usage reads as
	// unknown. See docs/peers.md.
	UsageFetch usage.Fetch
}

type Spec struct {
	Name           string
	Dir            string
	Model          string
	PermissionMode string
	Effort         string
	Title          string
	AllowedTools   []string
	ResumeID       string
	Control        bool
	Parent         string
	Scheduled      string
	Owner          string
	Hosted         bool
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
	opts      Options
	bus       *Bus
	mcp       *mcp.Server
	apiStore  *api.Store
	usagePoll *usage.Poller

	usageStop func()

	mu          sync.Mutex
	entries     map[string]*entry
	order       []string
	remotes     map[string]*remoteEntry
	remoteOrder []string
	pumps       sync.WaitGroup

	schedMu   sync.Mutex
	schedules map[string]*Schedule
	schedStop chan struct{}
	schedWG   sync.WaitGroup
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
	if opts.APIPortStart <= 0 {
		opts.APIPortStart = mcp.DefaultAPIPortStart
	}
	if opts.APIPortEnd < opts.APIPortStart {
		opts.APIPortEnd = mcp.DefaultAPIPortEnd
	}
	m := &Manager{
		opts:      opts,
		bus:       NewBus(),
		entries:   make(map[string]*entry),
		remotes:   make(map[string]*remoteEntry),
		schedules: make(map[string]*Schedule),
	}
	m.loadSchedules()
	return m, nil
}

func (m *Manager) Subscribe(buffer int) *Subscription { return m.bus.Subscribe(buffer) }

func (m *Manager) Root() string { return m.opts.Root }

func (m *Manager) apiPortRange() (int, int) { return m.opts.APIPortStart, m.opts.APIPortEnd }

func (m *Manager) pump(item *entry) {
	defer m.pumps.Done()
	enterWorktree := map[string]string{}
	exitWorktree := map[string]bool{}
	for ev := range item.sess.Events() {
		m.trackWorktree(ev, enterWorktree, exitWorktree)
		lines := item.skill.Track(ev.Protocol, m.opts.Renderer.Lines(ev))
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

// trackWorktree sets the working directory of a session from the EnterWorktree
// tool it runs, and clears it on ExitWorktree. The path comes from the
// tool_result, and falls back to the tool input. The two maps hold the pending
// tool ids, and belong to one pump goroutine. See docs/mcp/tools.md.
func (m *Manager) trackWorktree(ev session.Event, enter map[string]string, exit map[string]bool) {
	if ev.Kind != session.KindProtocol || ev.Protocol.Message == nil {
		return
	}
	for _, block := range ev.Protocol.Message.Content {
		switch block.Type {
		case "tool_use":
			if block.ID == "" {
				continue
			}
			if path, ok := block.EnterWorktree(); ok {
				enter[block.ID] = path
			} else if block.ExitWorktree() {
				exit[block.ID] = true
			}
		case "tool_result":
			id := block.ToolUseID
			input, isEnter := enter[id]
			_, isExit := exit[id]
			if !isEnter && !isExit {
				continue
			}
			delete(enter, id)
			delete(exit, id)
			if block.IsError {
				continue
			}
			if isEnter {
				path := protocol.WorktreePath(block.Content.Text())
				if path == "" {
					path = input
				}
				if path != "" {
					_, _ = m.SetWorkingDir(ev.Session, path)
				}
			} else {
				_, _ = m.UnsetWorkingDir(ev.Session)
			}
		}
	}
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
		_, remote := m.remotes[name]
		if !live && !remote && (keep || !m.remembered(name)) {
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

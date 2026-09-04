package manager

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/mcp"
	"github.com/dextermb/claude-multiplexer/internal/protocol"
	"github.com/dextermb/claude-multiplexer/internal/render"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

const (
	DefaultMaxLines = 5000
	DefaultRootName = ".multiplexier"
)

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
	// an MCP tool. Reload says the stored list changed. See docs/mcp.md.
	Notice string
	Reload bool
}

type entry struct {
	sess    *session.Session
	lines   *lineBuffer
	meta    Meta
	path    string
	base    totals
	token   string
	control bool

	partialMu sync.Mutex
	partial   strings.Builder

	todoMu sync.Mutex
	todos  []protocol.Todo

	snapMu sync.Mutex
	snap   session.Snapshot
}

func (e *entry) partialText() string {
	e.partialMu.Lock()
	defer e.partialMu.Unlock()
	return e.partial.String()
}

func (e *entry) setSnapshot(snap session.Snapshot) {
	e.snapMu.Lock()
	defer e.snapMu.Unlock()
	e.snap = snap
}

// view returns the live session snapshot, but with the totals the stream feeds
// taken from the last snapshot the pump saw. So the turn count never leads the
// buffered lines, and the title, the queue, and the state stay live. See
// docs/manager.md.
func (e *entry) view() session.Snapshot {
	live := e.sess.Snapshot()
	e.snapMu.Lock()
	cached := e.snap
	e.snapMu.Unlock()
	live.Turns = cached.Turns
	live.Cost = cached.Cost
	live.InputTokens = cached.InputTokens
	live.OutputTokens = cached.OutputTokens
	live.ContextTokens = cached.ContextTokens
	live.LastDuration = cached.LastDuration
	return live
}

func (e *entry) todoList() []protocol.Todo {
	e.todoMu.Lock()
	defer e.todoMu.Unlock()
	return e.todos
}

type totals struct {
	turns  int
	cost   float64
	input  int
	output int
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
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		opts.Root = filepath.Join(home, DefaultRootName)
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

func (m *Manager) Spawn(ctx context.Context, spec Spec) (string, error) {
	if spec.Dir == "" {
		return "", ErrNoDirectory
	}
	dir, err := filepath.Abs(spec.Dir)
	if err != nil {
		return "", err
	}
	if spec.Model == "" {
		spec.Model = m.opts.DefaultModel
	}
	if spec.PermissionMode == "" {
		spec.PermissionMode = m.opts.DefaultPermissionMode
	}

	m.mu.Lock()
	name := m.uniqueName(spec.Name, dir, spec.ResumeID != "")
	m.mu.Unlock()

	cfg := session.Config{
		Name:           name,
		Dir:            dir,
		Model:          spec.Model,
		PermissionMode: spec.PermissionMode,
		Effort:         spec.Effort,
		AllowedTools:   spec.AllowedTools,
		ResumeID:       spec.ResumeID,
		ClaudePath:     m.opts.ClaudePath,
		ReplayPrompts:  true,
		IncludePartial: true,
		TranscriptPath: transcriptPath(m.opts.Root, name),
	}
	token, err := m.equipTools(&cfg, name, spec.Control)
	if err != nil {
		return "", err
	}

	sess, err := session.New(cfg)
	if err != nil {
		m.releaseTools(token)
		return "", err
	}

	item := &entry{
		sess:    sess,
		lines:   newLineBuffer(m.opts.MaxLines),
		path:    metaPath(m.opts.Root, name),
		token:   token,
		control: spec.Control,
		meta: Meta{
			Name:           name,
			Dir:            dir,
			Parent:         spec.Parent,
			Model:          spec.Model,
			PermissionMode: spec.PermissionMode,
			Effort:         spec.Effort,
			Control:        spec.Control,
			CreatedAt:      time.Now(),
		},
	}
	item.snap = sess.Snapshot()

	m.mu.Lock()
	m.entries[name] = item
	m.order = append(m.order, name)
	m.mu.Unlock()

	if spec.ResumeID != "" {
		item.lines.append(m.Replay(name))
		item.todos = m.todosFromTranscript(name)
		item.lines.append([]render.Line{{Class: render.ClassMeta, Text: "— resumed —"}})
		if stored, err := ReadMeta(item.path); err == nil {
			item.meta = stored
			item.meta.Archived = false
			item.base = totals{
				turns:  stored.Turns,
				cost:   stored.Cost,
				input:  stored.InputTokens,
				output: stored.OutputTokens,
			}
		}
	}

	if err := sess.Start(ctx); err != nil {
		m.mu.Lock()
		delete(m.entries, name)
		m.order = removeName(m.order, name)
		m.mu.Unlock()
		m.releaseTools(token)
		return "", err
	}

	m.pumps.Add(1)
	go m.pump(item)
	return name, nil
}

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
	if final.Turns == 0 {
		_ = os.RemoveAll(sessionDir(m.opts.Root, item.meta.Name))
	}
	m.bus.Publish(Event{
		Session:  item.meta.Name,
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
		if list, ok := ev.Protocol.TodoWrite(); ok {
			item.todos = list
		}
	}
	return item.todos
}

func (m *Manager) rememberSession(item *entry, snap session.Snapshot) {
	if snap.Turns == 0 {
		return
	}
	next := item.meta
	next.ClaudeSessionID = snap.ClaudeSessionID
	next.Title = snap.Title
	next.Model = snap.Model
	next.PermissionMode = snap.PermissionMode
	next.Effort = snap.Effort
	next.Control = item.control
	next.Turns = item.base.turns + snap.Turns
	next.Cost = item.base.cost + snap.Cost
	next.InputTokens = item.base.input + snap.InputTokens
	next.OutputTokens = item.base.output + snap.OutputTokens
	if next.sameAs(item.meta) && !item.meta.LastActiveAt.IsZero() {
		return
	}
	next.LastActiveAt = time.Now()
	item.meta = next
	_ = writeMeta(item.path, item.meta)
}

func (m *Manager) Stored() []Meta {
	metas, err := scanMeta(m.opts.Root)
	if err != nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Meta, 0, len(metas))
	for _, meta := range metas {
		if _, live := m.entries[meta.Name]; live {
			continue
		}
		out = append(out, meta)
	}
	return out
}

func (m *Manager) Replay(name string) []render.Line {
	file, err := os.Open(transcriptPath(m.opts.Root, name))
	if err != nil {
		return nil
	}
	defer file.Close()

	reader := protocol.NewReader(file)
	var lines []render.Line
	for {
		ev, err := reader.Next()
		if errors.Is(err, protocol.ErrNotJSON) {
			continue
		}
		if err != nil {
			break
		}
		lines = append(lines, m.opts.Renderer.Lines(session.Event{
			Kind:     session.KindProtocol,
			Protocol: ev,
		})...)
	}
	if len(lines) > m.opts.MaxLines {
		lines = lines[len(lines)-m.opts.MaxLines:]
	}
	return lines
}

// Todos returns the current task list of a session. A live session holds the
// list in memory; a stored session gets it rebuilt from the transcript, which
// is the last TodoWrite in the stream. See docs/tui/tasks.md.
func (m *Manager) Todos(name string) []protocol.Todo {
	if item, err := m.entry(name); err == nil {
		return item.todoList()
	}
	return m.todosFromTranscript(name)
}

func (m *Manager) todosFromTranscript(name string) []protocol.Todo {
	file, err := os.Open(transcriptPath(m.opts.Root, name))
	if err != nil {
		return nil
	}
	defer file.Close()

	reader := protocol.NewReader(file)
	var todos []protocol.Todo
	for {
		ev, err := reader.Next()
		if errors.Is(err, protocol.ErrNotJSON) {
			continue
		}
		if err != nil {
			break
		}
		if list, ok := ev.TodoWrite(); ok {
			todos = list
		}
	}
	return todos
}

func (m *Manager) Archive(name string, archived bool) error {
	m.mu.Lock()
	item, live := m.entries[name]
	m.mu.Unlock()
	if live && item.sess.State().Live() {
		return fmt.Errorf("%w: %s", ErrStillLive, name)
	}

	path := metaPath(m.opts.Root, name)
	meta, err := ReadMeta(path)
	if err != nil {
		return err
	}
	meta.Archived = archived
	if archived {
		meta.ArchivedAt = time.Now()
	} else {
		meta.ArchivedAt = time.Time{}
	}
	if err := writeMeta(path, meta); err != nil {
		return err
	}

	if live {
		m.mu.Lock()
		delete(m.entries, name)
		m.order = removeName(m.order, name)
		m.mu.Unlock()
	}
	return nil
}

func (m *Manager) Meta(name string) (Meta, error) {
	return ReadMeta(metaPath(m.opts.Root, name))
}

func (m *Manager) Resume(ctx context.Context, meta Meta) (string, error) {
	if meta.ClaudeSessionID == "" {
		return "", fmt.Errorf("manager: session %q has no Claude session id", meta.Name)
	}
	if err := m.Remove(meta.Name); err != nil && !errors.Is(err, ErrUnknownSession) {
		return "", err
	}
	return m.Spawn(ctx, Spec{
		Name:           meta.Name,
		Dir:            meta.Dir,
		Model:          meta.Model,
		PermissionMode: meta.PermissionMode,
		Effort:         meta.Effort,
		Control:        meta.Control,
		Parent:         meta.Parent,
		ResumeID:       meta.ClaudeSessionID,
	})
}

func (m *Manager) Send(name, text string) error {
	item, err := m.entry(name)
	if err != nil {
		return err
	}
	return item.sess.Send(text)
}

func (m *Manager) Interrupt(name string, discardQueued bool) error {
	item, err := m.entry(name)
	if err != nil {
		return err
	}
	if discardQueued {
		item.sess.DiscardQueued()
	}
	return item.sess.Interrupt()
}

func (m *Manager) SetModel(name, model string) error {
	item, err := m.entry(name)
	if err != nil {
		return err
	}
	return item.sess.SetModel(model)
}

func (m *Manager) SetPermissionMode(name, mode string) error {
	item, err := m.entry(name)
	if err != nil {
		return err
	}
	return item.sess.SetPermissionMode(mode)
}

// SetTitle renames a session. A live session takes the new title and persists it
// on its next turn; a stored session gets it written straight to its meta.
func (m *Manager) SetTitle(name, title string) error {
	m.mu.Lock()
	item, live := m.entries[name]
	m.mu.Unlock()
	if live {
		item.sess.SetTitle(title)
		return nil
	}
	path := metaPath(m.opts.Root, name)
	meta, err := ReadMeta(path)
	if err != nil {
		return err
	}
	meta.Title = title
	return writeMeta(path, meta)
}

// ResumeWithEffort stops a running session and resumes it with a new effort
// level, because Claude Code has no live effort switch; see docs/protocol.md.
func (m *Manager) ResumeWithEffort(ctx context.Context, name, effort string) (string, error) {
	item, err := m.entry(name)
	if err != nil {
		return "", err
	}
	snap := item.sess.Snapshot()
	if snap.ClaudeSessionID == "" {
		return "", fmt.Errorf("manager: session %q has not started a turn yet", name)
	}
	stopCtx, cancel := context.WithTimeout(ctx, session.DefaultStopGrace)
	_ = item.sess.Stop(stopCtx)
	cancel()
	return m.Resume(ctx, Meta{
		Name:            snap.Name,
		Dir:             snap.Dir,
		Model:           snap.Model,
		PermissionMode:  snap.PermissionMode,
		Effort:          effort,
		Control:         item.control,
		Parent:          item.meta.Parent,
		ClaudeSessionID: snap.ClaudeSessionID,
	})
}

func (m *Manager) Stop(ctx context.Context, name string) error {
	item, err := m.entry(name)
	if err != nil {
		return err
	}
	return item.sess.Stop(ctx)
}

func (m *Manager) Remove(name string) error {
	item, err := m.entry(name)
	if err != nil {
		return err
	}
	if item.sess.State().Live() {
		return fmt.Errorf("%w: %s", ErrStillLive, name)
	}
	m.mu.Lock()
	delete(m.entries, name)
	m.order = removeName(m.order, name)
	m.mu.Unlock()
	return nil
}

func (m *Manager) Shutdown(ctx context.Context) {
	m.mu.Lock()
	items := make([]*entry, 0, len(m.order))
	for _, name := range m.order {
		items = append(items, m.entries[name])
	}
	m.mu.Unlock()

	var wg sync.WaitGroup
	for _, item := range items {
		wg.Add(1)
		go func(item *entry) {
			defer wg.Done()
			_ = item.sess.Stop(ctx)
		}(item)
	}
	wg.Wait()
	m.pumps.Wait()
	if m.mcp != nil {
		_ = m.mcp.Close(ctx)
	}
}

func (m *Manager) Names() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.order))
	copy(out, m.order)
	return out
}

func (m *Manager) Snapshots() []session.Snapshot {
	m.mu.Lock()
	items := make([]*entry, 0, len(m.order))
	for _, name := range m.order {
		items = append(items, m.entries[name])
	}
	m.mu.Unlock()

	out := make([]session.Snapshot, 0, len(items))
	for _, item := range items {
		out = append(out, item.view())
	}
	return out
}

func (m *Manager) Snapshot(name string) (session.Snapshot, error) {
	item, err := m.entry(name)
	if err != nil {
		return session.Snapshot{}, err
	}
	return item.view(), nil
}

func (m *Manager) Lines(name string) []render.Line {
	item, err := m.entry(name)
	if err != nil {
		return nil
	}
	return item.lines.all()
}

func (m *Manager) AppendLines(name string, lines []render.Line) error {
	item, err := m.entry(name)
	if err != nil {
		return err
	}
	item.lines.append(lines)
	return nil
}

func (m *Manager) Stderr(name string) []string {
	item, err := m.entry(name)
	if err != nil {
		return nil
	}
	return item.sess.Stderr()
}

func (m *Manager) TotalCost() float64 {
	var total float64
	for _, snap := range m.Snapshots() {
		total += snap.Cost
	}
	return total
}

func (m *Manager) entry(name string) (*entry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, ok := m.entries[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownSession, name)
	}
	return item, nil
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

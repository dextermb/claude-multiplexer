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
	ErrStillLive      = errors.New("manager: the session is still live")
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
	AllowedTools   []string
	ResumeID       string
}

type Event struct {
	Seq      uint64
	Session  string
	Kind     session.EventKind
	Lines    []render.Line
	Partial  string
	Snapshot session.Snapshot
	Closed   bool
}

type entry struct {
	sess    *session.Session
	lines   *lineBuffer
	meta    Meta
	path    string
	base    totals
	partial strings.Builder
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
		AllowedTools:   spec.AllowedTools,
		ResumeID:       spec.ResumeID,
		ClaudePath:     m.opts.ClaudePath,
		ReplayPrompts:  true,
		IncludePartial: true,
		TranscriptPath: transcriptPath(m.opts.Root, name),
	}
	sess, err := session.New(cfg)
	if err != nil {
		return "", err
	}

	item := &entry{
		sess:  sess,
		lines: newLineBuffer(m.opts.MaxLines),
		path:  metaPath(m.opts.Root, name),
		meta: Meta{
			Name:           name,
			Dir:            dir,
			Model:          spec.Model,
			PermissionMode: spec.PermissionMode,
			CreatedAt:      time.Now(),
		},
	}

	m.mu.Lock()
	m.entries[name] = item
	m.order = append(m.order, name)
	m.mu.Unlock()

	if spec.ResumeID != "" {
		item.lines.append(m.Replay(name))
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
		snap := item.sess.Snapshot()
		m.rememberSession(item, snap)
		m.bus.Publish(Event{
			Session:  ev.Session,
			Kind:     ev.Kind,
			Lines:    lines,
			Partial:  partial,
			Snapshot: snap,
		})
	}
	final := item.sess.Snapshot()
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

func (m *Manager) rememberSession(item *entry, snap session.Snapshot) {
	if snap.Turns == 0 {
		return
	}
	next := item.meta
	next.ClaudeSessionID = snap.ClaudeSessionID
	next.Model = snap.Model
	next.PermissionMode = snap.PermissionMode
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
		out = append(out, item.sess.Snapshot())
	}
	return out
}

func (m *Manager) Snapshot(name string) (session.Snapshot, error) {
	item, err := m.entry(name)
	if err != nil {
		return session.Snapshot{}, err
	}
	return item.sess.Snapshot(), nil
}

func (m *Manager) Lines(name string) []render.Line {
	item, err := m.entry(name)
	if err != nil {
		return nil
	}
	return item.lines.all()
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

package manager

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/mcp"
	"github.com/dextermb/claude-multiplexer/internal/protocol"
	"github.com/dextermb/claude-multiplexer/internal/render"
	"github.com/dextermb/claude-multiplexer/internal/session"
	"github.com/dextermb/claude-multiplexer/internal/template"
)

// StartMCP starts the server that serves the multiplexer's own tools to each
// session. Call it before the first Spawn, because a session that starts
// earlier never learns the address. See docs/mcp/transport.md.
func (m *Manager) StartMCP() error {
	if m.mcp != nil {
		return nil
	}
	server := mcp.NewServer(&bridge{m: m})
	if err := server.Start(); err != nil {
		return err
	}
	m.mcp = server
	return nil
}

func (m *Manager) MCPURL() string {
	if m.mcp == nil {
		return ""
	}
	return m.mcp.URL()
}

// equipTools gives one session its token, its configuration file, and the tool
// names it may call.
func (m *Manager) equipTools(cfg *session.Config, name string, control bool) (string, error) {
	if m.mcp == nil {
		return "", nil
	}
	token, err := m.mcp.Register(name, control)
	if err != nil {
		return "", err
	}
	path := mcpConfigPath(m.opts.Root, name)
	data, err := m.mcp.Config(token)
	if err != nil {
		m.mcp.Unregister(token)
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		m.mcp.Unregister(token)
		return "", err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		m.mcp.Unregister(token)
		return "", err
	}
	cfg.ExtraArgs = append(cfg.ExtraArgs, "--mcp-config", path)
	cfg.AllowedTools = append(append([]string{}, cfg.AllowedTools...), mcp.AllowedTools(control)...)
	return token, nil
}

func (m *Manager) releaseTools(token string) {
	if m.mcp == nil || token == "" {
		return
	}
	m.mcp.Unregister(token)
}

// SendFrom queues a prompt that came from another session, and marks the pane
// with the name of the sender, so the human sees who asked.
func (m *Manager) SendFrom(target, from, text string) (int, error) {
	item, err := m.entry(target)
	if err != nil {
		return 0, err
	}
	lines := []render.Line{{Class: render.ClassMeta, Text: "← prompt from " + from}}
	item.lines.append(lines)
	if err := item.sess.Send(text); err != nil {
		return 0, err
	}
	snap := item.sess.Snapshot()
	m.bus.Publish(Event{
		Session:  target,
		Kind:     session.KindState,
		Lines:    lines,
		Partial:  item.partialText(),
		Snapshot: snap,
	})
	return snap.Queued, nil
}

// Jobs reports the background jobs of a live session. A stored session has no
// running jobs, so it returns an empty list. See docs/mcp/tools.md.
func (m *Manager) Jobs(name string) ([]mcp.Job, error) {
	item, err := m.entry(name)
	if err != nil {
		if _, metaErr := m.Meta(name); metaErr == nil {
			return nil, nil
		}
		return nil, err
	}
	jobs := item.sess.Snapshot().Jobs
	out := make([]mcp.Job, 0, len(jobs))
	for _, job := range jobs {
		out = append(out, mcp.Job{
			ID:          job.ID,
			Description: job.Description,
			TaskType:    job.TaskType,
			Status:      job.Status.String(),
			Running:     job.Status.Running(),
		})
	}
	return out, nil
}

// StopJobFrom interrupts a session and queues an instruction to kill one
// background job, so the agent runs KillShell on the exact shell. The interrupt
// ends the turn at once, so the instruction runs next. See docs/mcp/tools.md.
func (m *Manager) StopJobFrom(target, from, jobID string) (int, error) {
	item, err := m.entry(target)
	if err != nil {
		return 0, err
	}
	job, ok := findJob(item.sess.Snapshot().Jobs, jobID)
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrUnknownJob, jobID)
	}
	if !job.Status.Running() {
		return 0, fmt.Errorf("%w: %s", ErrJobNotRunning, jobID)
	}
	lines := []render.Line{{Class: render.ClassMeta, Text: "← stop job " + jobID + " from " + from}}
	item.lines.append(lines)
	if err := item.sess.Send(killPrompt(job)); err != nil {
		return 0, err
	}
	_ = item.sess.Interrupt()
	snap := item.sess.Snapshot()
	m.bus.Publish(Event{
		Session:  target,
		Kind:     session.KindState,
		Lines:    lines,
		Partial:  item.partialText(),
		Snapshot: snap,
	})
	return snap.Queued, nil
}

func findJob(jobs []session.Job, id string) (session.Job, bool) {
	for _, job := range jobs {
		if job.ID == id {
			return job, true
		}
	}
	return session.Job{}, false
}

func killPrompt(job session.Job) string {
	text := "Stop the background job now. Use the KillShell tool with shell_id " + strconv.Quote(job.ID) + "."
	if job.Description != "" {
		text += " The job is: " + job.Description + "."
	}
	return text
}

// List describes every session the manager knows, live rows first.
func (m *Manager) List() []mcp.Session {
	m.mu.Lock()
	items := make([]*entry, 0, len(m.order))
	for _, name := range m.order {
		items = append(items, m.entries[name])
	}
	m.mu.Unlock()

	out := make([]mcp.Session, 0, len(items))
	for _, item := range items {
		snap := item.sess.Snapshot()
		out = append(out, mcp.Session{
			Name:    snap.Name,
			Title:   snap.Title,
			Dir:     snap.Dir,
			State:   snap.State.String(),
			Model:   snap.Model,
			Live:    true,
			Control: item.control,
			Queued:  snap.Queued,
			Turns:   snap.Turns,
			Cost:    snap.Cost,
		})
	}
	for _, meta := range m.Stored() {
		state := "stored"
		if meta.Archived {
			state = "archived"
		}
		out = append(out, mcp.Session{
			Name:     meta.Name,
			Title:    meta.Title,
			Dir:      meta.Dir,
			State:    state,
			Model:    meta.Model,
			Archived: meta.Archived,
			Control:  meta.Control,
			Turns:    meta.Turns,
			Cost:     meta.Cost,
		})
	}
	return out
}

// Grants reports which live sessions may drive their neighbours.
// Parents reports the control session that created each live session, for the
// sessions that a control session created. See docs/manager.md.
func (m *Manager) Parents() map[string]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]string, len(m.entries))
	for name, item := range m.entries {
		if parent := item.metaCopy().Parent; parent != "" {
			out[name] = parent
		}
	}
	return out
}

// WorkingDirs reports the working directory of each live session that set one.
func (m *Manager) WorkingDirs() map[string]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]string, len(m.entries))
	for name, item := range m.entries {
		if dir := item.metaCopy().WorkingDir; dir != "" {
			out[name] = dir
		}
	}
	return out
}

func (m *Manager) Grants() map[string]bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]bool, len(m.entries))
	for name, item := range m.entries {
		out[name] = item.control
	}
	return out
}

// Messages reads the recent conversation of a session from its transcript, so
// it works for a stored session as well as a live one.
func (m *Manager) Messages(name string, limit int) ([]mcp.Message, error) {
	file, err := os.Open(transcriptPath(m.opts.Root, name))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnknownSession, name)
	}
	defer file.Close()

	reader := protocol.NewReader(file)
	var out []mcp.Message
	for {
		ev, err := reader.Next()
		if errors.Is(err, protocol.ErrNotJSON) {
			continue
		}
		if err != nil {
			break
		}
		if message, ok := messageOf(ev); ok {
			out = append(out, message)
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out, nil
}

func messageOf(ev protocol.Event) (mcp.Message, bool) {
	switch ev.Type {
	case protocol.TypeUser:
		text := ev.Text()
		if text == "" {
			return mcp.Message{}, false
		}
		return mcp.Message{Role: "user", Text: text}, true
	case protocol.TypeAssistant:
		text := assistantText(ev)
		if text == "" {
			return mcp.Message{}, false
		}
		return mcp.Message{Role: "assistant", Text: text}, true
	case protocol.TypeResult:
		if ev.Result == nil {
			return mcp.Message{}, false
		}
		text := ev.Result.Result
		if text == "" {
			text = ev.Result.Subtype
		}
		return mcp.Message{Role: "result", Text: text}, true
	}
	return mcp.Message{}, false
}

// assistantText keeps the words and names each tool the model used, because the
// whole input of a tool call is large and it is rarely what a reader wants.
func assistantText(ev protocol.Event) string {
	var parts []string
	if text := ev.Text(); text != "" {
		parts = append(parts, text)
	}
	if ev.Message != nil {
		for _, block := range ev.Message.Content {
			if block.Type == "tool_use" && block.Name != "" {
				parts = append(parts, "[used "+block.Name+"]")
			}
		}
	}
	return strings.Join(parts, "\n")
}

// SetEditor writes the editor settings, so a session can name the editor the
// human opens a directory with. See docs/config.md.
func (m *Manager) SetEditor(editor string, terminal *bool) (string, error) {
	path := config.Target(m.opts.ConfigPaths...)
	if path == "" {
		return "", errors.New("manager: no settings file to write")
	}
	current, err := config.Load(path)
	if err != nil {
		return "", err
	}
	if editor != "" {
		current.Editor = editor
	}
	if terminal != nil {
		current.EditorTerminal = terminal
	}
	if err := config.Write(path, current); err != nil {
		return "", err
	}
	return path, nil
}

// UnsetEditor takes the editor settings out of the settings file. It reports
// whether the file held anything to take out. See docs/config.md.
func (m *Manager) UnsetEditor(field string) (string, bool, error) {
	path := config.Target(m.opts.ConfigPaths...)
	if path == "" {
		return "", false, errors.New("manager: no settings file to write")
	}
	current, err := config.Load(path)
	if err != nil {
		return "", false, err
	}
	next, err := config.Clear(current, field)
	if err != nil {
		return "", false, err
	}
	if config.SameEditor(next, current) {
		return path, false, nil
	}
	if err := config.Write(path, next); err != nil {
		return "", false, err
	}
	return path, true, nil
}

// ConfigPath names the settings files, in the order they are read. See
// docs/config.md.
func (m *Manager) ConfigPath() mcp.ConfigPath {
	paths := append([]string(nil), m.opts.ConfigPaths...)
	return mcp.ConfigPath{
		Paths:  paths,
		Active: config.Active(paths...),
		Target: config.Target(paths...),
	}
}

// TemplatePath names the directories a session reads a preset prompt from. The
// directory is the one the session started in, which is the one the interface
// reads. See docs/templates.md.
func (m *Manager) TemplatePath(name string) (mcp.TemplatePath, error) {
	dir, err := m.sessionDir(name)
	if err != nil {
		return mcp.TemplatePath{}, err
	}
	return mcp.TemplatePath{
		Session: name,
		Root:    m.opts.Root,
		Dir:     dir,
		Dirs:    template.Dirs(m.opts.Root, dir),
	}, nil
}

// sessionDir gives the directory a session started in, from the running child
// when it is live, and from the stored meta when it is not.
func (m *Manager) sessionDir(name string) (string, error) {
	item, err := m.entry(name)
	if err == nil {
		return item.sess.Snapshot().Dir, nil
	}
	meta, metaErr := m.Meta(name)
	if metaErr != nil {
		return "", err
	}
	return meta.Dir, nil
}

// SetBlockCap writes the rows a block draws before the pane caps it, so a
// session can change how much of a large result the human sees. An empty bucket
// sets the default for every type; a bucket sets that one type. A nil rows caps
// nothing (a bucket only). See docs/config.md.
func (m *Manager) SetBlockCap(bucket string, rows *int) (string, error) {
	if rows != nil && *rows < 0 {
		return "", errors.New("manager: the block cap must be zero or more")
	}
	if bucket == "" && rows == nil {
		return "", errors.New("manager: the default block cap needs a row count")
	}
	if bucket != "" && !config.ValidBucket(bucket) {
		return "", errors.New("manager: unknown block type " + bucket)
	}
	path := config.Target(m.opts.ConfigPaths...)
	if path == "" {
		return "", errors.New("manager: no settings file to write")
	}
	current, err := config.Load(path)
	if err != nil {
		return "", err
	}
	if bucket == "" {
		current.BlockCap = rows
	} else {
		if current.BlockCaps == nil {
			current.BlockCaps = map[string]*int{}
		}
		current.BlockCaps[bucket] = rows
	}
	if err := config.Write(path, current); err != nil {
		return "", err
	}
	return path, nil
}

// UnsetBlockCap takes a block cap out of the settings file. An empty bucket
// clears the default, so the pane returns to config.DefaultBlockCap; a bucket
// clears that type, so it takes the default again. It reports whether the file
// held the cap. See docs/config.md.
func (m *Manager) UnsetBlockCap(bucket string) (string, bool, error) {
	if bucket != "" && !config.ValidBucket(bucket) {
		return "", false, errors.New("manager: unknown block type " + bucket)
	}
	path := config.Target(m.opts.ConfigPaths...)
	if path == "" {
		return "", false, errors.New("manager: no settings file to write")
	}
	current, err := config.Load(path)
	if err != nil {
		return "", false, err
	}
	if bucket == "" {
		if current.BlockCap == nil {
			return path, false, nil
		}
		current.BlockCap = nil
	} else {
		if _, ok := current.BlockCaps[bucket]; !ok {
			return path, false, nil
		}
		delete(current.BlockCaps, bucket)
		if len(current.BlockCaps) == 0 {
			current.BlockCaps = nil
		}
	}
	if err := config.Write(path, current); err != nil {
		return "", false, err
	}
	return path, true, nil
}

// notify publishes a change that no session event follows, so the interface
// learns of it without a timer. See docs/mcp/notices.md.
func (m *Manager) notify(name, notice string, reload bool) {
	m.bus.Publish(Event{Session: name, Notice: notice, Reload: reload})
}

package manager

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/render"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

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
		item.tasks = m.tasksFromTranscript(name)
		item.lines.append([]render.Line{{Class: render.ClassMeta, Text: "— resumed —"}})
		if stored, err := ReadMeta(item.path); err == nil {
			stored.Archived = false
			item.setMeta(stored)
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

// SetWorkingDir points a session at the directory a tool of that session names,
// so the interface opens it instead of the directory the session started in. A
// relative path is resolved against that directory. See docs/mcp/tools.md.
func (m *Manager) SetWorkingDir(name, path string) (string, error) {
	item, err := m.entry(name)
	if err != nil {
		return "", err
	}
	meta := item.metaCopy()
	full := path
	if !filepath.IsAbs(full) {
		full = filepath.Join(meta.Dir, full)
	}
	full = filepath.Clean(full)
	info, err := os.Stat(full)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s: not a directory", full)
	}
	meta.WorkingDir = full
	item.setMeta(meta)
	if err := writeMeta(item.path, meta); err != nil {
		return "", err
	}
	return full, nil
}

// UnsetWorkingDir takes the working directory off a session, and reports
// whether it had one.
func (m *Manager) UnsetWorkingDir(name string) (bool, error) {
	item, err := m.entry(name)
	if err != nil {
		return false, err
	}
	meta := item.metaCopy()
	if meta.WorkingDir == "" {
		return false, nil
	}
	meta.WorkingDir = ""
	item.setMeta(meta)
	if err := writeMeta(item.path, meta); err != nil {
		return false, err
	}
	return true, nil
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
		Parent:          item.metaCopy().Parent,
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

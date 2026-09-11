package manager

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
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
		Title:          spec.Title,
		AllowedTools:   spec.AllowedTools,
		ResumeID:       spec.ResumeID,
		ClaudePath:     m.opts.ClaudePath,
		ReplayPrompts:  true,
		IncludePartial: true,
		TranscriptPath: transcriptPath(m.opts.Root, name),
	}
	if spec.Lender != "" {
		if err := m.hoist(&cfg, spec.Lender); err != nil {
			return "", err
		}
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
			Scheduled:      spec.Scheduled,
			Owner:          spec.Owner,
			Hosted:         spec.Hosted,
			Lender:         spec.Lender,
			TempDir:        spec.TempDir,
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
	if _, err := m.evict(meta.Name); err != nil && !errors.Is(err, ErrUnknownSession) {
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
		Title:          meta.Title,
		ResumeID:       meta.ClaudeSessionID,
	})
}

func (m *Manager) Send(name, text string) error {
	if re := m.remote(name); re != nil {
		if re.readOnly {
			return ErrReadOnly
		}
		return re.client.Send(context.Background(), re.remoteName, text)
	}
	item, err := m.entry(name)
	if err != nil {
		return err
	}
	return item.sess.Send(text)
}

// SendFrom queues a prompt that came from another session, and marks the pane
// with the name of the sender, so the human sees who asked.
func (m *Manager) SendFrom(target, from, text string) (int, error) {
	item, err := m.entry(target)
	if err != nil {
		return 0, err
	}
	lines := []render.Line{{Class: render.ClassMeta, Text: "← prompt from " + from}}
	if err := item.sess.Send(text); err != nil {
		return 0, err
	}
	snap := item.sess.Snapshot()
	m.bus.publishLines(item.lines, lines, Event{
		Session:  target,
		Kind:     session.KindState,
		Lines:    lines,
		Partial:  item.partialText(),
		Snapshot: snap,
	})
	return snap.Queued, nil
}

func (m *Manager) Interrupt(name string, discardQueued bool) error {
	if re := m.remote(name); re != nil {
		if re.readOnly {
			return ErrReadOnly
		}
		return re.client.Interrupt(context.Background(), re.remoteName)
	}
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
	full, err := resolveDir(meta.Dir, path)
	if err != nil {
		return "", err
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

// resolveDir turns a path a tool gives into an absolute, clean directory. A
// relative path is resolved against base. The directory must exist.
func resolveDir(base, path string) (string, error) {
	full := path
	if !filepath.IsAbs(full) {
		full = filepath.Join(base, full)
	}
	full = filepath.Clean(full)
	info, err := os.Stat(full)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s: not a directory", full)
	}
	return full, nil
}

// Project reads the directories of a session's project, in order. The list is
// empty when the session has no project. See docs/mcp/tools.md.
func (m *Manager) Project(name string) ([]string, error) {
	item, err := m.entry(name)
	if err != nil {
		return nil, err
	}
	return item.metaCopy().WorkingDirs, nil
}

// SetProject replaces the whole ordered set of project directories. Every path
// is resolved and validated, and the result is de-duplicated in order.
func (m *Manager) SetProject(name string, paths []string) ([]string, error) {
	item, err := m.entry(name)
	if err != nil {
		return nil, err
	}
	meta := item.metaCopy()
	dirs, err := resolveDirs(meta.Dir, paths)
	if err != nil {
		return nil, err
	}
	return m.writeProject(item, meta, dirs)
}

// AddProjectDir adds one directory to a session's project, and returns the new
// list. A directory that is already in the project is left as it is.
func (m *Manager) AddProjectDir(name, path string) ([]string, error) {
	item, err := m.entry(name)
	if err != nil {
		return nil, err
	}
	meta := item.metaCopy()
	full, err := resolveDir(meta.Dir, path)
	if err != nil {
		return nil, err
	}
	return m.writeProject(item, meta, appendUnique(meta.WorkingDirs, full))
}

// RemoveProjectDir takes one directory out of a session's project, and returns
// the new list. It resolves the path the same way it was added.
func (m *Manager) RemoveProjectDir(name, path string) ([]string, error) {
	item, err := m.entry(name)
	if err != nil {
		return nil, err
	}
	meta := item.metaCopy()
	full, err := resolveDir(meta.Dir, path)
	if err != nil {
		return nil, err
	}
	var kept []string
	for _, dir := range meta.WorkingDirs {
		if dir != full {
			kept = append(kept, dir)
		}
	}
	return m.writeProject(item, meta, kept)
}

// ClearProject empties a session's project, and reports whether it had one.
func (m *Manager) ClearProject(name string) (bool, error) {
	item, err := m.entry(name)
	if err != nil {
		return false, err
	}
	meta := item.metaCopy()
	if len(meta.WorkingDirs) == 0 {
		return false, nil
	}
	if _, err := m.writeProject(item, meta, nil); err != nil {
		return false, err
	}
	return true, nil
}

func (m *Manager) writeProject(item *entry, meta Meta, dirs []string) ([]string, error) {
	meta.WorkingDirs = dirs
	item.setMeta(meta)
	if err := writeMeta(item.path, meta); err != nil {
		return nil, err
	}
	return dirs, nil
}

func resolveDirs(base string, paths []string) ([]string, error) {
	var out []string
	for _, path := range paths {
		full, err := resolveDir(base, path)
		if err != nil {
			return nil, err
		}
		out = appendUnique(out, full)
	}
	return out, nil
}

func appendUnique(dirs []string, dir string) []string {
	if slices.Contains(dirs, dir) {
		return dirs
	}
	return append(dirs, dir)
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
// level, because Claude Code has no live effort switch; see docs/protocol/control.md.
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

// ResumeWithControl stops a running session and resumes it with control turned
// on or off, because the multiplexer equips its tools when a session starts;
// see docs/protocol/control.md.
func (m *Manager) ResumeWithControl(ctx context.Context, name string, control bool) (string, error) {
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
		Effort:          snap.Effort,
		Control:         control,
		Parent:          item.metaCopy().Parent,
		ClaudeSessionID: snap.ClaudeSessionID,
	})
}

func (m *Manager) Stop(ctx context.Context, name string) error {
	if re := m.remote(name); re != nil {
		if re.readOnly {
			return ErrReadOnly
		}
		return re.client.Stop(ctx, re.remoteName)
	}
	item, err := m.entry(name)
	if err != nil {
		return err
	}
	return item.sess.Stop(ctx)
}

// Remove drops a session for good, and deletes its temporary working directory
// when it ran in one. Resume uses evict instead, so it keeps the directory for
// the re-spawn. See docs/peers.md.
func (m *Manager) Remove(name string) error {
	item, err := m.evict(name)
	if err != nil {
		return err
	}
	meta := item.metaCopy()
	if meta.TempDir && meta.Dir != "" {
		_ = os.RemoveAll(meta.Dir)
	}
	return nil
}

// evict drops a session from the live set, and returns its entry. It leaves the
// working directory alone, so a resume that evicts then re-spawns keeps it.
func (m *Manager) evict(name string) (*entry, error) {
	item, err := m.entry(name)
	if err != nil {
		return nil, err
	}
	if item.sess.State().Live() {
		return nil, fmt.Errorf("%w: %s", ErrStillLive, name)
	}
	m.mu.Lock()
	delete(m.entries, name)
	m.order = removeName(m.order, name)
	m.mu.Unlock()
	return item, nil
}

func (m *Manager) Shutdown(ctx context.Context) {
	if m.usageStop != nil {
		m.usageStop()
		m.usageStop = nil
	}
	if m.schedStop != nil {
		close(m.schedStop)
		m.schedWG.Wait()
		m.schedStop = nil
	}

	m.mu.Lock()
	items := make([]*entry, 0, len(m.order))
	for _, name := range m.order {
		items = append(items, m.entries[name])
	}
	for _, re := range m.remotes {
		re.cancel()
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

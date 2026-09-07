package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/mcp"
)

// bridge adapts the manager to the MCP tools, and publishes a notice for each
// change a tool makes, so the interface can show who did what.
type bridge struct{ m *Manager }

func (b *bridge) SetTitle(name, title string) error {
	if err := b.m.SetTitle(name, title); err != nil {
		return err
	}
	notice := name + " cleared its title"
	if title != "" {
		notice = name + " renamed itself to " + title
	}
	b.m.notify(name, notice, true)
	return nil
}

func (b *bridge) SendFrom(target, from, text string) (int, error) {
	return b.m.SendFrom(target, from, text)
}

func (b *bridge) Stop(ctx context.Context, name, by string) error {
	if err := b.m.Stop(ctx, name); err != nil {
		return err
	}
	b.m.notify(name, by+" stopped "+name, false)
	return nil
}

func (b *bridge) Archive(name string, archived bool, by string) error {
	if err := b.m.Archive(name, archived); err != nil {
		return err
	}
	verb := " archived "
	if !archived {
		verb = " restored "
	}
	b.m.notify(name, by+verb+name, true)
	return nil
}

func (b *bridge) Create(dir, name, by string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("%w: %s", ErrNotDirectory, dir)
	}
	created, err := b.m.Spawn(context.Background(), Spec{Dir: abs, Name: name, Parent: by})
	if err != nil {
		return "", err
	}
	b.m.notify(created, by+" created "+created, true)
	return created, nil
}

func (b *bridge) List() []mcp.Session { return b.m.List() }

func (b *bridge) Messages(name string, limit int) ([]mcp.Message, error) {
	return b.m.Messages(name, limit)
}

func (b *bridge) Jobs(name string) ([]mcp.Job, error) { return b.m.Jobs(name) }

func (b *bridge) ConfigPath() mcp.ConfigPath { return b.m.ConfigPath() }

func (b *bridge) TemplatePath(name string) (mcp.TemplatePath, error) { return b.m.TemplatePath(name) }

func (b *bridge) SetWorkingDir(path, by string) (string, error) {
	full, err := b.m.SetWorkingDir(by, path)
	if err != nil {
		return "", err
	}
	b.m.notify(by, by+" set its working directory to "+full, true)
	return full, nil
}

func (b *bridge) UnsetWorkingDir(by string) (bool, error) {
	changed, err := b.m.UnsetWorkingDir(by)
	if err != nil {
		return false, err
	}
	if changed {
		b.m.notify(by, by+" cleared its working directory", true)
	}
	return changed, nil
}

func (b *bridge) Project(session string) ([]string, error) { return b.m.Project(session) }

func (b *bridge) SetProject(paths []string, by string) ([]string, error) {
	dirs, err := b.m.SetProject(by, paths)
	if err != nil {
		return nil, err
	}
	b.m.notify(by, by+" set its project to "+projectSummary(dirs), true)
	return dirs, nil
}

func (b *bridge) AddProjectDir(path, by string) ([]string, error) {
	dirs, err := b.m.AddProjectDir(by, path)
	if err != nil {
		return nil, err
	}
	b.m.notify(by, by+" added "+path+" to its project", true)
	return dirs, nil
}

func (b *bridge) RemoveProjectDir(path, by string) ([]string, error) {
	dirs, err := b.m.RemoveProjectDir(by, path)
	if err != nil {
		return nil, err
	}
	b.m.notify(by, by+" removed "+path+" from its project", true)
	return dirs, nil
}

func (b *bridge) ClearProject(by string) (bool, error) {
	changed, err := b.m.ClearProject(by)
	if err != nil {
		return false, err
	}
	if changed {
		b.m.notify(by, by+" cleared its project", true)
	}
	return changed, nil
}

func projectSummary(dirs []string) string {
	if len(dirs) == 0 {
		return "no directories"
	}
	return strconv.Itoa(len(dirs)) + " directories"
}

func (b *bridge) Layouts(session string) (mcp.LayoutList, error) {
	return b.m.LayoutList(session)
}

func (b *bridge) SaveLayout(name string, dims mcp.LayoutDims, by string) (string, error) {
	path, err := b.m.SaveLayout(name, layoutOf(dims), by)
	if err != nil {
		return "", err
	}
	b.m.notify(by, by+" saved the layout "+name, false)
	return path, nil
}

func (b *bridge) DeleteLayout(name, by string) (string, bool, error) {
	path, changed, err := b.m.DeleteLayout(name)
	if err != nil {
		return "", false, err
	}
	if changed {
		b.m.notify(by, by+" deleted the layout "+name, false)
	}
	return path, changed, nil
}

func (b *bridge) SetLayout(name, scope, by string) (string, error) {
	switch scope {
	case mcp.ScopeAll:
		path, err := b.m.SetActiveLayout(name)
		if err != nil {
			return "", err
		}
		b.m.notify(by, by+" set the layout "+name+" for all sessions", false)
		return path, nil
	case mcp.ScopeSession:
		if err := b.m.SetSessionLayout(by, name); err != nil {
			return "", err
		}
		b.m.notify(by, by+" set its layout to "+name, true)
		return "", nil
	default:
		return "", mcp.ErrBadScope
	}
}

func (b *bridge) UnsetLayout(scope, by string) (string, bool, error) {
	switch scope {
	case mcp.ScopeAll:
		path, changed, err := b.m.UnsetActiveLayout()
		if err != nil {
			return "", false, err
		}
		if changed {
			b.m.notify(by, by+" cleared the layout for all sessions", false)
		}
		return path, changed, nil
	case mcp.ScopeSession:
		changed, err := b.m.UnsetSessionLayout(by)
		if err != nil {
			return "", false, err
		}
		if changed {
			b.m.notify(by, by+" cleared its layout", true)
		}
		return "", changed, nil
	default:
		return "", false, mcp.ErrBadScope
	}
}

func layoutOf(dims mcp.LayoutDims) config.Layout {
	return config.Layout{
		PromptMin:    dims.PromptMin,
		PromptMax:    dims.PromptMax,
		SidebarSize:  dims.SidebarSize,
		TaskSize:     dims.TaskSize,
		DiffSize:     dims.DiffSize,
		DiffPosition: dims.DiffPosition,
	}
}

func (b *bridge) SetEditor(editor string, terminal *bool, by string) (string, error) {
	path, err := b.m.SetEditor(editor, terminal)
	if err != nil {
		return "", err
	}
	b.m.notify(by, by+" "+editorNotice(editor, terminal), false)
	return path, nil
}

func (b *bridge) UnsetEditor(field, by string) (string, bool, error) {
	path, changed, err := b.m.UnsetEditor(field)
	if err != nil {
		return "", false, err
	}
	if changed {
		b.m.notify(by, by+" cleared "+clearedNotice(field), false)
	}
	return path, changed, nil
}

func (b *bridge) SetConfig(path string, value json.RawMessage, by string) (string, error) {
	file, err := b.m.SetConfig(path, value)
	if err != nil {
		return "", err
	}
	b.m.notify(by, by+" set "+path+" in the settings", false)
	return file, nil
}

func (b *bridge) UnsetConfig(path, by string) (string, bool, error) {
	file, changed, err := b.m.UnsetConfig(path)
	if err != nil {
		return "", false, err
	}
	if changed {
		b.m.notify(by, by+" cleared "+path+" from the settings", false)
	}
	return file, changed, nil
}

func (b *bridge) SetBlockCap(bucket string, rows *int, by string) (string, error) {
	path, err := b.m.SetBlockCap(bucket, rows)
	if err != nil {
		return "", err
	}
	b.m.notify(by, by+" "+blockCapNotice(bucket, rows), false)
	return path, nil
}

func (b *bridge) UnsetBlockCap(bucket, by string) (string, bool, error) {
	path, changed, err := b.m.UnsetBlockCap(bucket)
	if err != nil {
		return "", false, err
	}
	if changed {
		b.m.notify(by, by+" "+blockCapClearedNotice(bucket), false)
	}
	return path, changed, nil
}

func blockCapNotice(bucket string, rows *int) string {
	if bucket == "" {
		if rows == nil || *rows == 0 {
			return "turned the block cap off"
		}
		return "set the block cap to " + strconv.Itoa(*rows) + " rows"
	}
	if rows == nil {
		return "turned the " + bucket + " block cap off"
	}
	if *rows == 0 {
		return "set the " + bucket + " block cap to the marker only"
	}
	return "set the " + bucket + " block cap to " + strconv.Itoa(*rows) + " rows"
}

func blockCapClearedNotice(bucket string) string {
	if bucket == "" {
		return "cleared the block cap"
	}
	return "cleared the " + bucket + " block cap"
}

func clearedNotice(field string) string {
	switch field {
	case config.FieldEditor:
		return "the editor"
	case config.FieldTerminal:
		return "the terminal flag of the editor"
	default:
		return "the editor settings"
	}
}

func editorNotice(editor string, terminal *bool) string {
	if editor == "" {
		if terminal != nil && *terminal {
			return "made the editor a terminal editor"
		}
		return "made the editor a window editor"
	}
	return "set the editor to " + editor
}

func (b *bridge) StopJob(target, jobID, by string) (int, error) {
	return b.m.StopJobFrom(target, by, jobID)
}

func (b *bridge) CreateSchedule(in mcp.ScheduleInput, by string) (mcp.Schedule, error) {
	sched, err := b.m.CreateSchedule(ScheduleSpec{
		Name:           in.Name,
		Cron:           in.Cron,
		Dir:            in.Dir,
		Prompt:         in.Prompt,
		Session:        in.Session,
		Model:          in.Model,
		PermissionMode: in.PermissionMode,
		Effort:         in.Effort,
		Control:        in.Control,
	})
	if err != nil {
		return mcp.Schedule{}, err
	}
	b.m.notify(by, by+" created the schedule "+sched.Name, true)
	return scheduleView(sched), nil
}

func (b *bridge) UpdateSchedule(name string, up mcp.ScheduleEdit, by string) (mcp.Schedule, error) {
	sched, err := b.m.UpdateSchedule(name, ScheduleUpdate{
		Cron:           up.Cron,
		Dir:            up.Dir,
		Prompt:         up.Prompt,
		Session:        up.Session,
		Model:          up.Model,
		PermissionMode: up.PermissionMode,
		Effort:         up.Effort,
		Control:        up.Control,
	})
	if err != nil {
		return mcp.Schedule{}, err
	}
	b.m.notify(by, by+" updated the schedule "+name, true)
	return scheduleView(sched), nil
}

func (b *bridge) ListSchedules() []mcp.Schedule {
	list := b.m.ListSchedules()
	out := make([]mcp.Schedule, 0, len(list))
	for _, sched := range list {
		out = append(out, scheduleView(sched))
	}
	return out
}

func (b *bridge) DeleteSchedule(name, by string) (bool, error) {
	if err := b.m.DeleteSchedule(name); err != nil {
		return false, err
	}
	b.m.notify(by, by+" deleted the schedule "+name, true)
	return true, nil
}

func (b *bridge) SetScheduleEnabled(name string, on bool, by string) (mcp.Schedule, error) {
	sched, err := b.m.SetScheduleEnabled(name, on)
	if err != nil {
		return mcp.Schedule{}, err
	}
	verb := "enabled"
	if !on {
		verb = "paused"
	}
	b.m.notify(by, by+" "+verb+" the schedule "+name, true)
	return scheduleView(sched), nil
}

func (b *bridge) SchedulePath() mcp.SchedulePath { return b.m.SchedulePath() }

func (b *bridge) CreateAPIAdmin() (string, error) { return b.m.CreateAPIAdmin() }

func (b *bridge) RotateAPIAdmin() (string, error) { return b.m.RotateAPIAdmin() }

func (b *bridge) RevokeAPIAdmin() error { return b.m.RevokeAPIAdmin() }

func (b *bridge) CreateAPIClient(name string) (mcp.APIClient, string, error) {
	return b.m.CreateAPIClient(name)
}

func (b *bridge) UpdateAPIClient(id string, name *string, disabled *bool) (mcp.APIClient, error) {
	return b.m.UpdateAPIClient(id, name, disabled)
}

func (b *bridge) RotateAPIClient(id string) (string, error) { return b.m.RotateAPIClient(id) }

func (b *bridge) RevokeAPIClient(id string) error { return b.m.RevokeAPIClient(id) }

func (b *bridge) ListAPIClients() []mcp.APIClient { return b.m.ListAPIClients() }

func (b *bridge) APIEndpoint() mcp.APIEndpoint { return b.m.APIEndpoint() }

func (b *bridge) RunSchedule(name, by string) (string, error) {
	session, err := b.m.RunSchedule(name)
	if err != nil {
		return "", err
	}
	b.m.notify(session, by+" ran the schedule "+name, true)
	return session, nil
}

func scheduleView(s Schedule) mcp.Schedule {
	view := mcp.Schedule{
		Name:        s.Name,
		Cron:        s.Cron,
		Dir:         s.Dir,
		Prompt:      s.Prompt,
		Session:     s.Session,
		Model:       s.Model,
		Enabled:     s.Enabled,
		LastSession: s.LastSession,
	}
	if !s.LastRun.IsZero() {
		view.LastRun = s.LastRun.Format(time.RFC3339)
	}
	return view
}

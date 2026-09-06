package manager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

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
		SidebarWidth: dims.SidebarWidth,
		TaskWidth:    dims.TaskWidth,
		DiffWidth:    dims.DiffWidth,
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

package manager

import (
	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/mcp"
)

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

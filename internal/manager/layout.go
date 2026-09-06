package manager

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/mcp"
)

var errNoSettings = errors.New("manager: no settings file to write")

// SessionLayouts reports the layout each live session names, so the interface
// resolves the dimensions of the selected session. A stored session reads its
// layout straight from its meta. See docs/config.md.
func (m *Manager) SessionLayouts() map[string]string {
	m.mu.Lock()
	out := make(map[string]string, len(m.entries))
	for name, item := range m.entries {
		if layout := item.metaCopy().Layout; layout != "" {
			out[name] = layout
		}
	}
	m.mu.Unlock()
	return out
}

// sessionLayout reads the layout of one session, live or stored.
func (m *Manager) sessionLayout(name string) string {
	m.mu.Lock()
	item, live := m.entries[name]
	m.mu.Unlock()
	if live {
		return item.metaCopy().Layout
	}
	meta, err := ReadMeta(metaPath(m.opts.Root, name))
	if err != nil {
		return ""
	}
	return meta.Layout
}

// LayoutList describes the named layouts, the global active layout, and the
// layout of one session. See docs/mcp/tools.md.
func (m *Manager) LayoutList(session string) (mcp.LayoutList, error) {
	cfg, err := config.Load(m.opts.ConfigPaths...)
	if err != nil {
		return mcp.LayoutList{}, err
	}
	out := mcp.LayoutList{
		Session:       session,
		ActiveGlobal:  cfg.ActiveLayout,
		ActiveSession: m.sessionLayout(session),
	}
	names := make([]string, 0, len(cfg.Layouts))
	for name := range cfg.Layouts {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		layout := cfg.Layouts[name]
		out.Layouts = append(out.Layouts, mcp.LayoutInfo{Name: name, LayoutDims: dimsOf(layout)})
	}
	return out, nil
}

// SaveLayout creates or replaces a named layout. It captures the current
// dimensions of the session, and a field in dims overrides the captured one, so
// the same call edits one field of an existing layout. See docs/config.md.
func (m *Manager) SaveLayout(name string, dims config.Layout, session string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("manager: a layout needs a name")
	}
	path := config.Target(m.opts.ConfigPaths...)
	if path == "" {
		return "", errNoSettings
	}
	current, err := config.Load(path)
	if err != nil {
		return "", err
	}
	resolved := config.ResolveLayout(current.Layouts, current.ActiveLayout, m.sessionLayout(session))
	layout := config.Layout{
		PromptMin:    intPtr(resolved.PromptMin),
		PromptMax:    intPtr(resolved.PromptMax),
		SidebarWidth: intPtr(resolved.SidebarWidth),
		TaskWidth:    intPtr(resolved.TaskWidth),
		DiffWidth:    intPtr(resolved.DiffWidth),
	}
	overlay(&layout, dims)
	if current.Layouts == nil {
		current.Layouts = map[string]config.Layout{}
	}
	current.Layouts[name] = layout
	if err := config.Write(path, current); err != nil {
		return "", err
	}
	return path, nil
}

// DeleteLayout removes a named layout, and reports whether it was there.
func (m *Manager) DeleteLayout(name string) (string, bool, error) {
	path := config.Target(m.opts.ConfigPaths...)
	if path == "" {
		return "", false, errNoSettings
	}
	current, err := config.Load(path)
	if err != nil {
		return "", false, err
	}
	if _, ok := current.Layouts[name]; !ok {
		return path, false, nil
	}
	delete(current.Layouts, name)
	if len(current.Layouts) == 0 {
		current.Layouts = nil
	}
	if err := config.Write(path, current); err != nil {
		return "", false, err
	}
	return path, true, nil
}

// SetActiveLayout names the layout every session takes, unless the session names
// its own. The layout must exist. See docs/config.md.
func (m *Manager) SetActiveLayout(name string) (string, error) {
	path := config.Target(m.opts.ConfigPaths...)
	if path == "" {
		return "", errNoSettings
	}
	current, err := config.Load(path)
	if err != nil {
		return "", err
	}
	if _, ok := current.Layouts[name]; !ok {
		return "", fmt.Errorf("manager: no layout named %q", name)
	}
	current.ActiveLayout = name
	if err := config.Write(path, current); err != nil {
		return "", err
	}
	return path, nil
}

// UnsetActiveLayout clears the global active layout, and reports whether one was
// set.
func (m *Manager) UnsetActiveLayout() (string, bool, error) {
	path := config.Target(m.opts.ConfigPaths...)
	if path == "" {
		return "", false, errNoSettings
	}
	current, err := config.Load(path)
	if err != nil {
		return "", false, err
	}
	if current.ActiveLayout == "" {
		return path, false, nil
	}
	current.ActiveLayout = ""
	if err := config.Write(path, current); err != nil {
		return "", false, err
	}
	return path, true, nil
}

// SetSessionLayout names the layout of one session, which overrides the global
// active layout. The layout must exist. It works for a live and a stored
// session. See docs/config.md.
func (m *Manager) SetSessionLayout(session, name string) error {
	cfg, err := config.Load(m.opts.ConfigPaths...)
	if err != nil {
		return err
	}
	if _, ok := cfg.Layouts[name]; !ok {
		return fmt.Errorf("manager: no layout named %q", name)
	}
	return m.mutateSessionLayout(session, name)
}

// UnsetSessionLayout takes the layout off one session, so it takes the global
// active layout again. It reports whether the session had one.
func (m *Manager) UnsetSessionLayout(session string) (bool, error) {
	if m.sessionLayout(session) == "" {
		return false, nil
	}
	if err := m.mutateSessionLayout(session, ""); err != nil {
		return false, err
	}
	return true, nil
}

// mutateSessionLayout writes the layout onto a session's meta, live or stored,
// in the pattern of SetTitle. See docs/manager.md.
func (m *Manager) mutateSessionLayout(name, layout string) error {
	m.mu.Lock()
	item, live := m.entries[name]
	m.mu.Unlock()
	if live {
		meta := item.metaCopy()
		meta.Layout = layout
		item.setMeta(meta)
		return writeMeta(item.path, meta)
	}
	path := metaPath(m.opts.Root, name)
	meta, err := ReadMeta(path)
	if err != nil {
		return err
	}
	meta.Layout = layout
	return writeMeta(path, meta)
}

func dimsOf(layout config.Layout) mcp.LayoutDims {
	return mcp.LayoutDims{
		PromptMin:    layout.PromptMin,
		PromptMax:    layout.PromptMax,
		SidebarWidth: layout.SidebarWidth,
		TaskWidth:    layout.TaskWidth,
		DiffWidth:    layout.DiffWidth,
	}
}

func overlay(base *config.Layout, dims config.Layout) {
	if dims.PromptMin != nil {
		base.PromptMin = dims.PromptMin
	}
	if dims.PromptMax != nil {
		base.PromptMax = dims.PromptMax
	}
	if dims.SidebarWidth != nil {
		base.SidebarWidth = dims.SidebarWidth
	}
	if dims.TaskWidth != nil {
		base.TaskWidth = dims.TaskWidth
	}
	if dims.DiffWidth != nil {
		base.DiffWidth = dims.DiffWidth
	}
}

func intPtr(n int) *int { return &n }

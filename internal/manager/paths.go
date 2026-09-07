package manager

import (
	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/mcp"
	"github.com/dextermb/claude-multiplexer/internal/template"
)

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

// SchedulePath names the directory the multiplexer writes schedule records to.
// See docs/scheduler.md.
func (m *Manager) SchedulePath() mcp.SchedulePath {
	return mcp.SchedulePath{Dir: scheduleDir(m.opts.Root)}
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

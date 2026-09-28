package tui

import (
	"os/exec"

	tea "charm.land/bubbletea/v2"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/open"
)

type openedMsg struct {
	what     string
	dir      string
	terminal bool
	err      error
}

// launchTerminal and launchDetached start a program. They are variables so a
// test can record the command instead of starting it.
var (
	launchTerminal = func(cmd *exec.Cmd, done func(error) tea.Msg) tea.Cmd {
		return tea.ExecProcess(cmd, done)
	}
	launchDetached = func(cmd *exec.Cmd, done func(error) tea.Msg) tea.Cmd {
		return func() tea.Msg {
			if err := cmd.Start(); err != nil {
				return done(err)
			}
			go cmd.Wait()
			return done(nil)
		}
	}
)

func (m Model) openInFiles() (tea.Model, tea.Cmd) {
	item, ok := m.selectedRow()
	if !ok {
		return m, nil
	}
	m.errText = ""
	var cmds []tea.Cmd
	for _, dir := range item.diffDirs() {
		cmds = append(cmds, launchCmd("file manager", open.FileManager(dir)))
	}
	return m, tea.Batch(cmds...)
}

func (m Model) openInEditor() (tea.Model, tea.Cmd) {
	item, ok := m.selectedRow()
	if !ok {
		return m, nil
	}
	settings, err := m.editorSettings()
	if err != nil {
		m.errText = err.Error()
		return m, nil
	}
	targets, err := open.Editors(settings, item.diffDirs())
	if err != nil {
		m.errText = err.Error()
		return m, nil
	}
	m.errText = ""
	var cmds []tea.Cmd
	for _, target := range targets {
		cmds = append(cmds, launchCmd("editor", target))
	}
	return m, tea.Batch(cmds...)
}

// editorSettings reads the settings file at the key press, so a file a session
// wrote with set_editor is seen without a restart. See docs/config.md.
func (m Model) editorSettings() (config.Config, error) {
	file, err := config.Load(m.opts.ConfigPaths...)
	if err != nil {
		return config.Config{}, err
	}
	return config.Resolve(m.opts.Config, file, config.LoadClaude(m.opts.ClaudePaths...)), nil
}

func launchCmd(what string, target open.Target) tea.Cmd {
	cmd := exec.Command(target.Command, target.Args...)
	cmd.Dir = target.Dir
	done := func(err error) tea.Msg {
		return openedMsg{what: what, dir: target.Dir, terminal: target.Terminal, err: err}
	}
	if target.Terminal {
		return launchTerminal(cmd, done)
	}
	return launchDetached(cmd, done)
}

func (m Model) handleOpened(msg openedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.status = ""
		m.errText = msg.what + ": " + msg.err.Error()
		return m, nil
	}
	m.errText = ""
	m.status = ""
	if !msg.terminal {
		m.status = "opened " + msg.dir
	}
	return m, nil
}

package tui

import (
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

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
		cmds = append(cmds, launchCmd("file manager", dir, open.FileManager(dir)))
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
	dirs := item.diffDirs()
	m.errText = ""
	first, err := open.Editor(settings, dirs[0])
	if err != nil {
		m.errText = err.Error()
		return m, nil
	}
	if first.Terminal {
		// A terminal editor holds the terminal, so it opens every directory as an argument of one process, not one window each.
		args := append(append([]string{}, first.Args...), dirs[1:]...)
		return m, launchCmd("editor", dirs[0], open.Target{Command: first.Command, Args: args, Terminal: true})
	}
	cmds := []tea.Cmd{launchCmd("editor", dirs[0], first)}
	for _, dir := range dirs[1:] {
		target, err := open.Editor(settings, dir)
		if err != nil {
			m.errText = err.Error()
			return m, nil
		}
		cmds = append(cmds, launchCmd("editor", dir, target))
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

func launchCmd(what, dir string, target open.Target) tea.Cmd {
	cmd := exec.Command(target.Command, target.Args...)
	cmd.Dir = dir
	done := func(err error) tea.Msg {
		return openedMsg{what: what, dir: dir, terminal: target.Terminal, err: err}
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

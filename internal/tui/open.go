package tui

import (
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

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
	return m.launch("file manager", item.dir, open.FileManager(item.dir))
}

func (m Model) openInEditor() (tea.Model, tea.Cmd) {
	item, ok := m.selectedRow()
	if !ok {
		return m, nil
	}
	target, err := open.Editor(m.opts.Config, item.dir)
	if err != nil {
		m.errText = err.Error()
		return m, nil
	}
	return m.launch("editor", item.dir, target)
}

func (m Model) launch(what, dir string, target open.Target) (tea.Model, tea.Cmd) {
	m.errText = ""
	cmd := exec.Command(target.Command, target.Args...)
	cmd.Dir = dir
	done := func(err error) tea.Msg {
		return openedMsg{what: what, dir: dir, terminal: target.Terminal, err: err}
	}
	if target.Terminal {
		return m, launchTerminal(cmd, done)
	}
	return m, launchDetached(cmd, done)
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

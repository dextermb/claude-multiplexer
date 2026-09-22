package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dextermb/claude-multiplexer/internal/config"
)

// commandTimeout stops a command script that runs too long, so a stuck script
// does not linger. The run is off the main loop, so a slow script never blocks
// the interface. See docs/config/commands.md.
const commandTimeout = 10 * time.Second

// commandOutputMsg carries one command result back to the main loop.
type commandOutputMsg struct {
	label string
	text  string
	err   error
}

// runCommand runs a command's script off the main loop, with the selected
// session on stdin. See docs/config/commands.md.
func (m Model) runCommand(c config.Command) (tea.Model, tea.Cmd) {
	return m, runCommandScript(c, m.commandPayload(), m.configDir())
}

// commandPayload gives the script the selected session, or the totals when no
// session is selected, the same JSON a custom bar element reads.
func (m Model) commandPayload() []byte {
	if item, ok := m.selectedRow(); ok {
		return m.sessionPayload(item)
	}
	return m.statusPayload()
}

func runCommandScript(c config.Command, payload []byte, baseDir string) tea.Cmd {
	return func() tea.Msg {
		text, err := runScriptSync(c.Script, payload, commandTimeout, baseDir)
		return commandOutputMsg{label: c.Label, text: text, err: err}
	}
}

// handleCommandOutput turns a command result into a status notice: the first
// stdout line, or a one-line error. Empty output shows nothing. See
// docs/config/commands.md.
func (m Model) handleCommandOutput(msg commandOutputMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.status = msg.label + ": " + firstLine([]byte(msg.err.Error()))
		return m, nil
	}
	if msg.text != "" {
		m.status = msg.label + ": " + msg.text
	}
	return m, nil
}

package tui

import (
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dextermb/claude-multiplexer/internal/render"
)

type bashResultMsg struct {
	session string
	command string
	output  string
	err     error
	feed    bool
}

// parseBang reads a bash command from a prompt. A leading `!` runs a command
// and shows the output; `!!` also feeds the output to Claude. See docs/tui/input.md.
func parseBang(text string) (command string, feed bool, ok bool) {
	if !strings.HasPrefix(text, "!") {
		return "", false, false
	}
	rest := text[1:]
	if strings.HasPrefix(rest, "!") {
		feed = true
		rest = rest[1:]
	}
	return strings.TrimSpace(rest), feed, true
}

func (m Model) runBash(command string, feed bool) (tea.Model, tea.Cmd) {
	if command == "" {
		m.errText = "empty bash command"
		return m, nil
	}
	item, ok := m.selectedRow()
	if !ok {
		m.errText = "no session is selected"
		return m, nil
	}
	m.errText = ""
	m.status = "running command…"
	return m, runBashCmd(item.dir, m.sel, command, feed)
}

func runBashCmd(dir, session, command string, feed bool) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("bash", "-c", command)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		return bashResultMsg{session: session, command: command, output: string(out), err: err, feed: feed}
	}
}

func (m Model) handleBash(msg bashResultMsg) (tea.Model, tea.Cmd) {
	m.status = ""
	if msg.feed {
		text := formatForClaude(msg.command, msg.output, msg.err)
		if err := m.mgr.Send(msg.session, text); err != nil {
			m.errText = err.Error()
			return m, nil
		}
		m.errText = ""
		m.queued[msg.session] = append(m.queued[msg.session], text)
		m.refresh()
		m.setContent()
		return m, m.ensureAnimating()
	}
	lines := render.BashLines(msg.command, msg.output, msg.err)
	_ = m.mgr.AppendLines(msg.session, lines)
	if msg.session == m.sel {
		m.appendOutput(lines)
	}
	return m, nil
}

func formatForClaude(command, output string, err error) string {
	var b strings.Builder
	b.WriteString("I ran this command:\n\n```\n")
	b.WriteString(command)
	b.WriteString("\n```\n\n")
	if body := strings.TrimRight(output, "\n"); body != "" {
		b.WriteString("It gave this output:\n\n```\n")
		b.WriteString(body)
		b.WriteString("\n```\n")
	} else {
		b.WriteString("It gave no output.\n")
	}
	if err != nil {
		b.WriteString("\nThe command failed: " + err.Error() + "\n")
	}
	return b.String()
}

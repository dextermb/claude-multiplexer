package tui

import (
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dextermb/claude-multiplexer/internal/manager"
	"github.com/dextermb/claude-multiplexer/internal/open"
)

// openSelectedPR is the d p action. It opens the pull request of the code base
// the current diff file belongs to. See docs/tui/diff.md.
func (m Model) openSelectedPR() (tea.Model, tea.Cmd) {
	item, ok := m.selectedRow()
	if !ok {
		return m, nil
	}
	entries := m.diffEntries()
	if m.diffSel < 0 || m.diffSel >= len(entries) {
		m.errText = "no file selected"
		return m, nil
	}
	url := prURLForDir(item.prs, entries[m.diffSel].dir)
	if url == "" {
		m.errText = "no pull request for this code base"
		return m, nil
	}
	m.errText = ""
	return m, openURLCmd(url)
}

// openAllPRs is the d P action. It opens the pull request of every code base of
// the project. See docs/tui/diff.md.
func (m Model) openAllPRs() (tea.Model, tea.Cmd) {
	item, ok := m.selectedRow()
	if !ok {
		return m, nil
	}
	var cmds []tea.Cmd
	seen := map[string]bool{}
	for _, pr := range item.prs {
		if pr.URL == "" || seen[pr.URL] {
			continue
		}
		seen[pr.URL] = true
		cmds = append(cmds, openURLCmd(pr.URL))
	}
	if len(cmds) == 0 {
		m.errText = "no pull requests"
		return m, nil
	}
	m.errText = ""
	return m, tea.Batch(cmds...)
}

// prURLForDir is the pull-request url of the code base at dir, or empty when the
// code base tracks none.
func prURLForDir(prs []manager.PRBadge, dir string) string {
	for _, pr := range prs {
		if pr.Dir == dir && pr.URL != "" {
			return pr.URL
		}
	}
	return ""
}

// openURLCmd opens a url in the browser, detached, and reports the outcome the
// same way the file-manager and editor launches do.
func openURLCmd(url string) tea.Cmd {
	target := open.Browser(url)
	cmd := exec.Command(target.Command, target.Args...)
	done := func(err error) tea.Msg {
		return openedMsg{what: "pull request", dir: url, err: err}
	}
	return launchDetached(cmd, done)
}

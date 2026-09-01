package tui

import (
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const mentionMark = '@'

type mentionToken struct {
	start int
	text  string
}

func mention(value string) (mentionToken, bool) {
	start := strings.LastIndexAny(value, " \t\n") + 1
	word := value[start:]
	if !strings.HasPrefix(word, string(mentionMark)) {
		return mentionToken{}, false
	}
	return mentionToken{start: start, text: word[1:]}, true
}

func (m Model) mentionBase() string {
	if item, ok := m.selectedRow(); ok && item.dir != "" {
		return item.dir
	}
	return m.opts.DefaultDir
}

// A completion runs only when the cursor sits after the last rune; see docs/tui/input.md.
func (m Model) cursorAtEnd() bool {
	value := m.prompt.Value()
	lines := strings.Split(value, "\n")
	if m.prompt.Line() != len(lines)-1 {
		return false
	}
	info := m.prompt.LineInfo()
	return info.StartColumn+info.ColumnOffset == len([]rune(lines[len(lines)-1]))
}

func (m *Model) syncMentions() {
	value := m.prompt.Value()
	base := m.mentionBase()
	if value == m.pathValue && base == m.pathBase {
		return
	}
	m.pathValue = value
	m.pathBase = base
	m.pathPicked = -1
	m.pathStem = ""
	m.pathMatches = nil

	token, ok := mention(value)
	if !ok || !m.cursorAtEnd() {
		return
	}
	_, m.pathMatches = completeEntry(base, token.text)
}

func (m Model) completeMention() (Model, bool) {
	token, ok := mention(m.prompt.Value())
	if !ok || !m.cursorAtEnd() {
		return m, false
	}
	completed, matches := completeEntry(m.mentionBase(), token.text)
	if len(matches) == 0 || completed == token.text {
		return m, false
	}
	m.setMention(token, completed)
	return m, true
}

func (m Model) walkMention(delta int) (Model, bool) {
	token, ok := mention(m.prompt.Value())
	if !ok || len(m.pathMatches) == 0 {
		return m, false
	}
	if m.pathPicked < 0 {
		m.pathStem, _ = splitPath(expandHome(token.text))
		m.pathPicked = 0
		if delta < 0 {
			m.pathPicked = len(m.pathMatches) - 1
		}
	} else {
		m.pathPicked = (m.pathPicked + delta + len(m.pathMatches)) % len(m.pathMatches)
	}
	picked := m.pathMatches[m.pathPicked]
	text := m.pathStem + picked.name
	if picked.dir {
		text += string(filepath.Separator)
	}
	m.setMention(token, text)
	m.pathValue = m.prompt.Value()
	return m, true
}

func (m *Model) setMention(token mentionToken, text string) {
	m.prompt.SetValue(m.prompt.Value()[:token.start] + string(mentionMark) + text)
	m.prompt.CursorEnd()
}

func (m Model) mentionHint() (string, bool) {
	if m.focus != focusPrompt || len(m.pathMatches) == 0 {
		return "", false
	}
	hint := pathHint(matchNames(m.pathMatches), m.pathPicked, m.width-2)
	if m.pathPicked < 0 {
		hint = truncate(hint+"   tab completes", m.width-2)
	}
	return hintStyle.Render(hint), true
}

func (m Model) mentionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if msg.String() != "shift+tab" {
		return m, nil, false
	}
	next, ok := m.walkMention(-1)
	if !ok {
		return m, nil, false
	}
	return next, nil, true
}

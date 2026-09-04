package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const sequenceTimeout = time.Second

type sequenceTimeoutMsg struct{ gen int }

type sequence struct {
	target string
	gen    int
}

var sequenceTargets = map[string]string{
	"s":      "s",
	"l":      "l",
	"o":      "o",
	"ctrl+s": "s",
	"ctrl+l": "l",
	"ctrl+o": "o",
}

// Inside the prompt only the control forms start a sequence, so every other key
// stays text.
func sequenceTarget(key string, inPrompt bool) (string, bool) {
	if inPrompt && !strings.HasPrefix(key, "ctrl+") {
		return "", false
	}
	target, ok := sequenceTargets[key]
	return target, ok
}

func (m Model) startSequence(target string) (tea.Model, tea.Cmd) {
	m.seqGen++
	m.seq = &sequence{target: target, gen: m.seqGen}
	m.errText = ""
	gen := m.seqGen
	return m, tea.Tick(sequenceTimeout, func(time.Time) tea.Msg {
		return sequenceTimeoutMsg{gen: gen}
	})
}

func (m Model) resolveSequence(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	target := m.seq.target
	m.seq = nil
	key := msg.String()
	if key == "esc" {
		return m, nil
	}
	if run, ok := sequenceActions[target+" "+key]; ok {
		return run(m)
	}
	m.status = "no key " + target + " " + key
	return m, nil
}

func (m Model) handleSequenceTimeout(msg sequenceTimeoutMsg) (tea.Model, tea.Cmd) {
	if m.seq != nil && m.seq.gen == msg.gen {
		m.seq = nil
	}
	return m, nil
}

type action func(Model) (tea.Model, tea.Cmd)

var sequenceActions = map[string]action{
	"s c": Model.openNewForm,
	"s t": Model.openPicker,
	"s r": Model.resumeSelected,
	"s n": Model.openRename,
	"s a": Model.archiveSelected,
	"s x": Model.askToStop,
	"s j": Model.openJobs,
	"s f": Model.openInFiles,
	"s d": Model.openInEditor,
	"s m": func(m Model) (tea.Model, tea.Cmd) { return m.openChoice(settingModel) },
	"s e": func(m Model) (tea.Model, tea.Cmd) { return m.openChoice(settingEffort) },
	"s p": func(m Model) (tea.Model, tea.Cmd) { return m.openChoice(settingMode) },

	"l f": Model.toggleFold,
	"l F": Model.foldOthers,
	"l u": Model.unfoldAll,
	"l a": Model.toggleArchived,

	"o m": Model.toggleMarkdown,
}

// sequenceHints lists the action keys of a target, for the status bar.
func sequenceHints(target string) string {
	var out []string
	for _, item := range bindings {
		if item.target != target || item.brief == "" {
			continue
		}
		out = append(out, item.brief)
	}
	return strings.Join(out, " · ")
}

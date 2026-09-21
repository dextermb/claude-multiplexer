package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dextermb/claude-multiplexer/internal/keys"
)

const sequenceTimeout = time.Second

type sequenceTimeoutMsg struct{ gen int }

type sequence struct {
	target keys.Action
	gen    int
}

func defaultKeymap() keys.Keymap {
	km, _, _ := keys.LoadKeymap(nil)
	return km
}

// sequenceTarget reports the target action a key starts, if any. Inside the
// prompt only the control forms start a sequence, so every other key stays text.
// The diff target starts only while the diff panel is open. See docs/tui/keys.md.
func (m Model) sequenceTarget(key string, inPrompt, diffOpen bool) (keys.Action, bool) {
	if inPrompt && !strings.HasPrefix(key, "ctrl+") {
		return "", false
	}
	action, ok := m.keys.Action(keys.CtxTarget, key)
	if !ok {
		return "", false
	}
	if action == keys.TargetDiff && !diffOpen {
		return "", false
	}
	return action, true
}

func (m Model) startSequence(target keys.Action) (tea.Model, tea.Cmd) {
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
	ctx, _ := keys.TargetContext(target)
	if action, ok := m.keys.Action(ctx, key); ok {
		if run, ok := chordActions[action]; ok {
			return run(m)
		}
	}
	m.status = "no key " + targetLabel(m.keys, target) + " " + key
	return m, nil
}

func (m Model) handleSequenceTimeout(msg sequenceTimeoutMsg) (tea.Model, tea.Cmd) {
	if m.seq != nil && m.seq.gen == msg.gen {
		m.seq = nil
	}
	return m, nil
}

type action func(Model) (tea.Model, tea.Cmd)

// chordActions runs the second key of a two-key sequence. The keymap resolves a
// key to an action; this maps an action to its handler. See docs/tui/keys.md.
var chordActions = map[keys.Action]action{
	keys.SessionNew:        Model.openNewForm,
	keys.SessionPresets:    Model.openPicker,
	keys.SessionResume:     Model.resumeSelected,
	keys.SessionRename:     Model.openRename,
	keys.SessionArchive:    Model.archiveSelected,
	keys.SessionStop:       Model.askToStop,
	keys.SessionJobs:       Model.openJobs,
	keys.SessionFocusTasks: Model.focusTaskPanel,
	keys.SessionFiles:      Model.openInFiles,
	keys.SessionDiff:       Model.toggleDiffPanel,
	keys.SessionReview:     Model.reviewSelected,
	keys.SessionEditor:     Model.openInEditor,
	keys.SessionModel:      func(m Model) (tea.Model, tea.Cmd) { return m.openChoice(settingModel) },
	keys.SessionEffort:     func(m Model) (tea.Model, tea.Cmd) { return m.openChoice(settingEffort) },
	keys.SessionMode:       func(m Model) (tea.Model, tea.Cmd) { return m.openChoice(settingMode) },
	keys.SessionControl:    Model.toggleControl,
	keys.SessionClearHold:  Model.clearContextHold,

	keys.ListFold:       Model.toggleFold,
	keys.ListFoldOthers: Model.foldOthers,
	keys.ListUnfold:     Model.unfoldAll,
	keys.ListArchived:   Model.toggleArchived,
	keys.ListSearch:     Model.focusSearch,
	keys.ListSidebar:    Model.toggleSidebar,
	keys.ListCollapse:   Model.collapseSidebar,
	keys.ListExpand:     Model.expandSidebar,

	keys.OutputMarkdown: Model.toggleMarkdown,
	keys.OutputLayouts:  Model.openLayoutSwitcher,
	keys.OutputAge:      Model.toggleAge,

	keys.DiffWider:    Model.widenDiff,
	keys.DiffNarrower: Model.narrowDiff,
	keys.DiffHalf:     Model.toggleHalfDiff,
	keys.DiffNumbers:  Model.toggleDiffNumbers,
	keys.DiffPr:       Model.openSelectedPR,
	keys.DiffAllPrs:   Model.openAllPRs,
}

// targetLabel is the first key of a target, for the status bar and the notices.
func targetLabel(km keys.Keymap, target keys.Action) string {
	if ks := km.Keys(target); len(ks) > 0 {
		return ks[0]
	}
	return string(target)
}

// globalEverywhere reports whether a global key works while the prompt has the
// focus, the same as the control forms, tab, and the page keys did before.
func globalEverywhere(key string) bool {
	return strings.HasPrefix(key, "ctrl+") || key == "tab" || key == "pgup" || key == "pgdown"
}

func (m Model) runGlobal(a keys.Action) (tea.Model, tea.Cmd) {
	switch a {
	case keys.GlobalNewSession:
		return m.openNewForm()
	case keys.GlobalPresets:
		return m.openPicker()
	case keys.GlobalToggleMouse:
		m.mouseOn = !m.mouseOn
		if m.mouseOn {
			return m, tea.EnableMouseCellMotion
		}
		return m, tea.DisableMouse
	case keys.GlobalQuit:
		return m.startQuit()
	case keys.GlobalFocusNext:
		if m.focus == focusPrompt {
			return m.complete()
		}
		return m.toggleFocus()
	case keys.GlobalPageUp:
		m.output.ViewUp()
		return m, nil
	case keys.GlobalPageDown:
		m.output.ViewDown()
		return m, nil
	}
	return m, nil
}

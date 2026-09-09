package tui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dextermb/claude-multiplexer/internal/manager"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

const (
	fieldHost = iota
	fieldHoist
	fieldDir
	fieldName
	fieldModel
	fieldMode
	fieldEffort
	fieldControl
	fieldFirst
	fieldCount
)

var fieldLabels = [fieldCount]string{"Host", "Peer mode", "Directory", "Name", "Model", "Permission mode", "Effort", "Control", "First prompt"}

// localHost is the host option for a session this host runs itself.
const localHost = "local"

// hoistMode runs a peer's session locally with the peer's credential; streamMode
// runs it on the peer and streams it in. See docs/peers/hoisted.md.
const (
	hoistMode  = "hoist"
	streamMode = "stream"
)

// The option lists of the select fields. An empty string sends nothing, so
// Claude Code takes the project or global default. See docs/config/new-session.md.
var (
	modelOptions   = append([]string{""}, modelChoices...)
	effortOptions  = append([]string{""}, session.EffortLevels...)
	controlOptions = []string{"no", "yes"}
	hoistOptions   = []string{hoistMode, streamMode}
)

type formResult int

const (
	formOpen formResult = iota
	formSubmitted
	formCancelled
)

// selectField is a one-row select that cycles a fixed list. It renders as
// ‹ value ›, and left/right change the value. See docs/tui/input.md.
type selectField struct {
	options []string
	labels  []string
	cursor  int
}

func newSelect(options []string, current string) *selectField {
	labels := make([]string, len(options))
	for i, option := range options {
		if option == "" {
			labels[i] = "default"
		} else {
			labels[i] = option
		}
	}
	s := &selectField{options: options, labels: labels}
	for i, option := range options {
		if option == current {
			s.cursor = i
			break
		}
	}
	return s
}

func (s *selectField) cycle(delta int) {
	s.cursor = (s.cursor + delta + len(s.options)) % len(s.options)
}

func (s *selectField) value() string { return s.options[s.cursor] }
func (s *selectField) label() string { return s.labels[s.cursor] }

type form struct {
	inputs     [fieldCount]textinput.Model
	selects    [fieldCount]*selectField
	focus      int
	err        string
	matches    []string
	picked     int
	stem       string
	hostShown  bool
	hoistable  map[string]bool
	defaultDir string
}

func newForm(dir string, defaults newSessionDefaults, peers, hoistable []string) *form {
	f := &form{hostShown: len(peers) > 0, hoistable: toSet(hoistable), defaultDir: dir}
	var placeholders [fieldCount]string
	placeholders[fieldDir] = dir
	placeholders[fieldName] = "taken from the directory"
	placeholders[fieldFirst] = "optional, and /preset works here"
	for i := range f.inputs {
		input := textinput.New()
		input.Placeholder = placeholders[i]
		input.CharLimit = 512
		input.Width = 40
		f.inputs[i] = input
	}
	f.inputs[fieldDir].SetValue(dir)
	f.selects[fieldHost] = newSelect(append([]string{localHost}, peers...), localHost)
	f.selects[fieldHoist] = newSelect(hoistOptions, hoistMode)
	f.selects[fieldModel] = newSelect(modelOptions, defaults.model)
	f.selects[fieldMode] = newSelect(modeChoices, defaults.mode)
	f.selects[fieldEffort] = newSelect(effortOptions, defaults.effort)
	f.selects[fieldControl] = newSelect(controlOptions, boolWord(defaults.control))
	f.focus = fieldDir
	f.inputs[fieldDir].CursorEnd()
	f.inputs[fieldDir].Focus()
	f.suggest()
	return f
}

func toSet(names []string) map[string]bool {
	set := make(map[string]bool, len(names))
	for _, name := range names {
		set[name] = true
	}
	return set
}

// visible reports whether a field is shown. The host field hides when no peer is
// configured, so the form is unchanged without peering. The peer-mode field
// shows only for a peer that holds a lent credential. See docs/peers.md and
// docs/peers/hoisted.md.
func (f *form) visible(i int) bool {
	switch i {
	case fieldHost:
		return f.hostShown
	case fieldHoist:
		return f.canHoist()
	default:
		return true
	}
}

// host is the chosen host: "local" for a session this host runs, or a peer name.
func (f *form) host() string { return f.selects[fieldHost].value() }

// canHoist reports whether the chosen host is a peer that lent a credential, so
// the form may run its session locally. See docs/peers/hoisted.md.
func (f *form) canHoist() bool {
	return f.host() != localHost && f.hoistable[f.host()]
}

// hoisting reports whether the form will run a hoisted session: a peer that can
// hoist, with the peer-mode field on hoist. See docs/peers/hoisted.md.
func (f *form) hoisting() bool {
	return f.canHoist() && f.selects[fieldHoist].value() == hoistMode
}

// localDir reports whether the directory is on this machine: a local session, or
// a hoisted one. A streamed session runs on the peer, so its directory is not.
func (f *form) localDir() bool {
	return f.host() == localHost || f.hoisting()
}

// syncDir follows the host and the peer mode: a local or hoisted session runs
// here, so the field takes the local default; a streamed session runs on the
// peer, so the field clears. See docs/peers/hoisted.md.
func (f *form) syncDir() {
	if f.localDir() {
		f.inputs[fieldDir].SetValue(f.defaultDir)
	} else {
		f.inputs[fieldDir].SetValue("")
	}
	f.inputs[fieldDir].CursorEnd()
	f.suggest()
}

func boolWord(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func (f *form) isSelect(i int) bool { return f.selects[i] != nil }

func (f *form) suggest() {
	f.picked = -1
	f.stem = ""
	// A streamed session's directory is on the peer, so local path completion
	// does not apply. A hoisted session runs here, so it does.
	if f.focus != fieldDir || !f.localDir() {
		f.matches = nil
		return
	}
	_, f.matches = completePath(f.inputs[fieldDir].Value())
}

func (f *form) cycle(delta int) bool {
	if f.focus != fieldDir || len(f.matches) == 0 {
		return false
	}
	if f.picked < 0 {
		f.stem, _ = splitPath(expandHome(f.inputs[fieldDir].Value()))
		f.picked = 0
		if delta < 0 {
			f.picked = len(f.matches) - 1
		}
	} else {
		f.picked = (f.picked + delta + len(f.matches)) % len(f.matches)
	}
	f.inputs[fieldDir].SetValue(f.stem + f.matches[f.picked] + string(filepath.Separator))
	f.inputs[fieldDir].CursorEnd()
	return true
}

func (f *form) completeDir() bool {
	if !f.localDir() {
		return false
	}
	value := f.inputs[fieldDir].Value()
	completed, matches := completePath(value)
	f.matches = matches
	if completed == value || completed == "" {
		return false
	}
	f.inputs[fieldDir].SetValue(completed)
	f.inputs[fieldDir].CursorEnd()
	f.suggest()
	return true
}

func (f *form) Update(msg tea.Msg) (formResult, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			return formCancelled, nil
		case "enter":
			if f.validate() {
				return formSubmitted, nil
			}
			return formOpen, nil
		case "tab":
			if f.focus == fieldDir {
				if f.picked >= 0 && f.cycle(1) {
					return formOpen, nil
				}
				if f.completeDir() {
					return formOpen, nil
				}
			}
			f.move(1)
			return formOpen, nil
		case "shift+tab":
			if f.cycle(-1) {
				return formOpen, nil
			}
			f.move(-1)
			return formOpen, nil
		case "down":
			f.move(1)
			return formOpen, nil
		case "up":
			f.move(-1)
			return formOpen, nil
		}
		if f.isSelect(f.focus) {
			before := f.host()
			switch key.String() {
			case "left", "h":
				f.selects[f.focus].cycle(-1)
			case "right", "l":
				f.selects[f.focus].cycle(1)
			}
			if (f.focus == fieldHost && f.host() != before) || f.focus == fieldHoist {
				f.syncDir()
			}
			return formOpen, nil
		}
	}
	var cmd tea.Cmd
	f.inputs[f.focus], cmd = f.inputs[f.focus].Update(msg)
	f.suggest()
	return formOpen, cmd
}

func (f *form) move(delta int) {
	if !f.isSelect(f.focus) {
		f.inputs[f.focus].Blur()
	}
	step := delta
	if step == 0 {
		step = 1
	}
	f.focus = (f.focus + delta + fieldCount) % fieldCount
	for !f.visible(f.focus) {
		f.focus = (f.focus + step + fieldCount) % fieldCount
	}
	if !f.isSelect(f.focus) {
		f.inputs[f.focus].Focus()
		f.inputs[f.focus].CursorEnd()
	}
	f.suggest()
}

func (f *form) insert(text string, paths bool) {
	if f.isSelect(f.focus) {
		return
	}
	input := &f.inputs[f.focus]
	if paths && f.focus == fieldDir {
		input.SetValue(directoryOf(unquote(text)))
		input.CursorEnd()
		return
	}
	runes := []rune(input.Value())
	at := input.Position()
	if at < 0 {
		at = 0
	}
	if at > len(runes) {
		at = len(runes)
	}
	input.SetValue(string(runes[:at]) + text + string(runes[at:]))
	input.SetCursor(at + len([]rune(text)))
}

func unquote(text string) string {
	if tokens := splitTokens(text); len(tokens) > 0 {
		return tokens[0]
	}
	return text
}

func directoryOf(path string) string {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return path
	}
	return filepath.Dir(path)
}

func (f *form) validate() bool {
	dir := strings.TrimSpace(f.inputs[fieldDir].Value())
	if !f.localDir() {
		// A streamed session's directory is on the peer, so it is not resolved or
		// checked here. A blank directory means a temporary one there.
		f.err = ""
		return true
	}
	if dir == "" {
		f.err = "give a directory"
		return false
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		f.err = err.Error()
		return false
	}
	info, err := os.Stat(abs)
	if err != nil {
		f.err = "no such directory: " + abs
		return false
	}
	if !info.IsDir() {
		f.err = abs + " is not a directory"
		return false
	}
	f.inputs[fieldDir].SetValue(abs)
	f.err = ""
	return true
}

func (f *form) spec() manager.Spec {
	spec := manager.Spec{
		Dir:            strings.TrimSpace(f.inputs[fieldDir].Value()),
		Name:           strings.TrimSpace(f.inputs[fieldName].Value()),
		Model:          f.selects[fieldModel].value(),
		PermissionMode: f.selects[fieldMode].value(),
		Effort:         f.selects[fieldEffort].value(),
		Control:        f.selects[fieldControl].value() == "yes",
	}
	if f.hoisting() {
		spec.Lender = f.host()
	}
	return spec
}

func (f *form) firstPrompt() string {
	return strings.TrimSpace(f.inputs[fieldFirst].Value())
}

func (f *form) View(width int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("New session"))
	b.WriteString("\n\n")
	inner := modalInner(width)
	for i := range f.inputs {
		if !f.visible(i) {
			continue
		}
		b.WriteString(fieldLabelStyle.Render(pad(fieldLabels[i], 16)))
		if f.isSelect(i) {
			b.WriteString(f.selectView(i))
		} else {
			b.WriteString(f.inputs[i].View())
		}
		b.WriteString("\n")
		if i == fieldDir && f.focus == fieldDir {
			if !f.localDir() {
				b.WriteString(strings.Repeat(" ", 16) + hintStyle.Render("blank for a temporary directory on "+f.host()) + "\n")
			} else if hint := pathHint(f.matches, f.picked, inner-18); hint != "" {
				b.WriteString(strings.Repeat(" ", 16) + hintStyle.Render(hint) + "\n")
			}
		}
	}
	if f.err != "" {
		b.WriteString("\n" + errorStyle.Render(f.err) + "\n")
	}
	b.WriteString("\n" + hintStyle.Render(f.hint()))
	return modalStyle.Width(inner).Render(b.String())
}

func (f *form) selectView(i int) string {
	label := f.selects[i].label()
	if i == f.focus {
		return selectArrowStyle.Render("‹ ") + selectValueStyle.Render(label) + selectArrowStyle.Render(" ›")
	}
	return rowStyle.Render("  " + label)
}

func (f *form) hint() string {
	switch {
	case f.isSelect(f.focus):
		return "←→ change · ↑↓ move · enter start · esc cancel"
	case f.focus == fieldDir:
		return "tab complete · shift+tab walk · enter start · esc cancel"
	}
	return "↑↓ move · enter start · esc cancel"
}

func centre(width, height int, content string) string {
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}

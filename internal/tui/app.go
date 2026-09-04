package tui

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dextermb/claude-multiplexer/internal/manager"
	"github.com/dextermb/claude-multiplexer/internal/markdown"
	"github.com/dextermb/claude-multiplexer/internal/protocol"
	"github.com/dextermb/claude-multiplexer/internal/render"
	"github.com/dextermb/claude-multiplexer/internal/session"
	"github.com/dextermb/claude-multiplexer/internal/template"
)

type focusArea int

const (
	focusSidebar focusArea = iota
	focusPrompt
	focusOutput
)

type Options struct {
	Manager               *manager.Manager
	DefaultDir            string
	DefaultModel          string
	DefaultPermissionMode string
	InitialDir            string
	InitialControl        bool
}

type eventMsg manager.Event
type busClosedMsg struct{}
type spawnedMsg struct {
	name string
	err  error
}
type storedMsg struct {
	metas []manager.Meta
}
type stoppedMsg struct {
	name string
	err  error
}
type interruptedMsg struct {
	name string
	err  error
}
type shutdownDoneMsg struct{}
type spinTickMsg struct{}
type archivedMsg struct {
	name     string
	archived bool
	err      error
}

type Model struct {
	opts Options
	mgr  *manager.Manager
	sub  *manager.Subscription

	rows         []row
	groups       []group
	lines        []listLine
	folded       map[string]bool
	roots        map[string]string
	stored       []manager.Meta
	storedLoaded bool
	greeted      bool
	showArchived bool
	replays      map[string][]render.Line
	partials     map[string]string
	queued       map[string][]string
	todos        map[string][]protocol.Todo
	spinFrame    int
	animating    bool
	md           *markdown.Renderer
	showRaw      bool
	templates    []template.Template
	help         *help
	picker       *picker
	fields       *fieldForm
	questions    map[string]*questionDialog
	choice       *choiceDialog
	rename       *renameDialog
	pager        *pager
	jobsModal    *jobsModal
	pending      string
	sel          string
	listOffset   int

	output      viewport.Model
	outputText  string
	shownLines  []render.Line
	content     string
	selection   selRange
	prompt      textarea.Model
	pathMatches []pathMatch
	pathPicked  int
	pathStem    string
	pathValue   string
	pathBase    string
	form        *form
	confirm     string
	focus       focusArea

	width     int
	height    int
	ready     bool
	lastSeq   uint64
	mouseOn   bool
	status    string
	errText   string
	armedQuit bool
	quitting  bool
}

func New(opts Options) Model {
	if opts.DefaultDir == "" {
		if cwd, err := os.Getwd(); err == nil {
			opts.DefaultDir = cwd
		}
	}
	if opts.DefaultPermissionMode == "" {
		opts.DefaultPermissionMode = session.DefaultPermissionMode
	}

	prompt := textarea.New()
	prompt.Placeholder = "Type a prompt, then press Enter"
	prompt.Prompt = "> "
	prompt.ShowLineNumbers = false
	prompt.CharLimit = 0
	prompt.SetHeight(promptRowsMin)

	return Model{
		replays:    make(map[string][]render.Line),
		partials:   make(map[string]string),
		queued:     make(map[string][]string),
		todos:      make(map[string][]protocol.Todo),
		questions:  make(map[string]*questionDialog),
		folded:     make(map[string]bool),
		roots:      make(map[string]string),
		md:         markdown.New(),
		opts:       opts,
		mgr:        opts.Manager,
		sub:        opts.Manager.Subscribe(manager.DefaultSubscriberBuffer),
		output:     viewport.New(0, 0),
		prompt:     prompt,
		pathPicked: -1,
		focus:      focusSidebar,
		mouseOn:    true,
	}
}

func Run(opts Options) error {
	program := tea.NewProgram(New(opts), tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := program.Run()
	return err
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{waitEvent(m.sub), textarea.Blink, reloadStored(m.mgr)}
	if m.opts.InitialDir != "" {
		cmds = append(cmds, spawnCmd(m.mgr, manager.Spec{Dir: m.opts.InitialDir, Control: m.opts.InitialControl}))
	}
	return tea.Batch(cmds...)
}

func spinTick() tea.Cmd {
	return tea.Tick(spinInterval, func(time.Time) tea.Msg {
		return spinTickMsg{}
	})
}

func waitEvent(sub *manager.Subscription) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-sub.C
		if !ok {
			return busClosedMsg{}
		}
		return eventMsg(ev)
	}
}

func reloadStored(mgr *manager.Manager) tea.Cmd {
	return func() tea.Msg {
		return storedMsg{metas: mgr.Stored()}
	}
}

func resumeCmd(mgr *manager.Manager, meta manager.Meta) tea.Cmd {
	return func() tea.Msg {
		name, err := mgr.Resume(context.Background(), meta)
		return spawnedMsg{name: name, err: err}
	}
}

func resumeEffortCmd(mgr *manager.Manager, name, effort string) tea.Cmd {
	return func() tea.Msg {
		next, err := mgr.ResumeWithEffort(context.Background(), name, effort)
		return spawnedMsg{name: next, err: err}
	}
}

func archiveCmd(mgr *manager.Manager, name string, archived bool) tea.Cmd {
	return func() tea.Msg {
		if err := mgr.Archive(name, archived); err != nil {
			return archivedMsg{err: err}
		}
		return archivedMsg{name: name, archived: archived}
	}
}

func spawnCmd(mgr *manager.Manager, spec manager.Spec) tea.Cmd {
	return func() tea.Msg {
		name, err := mgr.Spawn(context.Background(), spec)
		return spawnedMsg{name: name, err: err}
	}
}

func stopCmd(mgr *manager.Manager, name string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), session.DefaultStopGrace)
		defer cancel()
		return stoppedMsg{name: name, err: mgr.Stop(ctx, name)}
	}
}

func interruptCmd(mgr *manager.Manager, name string, discard bool) tea.Cmd {
	return func() tea.Msg {
		return interruptedMsg{name: name, err: mgr.Interrupt(name, discard)}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.update(msg)
	model, ok := next.(Model)
	if !ok {
		return next, cmd
	}
	model.syncPromptHeight()
	model.syncMentions()
	return model, cmd
}

func (m Model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.resize(msg.Width, msg.Height)
	case eventMsg:
		return m.handleEvent(manager.Event(msg))
	case busClosedMsg:
		return m, nil
	case spawnedMsg:
		return m.handleSpawned(msg)
	case storedMsg:
		return m.handleStored(msg)
	case archivedMsg:
		if msg.err != nil {
			m.errText = msg.err.Error()
			return m, nil
		}
		m.status = "archived " + msg.name
		if !msg.archived {
			m.status = "restored " + msg.name
		}
		return m, reloadStored(m.mgr)
	case stoppedMsg:
		if msg.err != nil {
			m.errText = msg.err.Error()
		}
		m.refresh()
		return m, nil
	case interruptedMsg:
		if msg.err != nil {
			m.errText = msg.err.Error()
		}
		m.refresh()
		return m, nil
	case shutdownDoneMsg:
		return m, tea.Quit
	case spinTickMsg:
		return m.handleSpin()
	case bashResultMsg:
		return m.handleBash(msg)
	case tea.MouseMsg:
		return m.handleMouse(msg)
	case tea.KeyMsg:
		if isMouseArtifact(msg) {
			return m, nil
		}
		if msg.Paste {
			return m.handlePaste(string(msg.Runes))
		}
		return m.handleKey(msg)
	}

	var cmd tea.Cmd
	if m.focus == focusPrompt && m.form == nil {
		m.prompt, cmd = m.prompt.Update(msg)
	}
	return m, cmd
}

func (m Model) resize(width, height int) (tea.Model, tea.Cmd) {
	m.width = width
	m.height = height
	m.ready = true

	m.output.Width = m.outputWidth()
	m.output.Height = m.outputHeight()
	m.prompt.SetWidth(width - gutterWidth)
	m.rebuildOutput()

	return m, m.maybeOpenForm()
}

func (m *Model) maybeOpenForm() tea.Cmd {
	if !m.ready || !m.storedLoaded || m.greeted {
		return nil
	}
	m.greeted = true
	if m.form != nil || len(m.rows) > 0 || m.opts.InitialDir != "" {
		return nil
	}
	m.form = newForm(m.opts.DefaultDir, m.opts.DefaultModel, m.opts.DefaultPermissionMode)
	return textarea.Blink
}

func (m Model) handleStored(msg storedMsg) (tea.Model, tea.Cmd) {
	m.stored = msg.metas
	m.storedLoaded = true
	m.refresh()
	if m.sel != "" {
		m.rebuildOutput()
	}
	return m, m.maybeOpenForm()
}

func (m Model) handleEvent(ev manager.Event) (tea.Model, tea.Cmd) {
	gap := m.lastSeq != 0 && ev.Seq != m.lastSeq+1
	m.lastSeq = ev.Seq
	if ev.Notice != "" || ev.Reload {
		return m.handleNotice(ev)
	}
	prevWidth := m.outputWidth()
	m.setPartial(ev.Session, ev.Partial)
	if !ev.Closed {
		m.todos[ev.Session] = ev.Todos
	}
	if hasPrompt(ev.Lines) {
		m.dropQueued(ev.Session)
	}
	m.refresh()

	widthChanged := ev.Session == m.sel && m.outputWidth() != prevWidth
	switch {
	case gap || widthChanged:
		m.rebuildOutput()
	case ev.Session == m.sel && len(ev.Lines) > 0:
		m.appendOutput(ev.Lines)
	case ev.Session == m.sel:
		m.setContent()
	}
	cmds := []tea.Cmd{waitEvent(m.sub)}
	if cmd := m.maybeAskQuestion(ev); cmd != nil {
		cmds = append(cmds, cmd)
	}
	if spin := m.ensureAnimating(); spin != nil {
		cmds = append(cmds, spin)
	}
	if ev.Closed {
		cmds = append(cmds, reloadStored(m.mgr))
	}
	return m, tea.Batch(cmds...)
}

// handleNotice takes a change an MCP tool made. It must not reach the normal
// path, because that path clears the streaming text of a session whose event
// carries none. See docs/mcp.md.
func (m Model) handleNotice(ev manager.Event) (tea.Model, tea.Cmd) {
	if ev.Notice != "" {
		m.status = ev.Notice
	}
	m.refresh()
	cmds := []tea.Cmd{waitEvent(m.sub)}
	if ev.Reload {
		cmds = append(cmds, reloadStored(m.mgr))
	}
	if ev.Session == m.sel {
		m.setContent()
	}
	return m, tea.Batch(cmds...)
}

func hasPrompt(lines []render.Line) bool {
	for _, line := range lines {
		if line.Class == render.ClassPrompt {
			return true
		}
	}
	return false
}

func (m *Model) dropQueued(name string) {
	if len(m.queued[name]) == 0 {
		return
	}
	rest := m.queued[name][1:]
	if len(rest) == 0 {
		delete(m.queued, name)
		return
	}
	m.queued[name] = rest
}

func (m *Model) maybeAskQuestion(ev manager.Event) tea.Cmd {
	if len(ev.Questions) == 0 {
		return nil
	}
	if m.questions[ev.Session] != nil {
		m.status = "another question waits for " + ev.Session
		return nil
	}
	m.questions[ev.Session] = newQuestionDialog(ev.Session, ev.Questions)
	m.status = ev.Session + " asks a question"
	return textinput.Blink
}

func (m Model) questionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab", "shift+tab":
		return m.toggleFocus()
	}
	result, cmd := m.questions[m.sel].Update(msg)
	switch result {
	case formCancelled:
		delete(m.questions, m.sel)
		m.status = "question dismissed"
		m.setContent()
		return m, nil
	case formSubmitted:
		return m.submitQuestion()
	}
	return m, cmd
}

func (m Model) submitQuestion() (tea.Model, tea.Cmd) {
	name := m.sel
	text := m.questions[name].answer()
	delete(m.questions, name)
	if err := m.mgr.Send(name, text); err != nil {
		m.errText = err.Error()
		return m, nil
	}
	m.errText = ""
	m.status = "answered"
	m.queued[name] = append(m.queued[name], text)
	m.refresh()
	m.setContent()
	m.focus = focusPrompt
	m.prompt.Focus()
	return m, tea.Batch(m.ensureAnimating(), textarea.Blink)
}

func (m Model) handleSpawned(msg spawnedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errText = msg.err.Error()
		return m, nil
	}
	m.errText = ""
	m.status = "started " + msg.name
	delete(m.replays, msg.name)
	m.todos[msg.name] = m.mgr.Todos(msg.name)
	m.refresh()
	m.sel = msg.name
	m.revealSelection()
	m.rebuildOutput()
	m.focus = focusPrompt
	m.prompt.Focus()

	if m.pending != "" {
		text := m.pending
		m.pending = ""
		next, cmd := m.dispatch(text)
		return next, tea.Batch(cmd, textarea.Blink, reloadStored(m.mgr))
	}
	return m, tea.Batch(textarea.Blink, reloadStored(m.mgr))
}

func looksDropped(msg tea.KeyMsg) bool {
	return msg.Type == tea.KeyRunes && len(msg.Runes) > 1 &&
		strings.ContainsRune(string(msg.Runes), '/')
}

func (m Model) handlePaste(raw string) (tea.Model, tea.Cmd) {
	if m.quitting || m.confirm != "" {
		return m, nil
	}
	m.armedQuit = false
	text, paths := dropText(raw)
	if text == "" {
		return m, nil
	}
	if m.form != nil {
		m.form.insert(text, paths)
		return m, nil
	}
	if paths {
		text += " "
	}
	if m.focus != focusPrompt {
		m.focus = focusPrompt
		m.prompt.Focus()
	}
	value := m.prompt.Value()
	if value != "" && !strings.HasSuffix(value, " ") && !strings.HasSuffix(value, "\n") {
		m.prompt.InsertString(" ")
	}
	m.prompt.InsertString(text)
	return m, textarea.Blink
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.quitting {
		return m, nil
	}
	if msg.String() == "ctrl+c" {
		return m.interrupt()
	}
	m.armedQuit = false
	if m.help != nil {
		open, cmd := m.help.Update(msg)
		if !open {
			m.help = nil
		}
		return m, cmd
	}
	if q := m.questions[m.sel]; q != nil && m.focus == focusOutput {
		return m.questionKey(msg)
	}
	if m.choice != nil {
		return m.choiceKey(msg)
	}
	if m.rename != nil {
		return m.renameKey(msg)
	}
	if m.pager != nil {
		return m.pagerKey(msg)
	}
	if m.jobsModal != nil {
		return m.jobsModalKey(msg)
	}
	if m.picker != nil {
		return m.pickerKey(msg)
	}
	if m.fields != nil {
		return m.fieldsKey(msg)
	}
	if m.form != nil {
		result, cmd := m.form.Update(msg)
		switch result {
		case formCancelled:
			m.form = nil
			return m, nil
		case formSubmitted:
			spec := m.form.spec()
			m.pending = m.form.firstPrompt()
			m.form = nil
			return m, spawnCmd(m.mgr, spec)
		}
		return m, cmd
	}
	if m.confirm != "" {
		switch msg.String() {
		case "y", "enter":
			name := m.confirm
			m.confirm = ""
			return m, stopCmd(m.mgr, name)
		default:
			m.confirm = ""
			return m, nil
		}
	}

	switch msg.String() {
	case "ctrl+p":
		return m.openPicker()
	case "ctrl+n":
		m.form = newForm(m.opts.DefaultDir, m.opts.DefaultModel, m.opts.DefaultPermissionMode)
		return m, textinputBlink()
	case "ctrl+x":
		return m.askToStop()
	case "ctrl+t":
		m.mouseOn = !m.mouseOn
		if m.mouseOn {
			return m, tea.EnableMouseCellMotion
		}
		return m, tea.DisableMouse
	case "tab":
		if m.focus == focusPrompt {
			return m.complete()
		}
		return m.toggleFocus()
	case "pgup":
		m.output.ViewUp()
		return m, nil
	case "pgdown":
		m.output.ViewDown()
		return m, nil
	}

	if m.focus != focusPrompt && looksDropped(msg) {
		return m.handlePaste(string(msg.Runes))
	}

	switch m.focus {
	case focusPrompt:
		return m.promptKey(msg)
	case focusOutput:
		return m.outputKey(msg)
	}
	return m.sidebarKey(msg)
}

func (m Model) promptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if next, cmd, ok := m.stopBusy(); ok {
			return next, cmd
		}
		m.focus = focusSidebar
		m.prompt.Blur()
		return m, nil
	case "enter":
		return m.send()
	case "ctrl+j":
		var cmd tea.Cmd
		m.prompt, cmd = m.prompt.Update(tea.KeyMsg{Type: tea.KeyEnter})
		return m, cmd
	}
	if next, cmd, ok := m.mentionKey(msg); ok {
		return next, cmd
	}
	var cmd tea.Cmd
	m.prompt, cmd = m.prompt.Update(msg)
	return m, cmd
}

func (m Model) pagerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	open, cmd := m.pager.Update(msg)
	if !open {
		m.pager = nil
	}
	return m, cmd
}

func (m Model) jobsModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	open, cmd := m.jobsModal.Update(msg)
	if !open {
		m.jobsModal = nil
	}
	return m, cmd
}

func (m Model) openJobs() (tea.Model, tea.Cmd) {
	item, ok := m.selectedRow()
	if !ok {
		return m, nil
	}
	m.jobsModal = newJobsModal(item.displayName(), item.jobList, m.baseOutputWidth(), m.outputHeight())
	return m, nil
}

func (m Model) outputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.clearSelection()
	switch msg.String() {
	case "esc":
		if next, cmd, ok := m.stopBusy(); ok {
			return next, cmd
		}
		m.focus = focusSidebar
		return m, nil
	case "enter":
		if entries := collectExpandables(m.shownLines); len(entries) > 0 {
			m.pager = newPager(entries, m.baseOutputWidth(), m.outputHeight())
			return m, nil
		}
		m.focus = focusPrompt
		m.prompt.Focus()
		return m, textarea.Blink
	case "i":
		m.focus = focusPrompt
		m.prompt.Focus()
		return m, textarea.Blink
	case "up", "k":
		m.output.LineUp(1)
	case "down", "j":
		m.output.LineDown(1)
	case "u", "ctrl+u":
		m.output.HalfViewUp()
	case "d", "ctrl+d":
		m.output.HalfViewDown()
	case "g", "home":
		m.output.GotoTop()
	case "G", "end":
		m.output.GotoBottom()
	case "m":
		return m.toggleMarkdown()
	case "M":
		return m.openChoice(settingModel)
	case "e":
		return m.openChoice(settingEffort)
	case "p":
		return m.openChoice(settingMode)
	case "J":
		return m.openJobs()
	case "?":
		m.help = newHelp()
		return m, textinput.Blink
	case "q":
		return m.startQuit()
	}
	return m, nil
}

func (m Model) toggleMarkdown() (tea.Model, tea.Cmd) {
	m.showRaw = !m.showRaw
	m.status = "markdown on"
	if m.showRaw {
		m.status = "raw text"
	}
	m.rebuildOutput()
	return m, nil
}

func (m Model) sidebarKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m.startQuit()
	case "n":
		m.form = newForm(m.opts.DefaultDir, m.opts.DefaultModel, m.opts.DefaultPermissionMode)
		return m, textinputBlink()
	case "x":
		return m.askToStop()
	case "up", "k":
		return m.move(-1)
	case "down", "j":
		return m.move(1)
	case "r":
		return m.resumeSelected()
	case "a":
		return m.archiveSelected()
	case "R":
		return m.openRename()
	case "m":
		return m.toggleMarkdown()
	case "M":
		return m.openChoice(settingModel)
	case "e":
		return m.openChoice(settingEffort)
	case "p":
		return m.openChoice(settingMode)
	case "?":
		m.help = newHelp()
		return m, textinput.Blink
	case "J":
		return m.openJobs()
	case "z":
		return m.toggleFold()
	case "Z":
		return m.toggleAllFolds()
	case "t":
		return m.openPicker()
	case "A":
		m.showArchived = !m.showArchived
		m.status = "archived sessions hidden"
		if m.showArchived {
			m.status = "archived sessions shown"
		}
		m.refresh()
		m.rebuildOutput()
		return m, nil
	case "enter", "i":
		if item, ok := m.selectedRow(); ok && !item.running() {
			return m.resumeSelected()
		}
		m.focus = focusPrompt
		m.prompt.Focus()
		return m, textarea.Blink
	}
	return m, nil
}

func (m Model) resumeSelected() (tea.Model, tea.Cmd) {
	item, ok := m.selectedRow()
	if !ok {
		return m, nil
	}
	if item.running() {
		m.focus = focusPrompt
		m.prompt.Focus()
		return m, textarea.Blink
	}
	meta := item.meta
	if meta.ClaudeSessionID == "" {
		stored, err := m.mgr.Meta(item.name)
		if err != nil {
			m.errText = "this session has nothing to resume from"
			return m, nil
		}
		meta = stored
	}
	m.errText = ""
	m.status = "resuming " + item.name
	return m, resumeCmd(m.mgr, meta)
}

func (m Model) archiveSelected() (tea.Model, tea.Cmd) {
	item, ok := m.selectedRow()
	if !ok {
		return m, nil
	}
	if item.running() {
		m.errText = "stop the session before you archive it"
		return m, nil
	}
	m.errText = ""
	return m, archiveCmd(m.mgr, item.name, !item.archived)
}

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.form != nil || m.confirm != "" || m.quitting || m.questions[m.sel] != nil || m.choice != nil || m.rename != nil || m.pager != nil || m.jobsModal != nil {
		return m, nil
	}

	switch msg.Button {
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}
		if msg.X < sidebarWidth {
			if msg.Button == tea.MouseButtonWheelUp {
				return m.move(-1)
			}
			return m.move(1)
		}
		var cmd tea.Cmd
		m.output, cmd = m.output.Update(msg)
		return m, cmd
	case tea.MouseButtonLeft:
		return m.handleLeftMouse(msg)
	}
	return m, nil
}

func (m Model) handleLeftMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Y >= m.bodyHeight() && msg.Y < m.bodyHeight()+m.promptHeight() {
			m.clearSelection()
			m.focus = focusPrompt
			m.prompt.Focus()
			return m, textarea.Blink
		}
		if p, ok := m.outputPos(msg.X, msg.Y); ok {
			m.focus = focusOutput
			m.prompt.Blur()
			m.selection = selRange{active: true, dragging: true, anchor: p, cursor: p}
			m.setContent()
			return m, nil
		}
		if msg.X >= sidebarWidth {
			m.clearSelection()
			m.focus = focusOutput
			m.prompt.Blur()
			return m, nil
		}
		m.clearSelection()
		index := m.listOffset + msg.Y - titleHeight
		if msg.Y >= titleHeight && index >= 0 && index < len(m.lines) {
			line := m.lines[index]
			m.focus = focusSidebar
			m.prompt.Blur()
			if line.header() {
				m.setFold(m.groups[line.group].key, !m.groups[line.group].folded)
				return m, nil
			}
			m.sel = m.rows[line.row].name
			m.rebuildOutput()
		}
		return m, nil
	case tea.MouseActionMotion:
		if m.selection.dragging {
			if p, ok := m.outputPos(msg.X, msg.Y); ok {
				m.selection.cursor = p
				m.setContent()
			}
		}
		return m, nil
	case tea.MouseActionRelease:
		if m.selection.dragging {
			m.selection.dragging = false
			if p, ok := m.outputPos(msg.X, msg.Y); ok {
				m.selection.cursor = p
			}
			if m.selection.empty() {
				m.selection.active = false
			}
			m.copySelection()
			m.setContent()
		}
		return m, nil
	}
	return m, nil
}

func (m Model) outputPos(x, y int) (pos, bool) {
	if x < sidebarWidth+gutterWidth {
		return pos{}, false
	}
	row := y - barHeight
	if row < 0 || row >= m.outputHeight() {
		return pos{}, false
	}
	return pos{line: m.output.YOffset + row, col: x - sidebarWidth - gutterWidth}, true
}

func (m *Model) clearSelection() {
	if !m.selection.active {
		return
	}
	m.selection = selRange{}
	m.setContent()
}

func (m *Model) copySelection() {
	if m.selection.empty() {
		return
	}
	a, b := m.selection.bounds()
	text := selectedText(plainLines(m.content), a, b)
	if strings.TrimSpace(text) == "" {
		return
	}
	if err := copyToClipboard(text); err != nil {
		m.errText = "could not copy: " + err.Error()
		return
	}
	m.errText = ""
	m.status = "copied " + plural(strings.Count(text, "\n")+1, "line")
}

func (m Model) highlight(content string) string {
	if !m.selection.active || m.selection.empty() {
		return content
	}
	a, b := m.selection.bounds()
	styled := strings.Split(content, "\n")
	plain := plainLines(content)
	for i := a.line; i <= b.line && i < len(styled) && i < len(plain); i++ {
		if i < 0 {
			continue
		}
		runes := []rune(plain[i])
		from, to := 0, len(runes)
		if i == a.line {
			from = a.col
		}
		if i == b.line {
			to = b.col
		}
		if from < 0 {
			from = 0
		}
		if from > len(runes) {
			from = len(runes)
		}
		if to > len(runes) {
			to = len(runes)
		}
		if to < from {
			to = from
		}
		styled[i] = string(runes[:from]) + selectionStyle.Render(string(runes[from:to])) + string(runes[to:])
	}
	return strings.Join(styled, "\n")
}

func (m Model) move(delta int) (tea.Model, tea.Cmd) {
	visible := m.visibleRowIndexes()
	if len(visible) == 0 {
		return m, nil
	}
	index := 0
	for i, at := range visible {
		if m.rows[at].name == m.sel {
			index = i + delta
			break
		}
	}
	if index < 0 {
		index = 0
	}
	if index >= len(visible) {
		index = len(visible) - 1
	}
	name := m.rows[visible[index]].name
	if name == m.sel {
		return m, nil
	}
	m.sel = name
	m.rebuildOutput()
	m.clampOffset()
	return m, nil
}

func (m Model) toggleFold() (tea.Model, tea.Cmd) {
	index := m.selGroup()
	if index < 0 {
		return m, nil
	}
	m.setFold(m.groups[index].key, !m.groups[index].folded)
	return m, nil
}

// toggleAllFolds unfolds every group when one is folded, and otherwise folds
// every group but the one that holds the selection.
func (m Model) toggleAllFolds() (tea.Model, tea.Cmd) {
	if len(m.groups) == 0 {
		return m, nil
	}
	unfold := false
	for _, item := range m.groups {
		if item.folded {
			unfold = true
			break
		}
	}
	keep := ""
	if index := m.selGroup(); index >= 0 {
		keep = m.groups[index].key
	}
	if m.folded == nil {
		m.folded = make(map[string]bool)
	}
	for _, item := range m.groups {
		if unfold || item.key == keep {
			delete(m.folded, item.key)
			continue
		}
		m.folded[item.key] = true
	}
	m.status = "groups folded"
	if unfold {
		m.status = "groups unfolded"
	}
	m.applyFolds(keep)
	return m, nil
}

func (m Model) toggleFocus() (tea.Model, tea.Cmd) {
	switch m.focus {
	case focusSidebar:
		m.focus = focusPrompt
		m.prompt.Focus()
		return m, textarea.Blink
	case focusPrompt:
		m.focus = focusOutput
		m.prompt.Blur()
		return m, nil
	}
	m.focus = focusSidebar
	return m, nil
}

func (m Model) send() (tea.Model, tea.Cmd) {
	text := strings.TrimSpace(m.prompt.Value())
	item, ok := m.selectedRow()
	if !ok {
		if text == "" {
			return m, nil
		}
		m.errText = "no session is selected"
		return m, nil
	}
	if text == "" {
		if item.state == session.StateBusy && len(m.queued[m.sel]) > 0 {
			m.errText = ""
			m.status = "sending now…"
			return m, interruptCmd(m.mgr, m.sel, false)
		}
		return m, nil
	}
	if !item.running() {
		m.errText = "this session is not running — press Enter to resume it"
		return m, nil
	}
	m.prompt.Reset()
	return m.dispatch(text)
}

func (m Model) stopBusy() (tea.Model, tea.Cmd, bool) {
	item, ok := m.selectedRow()
	if !ok || item.state != session.StateBusy {
		return m, nil, false
	}
	delete(m.queued, m.sel)
	m.errText = ""
	m.status = "interrupted"
	m.refresh()
	m.setContent()
	return m, interruptCmd(m.mgr, m.sel, true), true
}

func (m Model) dispatch(text string) (tea.Model, tea.Cmd) {
	if command, feed, ok := parseBang(text); ok {
		return m.runBash(command, feed)
	}
	if name, args, ok := template.ParseInvocation(text); ok {
		m.reloadTemplates()
		if tpl, found := template.Find(m.templates, name); found {
			values, missing := tpl.Fill(args)
			if len(missing) > 0 {
				m.status = "fill in /" + tpl.Name
				return m.openFields(tpl, values)
			}
			text = tpl.Expand(values)
		}
	}
	if err := m.mgr.Send(m.sel, text); err != nil {
		m.errText = err.Error()
		return m, nil
	}
	m.errText = ""
	m.queued[m.sel] = append(m.queued[m.sel], text)
	m.refresh()
	m.setContent()
	return m, m.ensureAnimating()
}

func (m Model) askToStop() (tea.Model, tea.Cmd) {
	item, ok := m.selectedRow()
	if !ok {
		return m, nil
	}
	if !item.running() {
		m.errText = "that session is not running"
		return m, nil
	}
	m.errText = ""
	m.confirm = item.name
	return m, nil
}

func (m Model) openChoice(kind settingKind) (tea.Model, tea.Cmd) {
	item, ok := m.selectedRow()
	if !ok {
		return m, nil
	}
	if !item.running() {
		m.errText = "start the session before you change it"
		return m, nil
	}
	current := item.model
	switch kind {
	case settingMode:
		current = item.mode
	case settingEffort:
		current = item.effort
	}
	m.choice = newChoiceDialog(kind, item.name, current)
	m.errText = ""
	return m, nil
}

func (m Model) choiceKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	result, cmd := m.choice.Update(msg)
	switch result {
	case formCancelled:
		m.choice = nil
		return m, nil
	case formSubmitted:
		return m.submitChoice()
	}
	return m, cmd
}

func (m Model) submitChoice() (tea.Model, tea.Cmd) {
	kind, name, value := m.choice.kind, m.choice.session, m.choice.chosen()
	m.choice = nil
	switch kind {
	case settingModel:
		return m.appliedSetting(m.mgr.SetModel(name, value), "model", value)
	case settingMode:
		return m.appliedSetting(m.mgr.SetPermissionMode(name, value), "mode", value)
	case settingEffort:
		m.errText = ""
		m.status = "resuming " + name + " with " + value + " effort"
		return m, resumeEffortCmd(m.mgr, name, value)
	}
	return m, nil
}

func (m Model) openRename() (tea.Model, tea.Cmd) {
	item, ok := m.selectedRow()
	if !ok {
		m.errText = "no session is selected"
		return m, nil
	}
	m.rename = newRenameDialog(item.name, item.title)
	m.errText = ""
	return m, textinput.Blink
}

func (m Model) renameKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	result, cmd := m.rename.Update(msg)
	switch result {
	case formCancelled:
		m.rename = nil
		return m, nil
	case formSubmitted:
		return m.submitRename()
	}
	return m, cmd
}

func (m Model) submitRename() (tea.Model, tea.Cmd) {
	name, title := m.rename.session, m.rename.value()
	m.rename = nil
	if err := m.mgr.SetTitle(name, title); err != nil {
		m.errText = err.Error()
		return m, nil
	}
	for i := range m.stored {
		if m.stored[i].Name == name {
			m.stored[i].Title = title
		}
	}
	m.errText = ""
	m.status = "renamed " + name
	m.refresh()
	m.rebuildOutput()
	return m, nil
}

func (m Model) appliedSetting(err error, label, value string) (tea.Model, tea.Cmd) {
	if err != nil {
		m.errText = err.Error()
		return m, nil
	}
	m.errText = ""
	m.status = label + " → " + value
	m.refresh()
	return m, nil
}

func (m Model) openPicker() (tea.Model, tea.Cmd) {
	m.reloadTemplates()
	m.picker = newPicker(m.templates, m.templateDirs())
	m.errText = ""
	return m, textinput.Blink
}

func (m Model) pickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	result, cmd := m.picker.Update(msg)
	switch result {
	case pickerCancelled:
		m.picker = nil
		return m, nil
	case pickerChosen:
		tpl, ok := m.picker.selected()
		m.picker = nil
		if !ok {
			return m, nil
		}
		return m.openFields(tpl, nil)
	}
	return m, cmd
}

func (m Model) openFields(tpl template.Template, values map[string]string) (tea.Model, tea.Cmd) {
	if len(tpl.Fields) == 0 {
		return m.fillPrompt(tpl.Expand(nil))
	}
	m.fields = newFieldForm(tpl, values)
	return m, textinput.Blink
}

func (m Model) fieldsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	result, cmd := m.fields.Update(msg)
	switch result {
	case formCancelled:
		m.fields = nil
		return m, nil
	case formSubmitted:
		text := m.fields.prompt()
		m.fields = nil
		return m.fillPrompt(text)
	}
	return m, cmd
}

func (m Model) fillPrompt(text string) (tea.Model, tea.Cmd) {
	m.prompt.SetValue(text)
	m.prompt.CursorEnd()
	m.focus = focusPrompt
	m.prompt.Focus()
	return m, textarea.Blink
}

func (m Model) complete() (tea.Model, tea.Cmd) {
	m.reloadTemplates()
	if names := completionNames(m.templates, m.prompt.Value()); len(names) > 0 {
		m.prompt.SetValue(names[0] + " ")
		m.prompt.CursorEnd()
		return m, nil
	}
	if m.pathPicked >= 0 {
		if next, ok := m.walkMention(1); ok {
			return next, nil
		}
	}
	if next, ok := m.completeMention(); ok {
		return next, nil
	}
	return m.toggleFocus()
}

func (m *Model) reloadTemplates() {
	m.templates = template.Load(m.templateDirs()...)
}

func (m Model) templateDirs() []string {
	dir := ""
	if item, ok := m.selectedRow(); ok {
		dir = item.dir
	}
	return template.Dirs(m.mgr.Root(), dir)
}

func (m Model) interrupt() (tea.Model, tea.Cmd) {
	if m.armedQuit {
		return m.startQuit()
	}
	m.armedQuit = true
	m.errText = ""
	m.form = nil
	m.fields = nil
	m.picker = nil
	m.help = nil
	m.questions = map[string]*questionDialog{}
	m.choice = nil
	m.rename = nil
	m.pager = nil
	m.jobsModal = nil
	m.confirm = ""
	m.prompt.Reset()
	m.status = "press ctrl+c again to quit"
	return m, nil
}

func (m Model) startQuit() (tea.Model, tea.Cmd) {
	m.quitting = true
	m.status = "stopping every session…"
	mgr := m.mgr
	return m, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), session.DefaultStopGrace)
		defer cancel()
		mgr.Shutdown(ctx)
		return shutdownDoneMsg{}
	}
}

func textinputBlink() tea.Cmd {
	return textarea.Blink
}

func (m *Model) refresh() {
	rows := make([]row, 0, len(m.stored)+4)
	grants := m.mgr.Grants()
	parents := m.mgr.Parents()
	for _, snap := range m.mgr.Snapshots() {
		item := rowFromSnapshot(snap)
		item.control = grants[snap.Name]
		item.parent = parents[snap.Name]
		rows = append(rows, item)
	}
	for _, meta := range m.stored {
		if meta.Archived && !m.showArchived {
			continue
		}
		rows = append(rows, rowFromMeta(meta))
	}
	children := make(map[string]bool, len(rows))
	for _, item := range rows {
		if item.parent != "" {
			children[item.parent] = true
		}
	}
	for i := range rows {
		rows[i].group = m.rowGroup(rows[i], children)
	}
	m.rows, m.groups = groupRows(rows, m.folded)
	m.buildLines()

	if m.selLine() < 0 {
		m.selectFirst()
	}
	if len(m.rows) == 0 {
		m.sel = ""
	}
	m.clampOffset()
}

// rowGroup keys a row on the control session that created it, or that it
// created rows for, and on its repository when neither holds.
func (m *Model) rowGroup(item row, children map[string]bool) string {
	if item.parent != "" {
		return byPrefix + item.parent
	}
	if children[item.name] {
		return byPrefix + item.name
	}
	return dirPrefix + m.groupKey(item.dir)
}

// groupKey caches the walk to the repository, because refresh runs on every
// event and a session directory never changes.
func (m *Model) groupKey(dir string) string {
	if dir == "" {
		return ""
	}
	if root, ok := m.roots[dir]; ok {
		return root
	}
	root := repoRoot(dir)
	if m.roots == nil {
		m.roots = make(map[string]string)
	}
	m.roots[dir] = root
	return root
}

func (m *Model) buildLines() {
	m.lines = listLines(m.rows, m.groups)
}

// selLine is the line the selected session is drawn on, or -1 when a fold hides
// it or nothing is selected.
func (m Model) selLine() int {
	for i, line := range m.lines {
		if !line.header() && m.rows[line.row].name == m.sel {
			return i
		}
	}
	return -1
}

func (m Model) selGroup() int {
	for _, item := range m.rows {
		if item.name != m.sel {
			continue
		}
		for i := range m.groups {
			if m.groups[i].key == item.group {
				return i
			}
		}
	}
	return -1
}

func (m *Model) selectFirst() {
	for _, line := range m.lines {
		if !line.header() {
			m.sel = m.rows[line.row].name
			return
		}
	}
	if len(m.rows) == 0 {
		return
	}
	m.setFold(m.rows[0].group, false)
	m.sel = m.rows[0].name
}

// selectNear takes the selection to the row after a group that just folded, or
// to the row before it when the group is the last one.
func (m *Model) selectNear(key string) {
	at := -1
	for i, line := range m.lines {
		if line.header() && m.groups[line.group].key == key {
			at = i
			break
		}
	}
	if at < 0 {
		m.selectFirst()
		return
	}
	for i := at + 1; i < len(m.lines); i++ {
		if !m.lines[i].header() {
			m.sel = m.rows[m.lines[i].row].name
			return
		}
	}
	for i := at - 1; i >= 0; i-- {
		if !m.lines[i].header() {
			m.sel = m.rows[m.lines[i].row].name
			return
		}
	}
}

func (m *Model) setFold(key string, folded bool) {
	if m.folded == nil {
		m.folded = make(map[string]bool)
	}
	if folded {
		m.folded[key] = true
	} else {
		delete(m.folded, key)
	}
	m.applyFolds(key)
}

func (m *Model) applyFolds(key string) {
	for i := range m.groups {
		m.groups[i].folded = m.folded[m.groups[i].key]
	}
	m.buildLines()
	previous := m.sel
	if m.selLine() < 0 {
		m.selectNear(key)
	}
	if m.sel != previous {
		m.rebuildOutput()
	}
	m.clampOffset()
}

// revealSelection unfolds the group of the selected session, so a session that
// starts in a folded group is visible.
func (m *Model) revealSelection() {
	for _, item := range m.rows {
		if item.name == m.sel && m.folded[item.group] {
			m.setFold(item.group, false)
			return
		}
	}
	m.clampOffset()
}

func (m *Model) clampOffset() {
	visible := m.visibleLines()
	index := m.selLine()
	if index >= 0 && visible > 0 {
		if index < m.listOffset {
			m.listOffset = index
		}
		if index >= m.listOffset+visible {
			m.listOffset = index - visible + 1
		}
	}
	if last := len(m.lines) - visible; m.listOffset > last {
		m.listOffset = last
	}
	if m.listOffset < 0 {
		m.listOffset = 0
	}
}

func (m Model) selIndex() int {
	for i, item := range m.rows {
		if item.name == m.sel {
			return i
		}
	}
	return -1
}

func (m Model) selectedRow() (row, bool) {
	index := m.selIndex()
	if index < 0 {
		return row{}, false
	}
	return m.rows[index], true
}

func (m Model) bodyHeight() int {
	height := m.height - m.promptHeight() - statusHeight
	if height < 1 {
		return 1
	}
	return height
}

func (m Model) outputHeight() int {
	height := m.bodyHeight() - barHeight
	if height < 1 {
		return 1
	}
	return height
}

func (m Model) visibleLines() int {
	lines := m.bodyHeight() - titleHeight
	if lines < 1 {
		return 1
	}
	return lines
}

// visibleRowIndexes lists the rows a fold does not hide, in the order they are
// drawn, as indexes into m.rows.
func (m Model) visibleRowIndexes() []int {
	out := make([]int, 0, len(m.rows))
	for _, line := range m.lines {
		if !line.header() {
			out = append(out, line.row)
		}
	}
	return out
}

func (m Model) baseOutputWidth() int {
	width := m.width - sidebarWidth - gutterWidth
	if width < 10 {
		return 10
	}
	return width
}

func (m Model) outputWidth() int {
	if m.showSidePanel() {
		return m.baseOutputWidth() - taskPanelWidth
	}
	return m.baseOutputWidth()
}

// showSidePanel says whether the side panel has room beside the output. It reads
// baseOutputWidth, not outputWidth, because outputWidth depends on it. See
// docs/tui/tasks.md.
func (m Model) showSidePanel() bool {
	if len(m.todos[m.sel]) == 0 && len(m.selectedJobs()) == 0 {
		return false
	}
	return m.baseOutputWidth()-taskPanelWidth >= minOutputWithPanel
}

func (m Model) selectedJobs() []session.Job {
	if item, ok := m.selectedRow(); ok {
		return item.jobList
	}
	return nil
}

func (m *Model) rebuildOutput() {
	m.selection = selRange{}
	if m.sel == "" {
		m.outputText = ""
		m.shownLines = nil
		m.output.SetContent("")
		return
	}
	lines := m.linesFor(m.sel)
	m.output.Width = m.outputWidth()
	m.shownLines = append([]render.Line(nil), lines...)
	m.outputText = m.wrap(lines)
	m.setContent()
	m.output.GotoBottom()
}

func (m *Model) appendOutput(lines []render.Line) {
	m.shownLines = append(m.shownLines, lines...)
	chunk := m.wrap(lines)
	if m.outputText == "" {
		m.outputText = chunk
	} else {
		m.outputText += "\n" + chunk
	}
	m.setContent()
}

func (m *Model) setPartial(name, text string) {
	if text == "" {
		delete(m.partials, name)
		return
	}
	m.partials[name] = text
}

func (m *Model) setContent() {
	atBottom := m.output.AtBottom()
	m.content = m.outputText + m.liveView()
	m.output.SetContent(m.highlight(m.content))
	if atBottom {
		m.output.GotoBottom()
	}
}

func (m Model) liveView() string {
	var parts []string
	if partial := m.partials[m.sel]; partial != "" {
		parts = append(parts, classStyle(render.ClassText).Width(m.outputWidth()).Render(partial+cursorMark))
	}
	for _, text := range m.queued[m.sel] {
		parts = append(parts, m.wrap(render.PromptLines(text)))
	}
	if m.thinkingSelected() {
		parts = append(parts, spinnerStyle.Render(spinnerFrame(m.spinFrame)+" thinking…"))
	}
	if len(parts) == 0 {
		return ""
	}
	body := strings.Join(parts, "\n")
	if m.outputText == "" {
		return body
	}
	return "\n" + body
}

func (m Model) thinkingSelected() bool {
	if m.sel == "" || m.partials[m.sel] != "" {
		return false
	}
	if len(m.queued[m.sel]) > 0 {
		return true
	}
	item, ok := m.selectedRow()
	return ok && item.state == session.StateBusy
}

func (m Model) spinning() bool {
	return m.thinkingSelected() || m.anyBusy()
}

func (m Model) anyBusy() bool {
	for _, item := range m.rows {
		if item.live && item.state == session.StateBusy {
			return true
		}
	}
	return false
}

func (m *Model) ensureAnimating() tea.Cmd {
	if m.animating || !m.spinning() {
		return nil
	}
	m.animating = true
	return spinTick()
}

func (m Model) handleSpin() (tea.Model, tea.Cmd) {
	if !m.spinning() {
		m.animating = false
		return m, nil
	}
	m.spinFrame++
	m.setContent()
	return m, spinTick()
}

func (m *Model) linesFor(name string) []render.Line {
	item, ok := m.selectedRow()
	if ok && item.name == name && !item.live {
		if cached, seen := m.replays[name]; seen {
			return cached
		}
		lines := m.mgr.Replay(name)
		m.replays[name] = lines
		m.todos[name] = m.mgr.Todos(name)
		return lines
	}
	return m.mgr.Lines(name)
}

func (m Model) wrap(lines []render.Line) string {
	width := m.outputWidth()
	wrapped := make([]string, 0, len(lines))
	for _, line := range lines {
		if line.Text == "" {
			wrapped = append(wrapped, "")
			continue
		}
		if line.Class == render.ClassText && !m.showRaw {
			wrapped = append(wrapped, m.md.Render(line.Text, width))
			continue
		}
		text := line.Text
		if line.Full != "" {
			text += "  ⏎"
		}
		if line.Class == render.ClassPrompt && !m.showRaw {
			styled := inlineEmphasis(text, classStyle(render.ClassPrompt))
			wrapped = append(wrapped, lipgloss.NewStyle().Width(width).Render(styled))
			continue
		}
		wrapped = append(wrapped, classStyle(line.Class).Width(width).Render(text))
	}
	return strings.Join(wrapped, "\n")
}

func (m Model) View() string {
	if !m.ready {
		return "starting…"
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top, m.sidebarView(), withEdge(m.paneView(), m.focus == focusOutput))
	if dialog, ok := m.bodyDialogView(); ok {
		body = dialog
	}
	prompt := withEdge(m.promptView(), m.focus == focusPrompt)
	return lipgloss.JoinVertical(lipgloss.Left, body, prompt, m.statusView())
}

// A session dialog draws in the pane, not over the whole body; see docs/tui.md.
func (m Model) paneView() string {
	if dialog, ok := m.sessionDialogView(); ok {
		return lipgloss.JoinVertical(lipgloss.Left, m.barView(), dialog)
	}
	pane := lipgloss.JoinVertical(lipgloss.Left, m.barView(), m.outputView())
	if m.showSidePanel() {
		pane = lipgloss.JoinHorizontal(lipgloss.Top, pane, m.sidePanelView())
	}
	return pane
}

func (m Model) sessionDialogView() (string, bool) {
	width, height := m.baseOutputWidth(), m.outputHeight()
	switch {
	case m.confirm != "":
		return centre(width, height, m.confirmView(width)), true
	case m.jobsModal != nil:
		return centre(width, height, m.jobsModal.View(width, height)), true
	case m.pager != nil:
		return centre(width, height, m.pager.View(width, height)), true
	case m.rename != nil:
		return centre(width, height, m.rename.View(width)), true
	case m.choice != nil:
		return centre(width, height, m.choice.View(width)), true
	}
	return "", false
}

func (m Model) bodyDialogView() (string, bool) {
	width, height := m.width, m.bodyHeight()
	switch {
	case m.help != nil:
		return centre(width, height, m.help.View(width, height)), true
	case m.fields != nil:
		return centre(width, height, m.fields.View(width)), true
	case m.picker != nil:
		return centre(width, height, m.picker.View(width)), true
	case m.form != nil:
		return centre(width, height, m.form.View(width)), true
	}
	return "", false
}

func withEdge(block string, on bool) string {
	ch := " "
	if on {
		ch = focusEdgeStyle.Render(edgeMark)
	}
	height := lipgloss.Height(block)
	edge := make([]string, height)
	for i := range edge {
		edge[i] = ch
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, strings.Join(edge, "\n"), block)
}

func (m Model) sidebarView() string {
	rows := make([]string, 0, m.bodyHeight())

	visible := m.visibleLines()
	for i := m.listOffset; i < len(m.lines) && len(rows) < visible; i++ {
		line := m.lines[i]
		if line.header() {
			rows = append(rows, m.groupHeader(m.groups[line.group]))
			continue
		}
		rows = append(rows, m.sessionRow(m.rows[line.row]))
	}
	for len(rows) < m.bodyHeight() {
		rows = append(rows, strings.Repeat(" ", sidebarInner))
	}
	block := sidebarStyle.Width(sidebarInner).Height(m.bodyHeight()).Render(strings.Join(rows, "\n"))
	return withEdge(block, m.focus == focusSidebar)
}

func (m Model) sessionRow(item row) string {
	width := sidebarInner
	badge := ""
	if item.control && !headsGroup(item) {
		badge = " " + controlMark
	}
	if item.jobs > 0 {
		badge += fmt.Sprintf(" ⚙%d", item.jobs)
	}
	if item.queued > 0 {
		badge += fmt.Sprintf(" ⇢%d", item.queued)
	}
	nameWidth := width - 3 - lipgloss.Width(badge)
	if nameWidth < 1 {
		nameWidth = 1
	}
	rest := " " + pad(item.displayName(), nameWidth) + badge
	glyph := rowGlyph(item, m.spinFrame)
	if item.name == m.sel {
		return selectedRowStyle.Render(" ") +
			item.style().Background(lipgloss.Color("62")).Render(glyph) +
			selectedRowStyle.Width(width-2).Render(rest)
	}
	nameStyle := rowStyle
	if item.archived {
		nameStyle = rowMutedStyle
	}
	return " " + item.style().Render(glyph) + nameStyle.Width(width-2).Render(rest)
}

// groupHeader names one directory. A folded header also carries the glyph of the
// most urgent row it hides.
func (m Model) groupHeader(item group) string {
	mark, glyph := foldOpenMark, ""
	if item.folded {
		lead := row{live: item.live, archived: item.archived, state: item.state}
		mark = foldShutMark
		glyph = lead.style().Render(rowGlyph(lead, m.spinFrame)) + " "
	}
	label := item.label
	if item.creator {
		label = controlMark + " " + label
	}
	count := strconv.Itoa(item.count)
	width := sidebarInner - 3 - lipgloss.Width(glyph) - len(count)
	if width < 1 {
		width = 1
	}
	return groupMarkStyle.Render(mark) + " " +
		groupLabelStyle.Render(pad(label, width)) + " " +
		glyph + groupCountStyle.Render(count)
}

func (m Model) outputView() string {
	if q := m.questions[m.sel]; q != nil {
		return centre(m.outputWidth(), m.outputHeight(), q.View(m.outputWidth()))
	}
	if len(m.rows) == 0 {
		text := "No sessions yet.\n\nPress n to start one.\nPress ctrl+c to quit."
		if len(m.stored) > 0 {
			text = "Every stored session is archived.\n\nPress A to show them.\nPress n to start a new one."
		}
		return emptyStyle.Width(m.outputWidth()).Height(m.outputHeight()).Render(text)
	}
	return m.output.View()
}

func (m Model) sidePanelView() string {
	var rows []string
	if jobs := orderJobs(m.selectedJobs()); len(jobs) > 0 {
		running := 0
		for _, job := range jobs {
			if job.Status.Running() {
				running++
			}
		}
		rows = append(rows, taskHeaderStyle.Render(fmt.Sprintf("Jobs · %d/%d", running, len(jobs))), "")
		for _, job := range jobs {
			rows = append(rows, m.panelJobRow(job))
		}
	}
	if todos := m.todos[m.sel]; len(todos) > 0 {
		if len(rows) > 0 {
			rows = append(rows, "")
		}
		done := 0
		for _, todo := range todos {
			if todo.Status == protocol.TodoCompleted {
				done++
			}
		}
		rows = append(rows, taskHeaderStyle.Render(fmt.Sprintf("Tasks · %d/%d", done, len(todos))), "")
		item, ok := m.selectedRow()
		busy := ok && item.live && item.state == session.StateBusy
		for _, todo := range todos {
			rows = append(rows, m.taskRow(todo, busy))
		}
	}
	block := strings.Join(rows, "\n")
	return taskPanelStyle.Width(taskPanelWidth - 1).Height(m.bodyHeight()).Render(block)
}

func (m Model) panelJobRow(job session.Job) string {
	desc := job.Description
	if desc == "" {
		desc = job.ID
	}
	textStyle := rowStyle
	if !job.Status.Running() {
		textStyle = rowMutedStyle
	}
	return jobStyle(job.Status).Render(jobGlyph(job.Status)) + " " + textStyle.Render(truncate(desc, taskPanelInner-2))
}

func (m Model) taskRow(todo protocol.Todo, busy bool) string {
	glyph, glyphStyle, textStyle := "○", taskPendingStyle, rowStyle
	text := todo.Content
	switch todo.Status {
	case protocol.TodoCompleted:
		glyph, glyphStyle, textStyle = "✔", taskDoneStyle, rowMutedStyle
	case protocol.TodoInProgress:
		glyph, glyphStyle = "◐", taskActiveStyle
		if todo.ActiveForm != "" {
			text = todo.ActiveForm
		}
		if busy {
			glyph = spinnerFrame(m.spinFrame)
		}
	}
	return glyphStyle.Render(glyph) + " " + textStyle.Render(truncate(text, taskPanelInner-2))
}

func (m Model) barView() string {
	width := m.outputWidth()
	item, ok := m.selectedRow()
	if !ok {
		return barStyle.Width(width).Render(barMutedStyle.Render(" no session"))
	}

	lefts, rights := barDetails(item), m.barRights(item)
	for _, pair := range barLadder(len(lefts), len(rights)) {
		left := barLeft(item.displayName(), lefts[pair[0]])
		right := rights[pair[1]]
		if gap := width - lipgloss.Width(left) - lipgloss.Width(right); gap >= 0 {
			return barLine(width, left, right, gap)
		}
	}

	room := maxInt(3, width-1)
	left := barLeft(truncate(item.displayName(), room), nil)
	return barLine(width, left, "", maxInt(0, width-lipgloss.Width(left)))
}

func barLadder(lefts, rights int) [][2]int {
	pairs := [][2]int{{0, 0}}
	left, right, takeLeft := 0, 0, true
	for left < lefts-1 || right < rights-1 {
		switch {
		case takeLeft && left < lefts-1:
			left++
		case right < rights-1:
			right++
		default:
			left++
		}
		takeLeft = !takeLeft
		pairs = append(pairs, [2]int{left, right})
	}
	return pairs
}

func barLine(width int, left, right string, gap int) string {
	return barStyle.MaxHeight(1).Width(width).
		Render(left + barStyle.Render(strings.Repeat(" ", gap)) + right)
}

func barLeft(name string, details []string) string {
	left := barNameStyle.Render(" " + name)
	if len(details) > 0 {
		left += barMutedStyle.Render(" · " + strings.Join(details, " · "))
	}
	return left
}

type barSeg struct {
	text  string
	style lipgloss.Style
}

func (m Model) barRights(item row) []string {
	segs := m.rightSegs(item)
	out := make([]string, 0, len(segs)+1)
	for n := len(segs); n >= 0; n-- {
		out = append(out, renderRight(segs[:n]))
	}
	return out
}

func (m Model) rightSegs(item row) []barSeg {
	segs := []barSeg{{item.label, item.style().Background(barBackground)}}
	if item.live && item.context > 0 {
		segs = append(segs, barSeg{contextLabel(item), barMutedStyle})
	}
	if m.showRaw {
		segs = append(segs, barSeg{"raw", barMutedStyle})
	}
	if scroll := m.scrollIndicator(); scroll != "" {
		segs = append(segs, barSeg{scroll, barMutedStyle})
	}
	if item.jobs > 0 {
		segs = append(segs, barSeg{fmt.Sprintf("⚙%d", item.jobs), barMutedStyle})
	}
	if item.queued > 0 {
		segs = append(segs, barSeg{fmt.Sprintf("⇢%d", item.queued), barMutedStyle})
	}
	if item.input+item.output > 0 {
		segs = append(segs, barSeg{
			fmt.Sprintf("%s in %s out", formatCount(item.input), formatCount(item.output)),
			barMutedStyle,
		})
	}
	return append(segs, barSeg{fmt.Sprintf("$%.4f", item.cost), barCostStyle})
}

func contextLabel(item row) string {
	if limit := contextWindow(item.model); limit > 0 {
		pct := item.context * 100 / limit
		return fmt.Sprintf("ctx %s/%s (%d%%)", formatCount(item.context), formatCount(limit), pct)
	}
	return fmt.Sprintf("ctx %s", formatCount(item.context))
}

func renderRight(segs []barSeg) string {
	if len(segs) == 0 {
		return ""
	}
	var b strings.Builder
	for i, seg := range segs {
		if i > 0 {
			b.WriteString(barMutedStyle.Render(" · "))
		}
		b.WriteString(seg.style.Render(seg.text))
	}
	b.WriteString(barStyle.Render(" "))
	return b.String()
}

func (m Model) scrollIndicator() string {
	if m.output.AtBottom() {
		return ""
	}
	return fmt.Sprintf("↑ %d%%", int(m.output.ScrollPercent()*100))
}

func barDetails(item row) [][]string {
	var full []string
	if item.control {
		full = append(full, "control")
	}
	if item.model != "" {
		full = append(full, item.model)
	}
	full = append(full, item.mode)
	if item.effort != "" {
		full = append(full, item.effort+" effort")
	}
	out := make([][]string, 0, len(full)+1)
	for n := len(full); n >= 1; n-- {
		out = append(out, full[:n])
	}
	return append(out, nil)
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

func formatCount(n int) string {
	return fmt.Sprintf("%.1fk", float64(n)/1000)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m Model) promptView() string {
	label := "prompt"
	if m.sel != "" {
		label = m.sel
	}
	if hint, ok := m.mentionHint(); ok {
		return hint + "\n" + m.prompt.View()
	}
	if names := completionNames(m.templates, m.prompt.Value()); len(names) > 0 && m.focus == focusPrompt {
		return hintStyle.Render(truncate(strings.Join(names, "  ")+"   tab completes", m.width-2)) +
			"\n" + m.prompt.View()
	}
	if item, ok := m.selectedRow(); ok && !item.running() {
		return hintStyle.Render(label+" — not running, press Enter to resume") + "\n" + m.prompt.View()
	}
	if item, ok := m.selectedRow(); ok && item.state == session.StateBusy && m.focus == focusPrompt {
		hint := label + " — esc stops"
		if len(m.queued[m.sel]) > 0 {
			hint += " · enter sends queued"
		}
		return hintStyle.Render(hint) + "\n" + m.prompt.View()
	}
	if m.focus == focusPrompt {
		return promptLabelStyle.Render(label+" ⌁ ") + "\n" + m.prompt.View()
	}
	return hintStyle.Render(label+" — press Enter or Tab to type") + "\n" + m.prompt.View()
}

func (m Model) confirmView(width int) string {
	return modalStyle.Width(modalInner(width)).Render(fmt.Sprintf("Stop session %q?\n\n%s",
		m.confirm, hintStyle.Render("y stop · any other key cancel")))
}

func (m Model) statusView() string {
	if m.errText != "" {
		return statusStyle.Width(m.width).Render(errorStyle.Render(truncate(m.errText, m.width-2)))
	}
	var live, busy int
	for _, item := range m.rows {
		if !item.live {
			continue
		}
		live++
		if item.state == session.StateBusy {
			busy++
		}
	}

	left := []barSeg{{plural(live, "session"), statusMutedStyle}}
	if busy > 0 {
		left = append(left, barSeg{fmt.Sprintf("%d busy", busy), statusMutedStyle})
	}
	left = append(left, barSeg{fmt.Sprintf("$%.4f", m.mgr.TotalCost()), statusCostStyle})
	if m.status != "" {
		left = append(left, barSeg{m.status, statusMutedStyle})
	}
	right := statusMutedStyle.Render(statusHints())

	return statusStyle.Width(m.width).Render(statusLine(m.width-2, left, right))
}

func statusSegs(segs []barSeg) string {
	parts := make([]string, len(segs))
	for i, seg := range segs {
		parts[i] = seg.style.Render(seg.text)
	}
	return strings.Join(parts, statusMutedStyle.Render(" · "))
}

func statusLine(width int, left []barSeg, right string) string {
	full := statusSegs(left)
	if gap := width - lipgloss.Width(full) - lipgloss.Width(right); gap >= 0 {
		return statusFill(full, right, gap)
	}
	for n := len(left); n >= 1; n-- {
		side := statusSegs(left[:n])
		if gap := width - lipgloss.Width(side); gap >= 0 {
			return statusFill(side, "", gap)
		}
	}
	return statusMutedStyle.Render(truncate(left[0].text, width))
}

func statusFill(left, right string, gap int) string {
	return left + statusMutedStyle.Render(strings.Repeat(" ", gap)) + right
}

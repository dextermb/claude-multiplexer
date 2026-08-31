package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dextermb/claude-multiplexer/internal/manager"
	"github.com/dextermb/claude-multiplexer/internal/markdown"
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
type shutdownDoneMsg struct{}
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
	stored       []manager.Meta
	storedLoaded bool
	greeted      bool
	showArchived bool
	replays      map[string][]render.Line
	partials     map[string]string
	md           *markdown.Renderer
	showRaw      bool
	templates    []template.Template
	help         *help
	picker       *picker
	fields       *fieldForm
	pending      string
	sel          string
	listOffset   int

	output     viewport.Model
	outputText string
	prompt     textarea.Model
	form       *form
	confirm    string
	focus      focusArea

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
	prompt.SetHeight(promptHeight - 1)

	return Model{
		replays:  make(map[string][]render.Line),
		partials: make(map[string]string),
		md:       markdown.New(),
		opts:     opts,
		mgr:      opts.Manager,
		sub:      opts.Manager.Subscribe(manager.DefaultSubscriberBuffer),
		output:   viewport.New(0, 0),
		prompt:   prompt,
		focus:    focusSidebar,
		mouseOn:  true,
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
		cmds = append(cmds, spawnCmd(m.mgr, manager.Spec{Dir: m.opts.InitialDir}))
	}
	return tea.Batch(cmds...)
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

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	case shutdownDoneMsg:
		return m, tea.Quit
	case tea.MouseMsg:
		return m.handleMouse(msg)
	case tea.KeyMsg:
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
	m.prompt.SetWidth(width)
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
	m.setPartial(ev.Session, ev.Partial)
	m.refresh()

	switch {
	case gap:
		m.rebuildOutput()
	case ev.Session == m.sel && len(ev.Lines) > 0:
		m.appendOutput(ev.Lines)
	case ev.Session == m.sel:
		m.setContent()
	}
	if ev.Closed {
		return m, tea.Batch(waitEvent(m.sub), reloadStored(m.mgr))
	}
	return m, waitEvent(m.sub)
}

func (m Model) handleSpawned(msg spawnedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.errText = msg.err.Error()
		return m, nil
	}
	m.errText = ""
	m.status = "started " + msg.name
	delete(m.replays, msg.name)
	m.refresh()
	m.sel = msg.name
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
	var cmd tea.Cmd
	m.prompt, cmd = m.prompt.Update(msg)
	return m, cmd
}

func (m Model) outputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.focus = focusSidebar
		return m, nil
	case "enter", "i":
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
	case "m":
		return m.toggleMarkdown()
	case "?":
		m.help = newHelp()
		return m, textinput.Blink
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
	if m.form != nil || m.confirm != "" || m.quitting {
		return m, nil
	}
	if msg.Action != tea.MouseActionPress {
		return m, nil
	}

	switch msg.Button {
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
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
		if msg.Y >= m.bodyHeight() && msg.Y < m.bodyHeight()+promptHeight {
			m.focus = focusPrompt
			m.prompt.Focus()
			return m, textarea.Blink
		}
		if msg.X >= sidebarWidth {
			m.focus = focusOutput
			m.prompt.Blur()
			return m, nil
		}
		if msg.X < sidebarWidth {
			index := m.listOffset + msg.Y - titleHeight
			if msg.Y >= titleHeight && index >= 0 && index < len(m.rows) {
				m.sel = m.rows[index].name
				m.focus = focusSidebar
				m.prompt.Blur()
				m.rebuildOutput()
			}
			return m, nil
		}
	}
	return m, nil
}

func (m Model) move(delta int) (tea.Model, tea.Cmd) {
	if len(m.rows) == 0 {
		return m, nil
	}
	index := m.selIndex() + delta
	if index < 0 {
		index = 0
	}
	if index >= len(m.rows) {
		index = len(m.rows) - 1
	}
	if m.rows[index].name == m.sel {
		return m, nil
	}
	m.sel = m.rows[index].name
	m.rebuildOutput()
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
	if text == "" {
		return m, nil
	}
	item, ok := m.selectedRow()
	if !ok {
		m.errText = "no session is selected"
		return m, nil
	}
	if !item.running() {
		m.errText = "this session is not running — press Enter to resume it"
		return m, nil
	}
	m.prompt.Reset()
	return m.dispatch(text)
}

func (m Model) dispatch(text string) (tea.Model, tea.Cmd) {
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
	m.refresh()
	return m, nil
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
	names := completionNames(m.templates, m.prompt.Value())
	if len(names) == 0 {
		return m.toggleFocus()
	}
	m.prompt.SetValue(names[0] + " ")
	m.prompt.CursorEnd()
	return m, nil
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
	for _, snap := range m.mgr.Snapshots() {
		rows = append(rows, rowFromSnapshot(snap))
	}
	for _, meta := range m.stored {
		if meta.Archived && !m.showArchived {
			continue
		}
		rows = append(rows, rowFromMeta(meta))
	}
	m.rows = rows

	if m.selIndex() < 0 && len(m.rows) > 0 {
		m.sel = m.rows[0].name
	}
	if len(m.rows) == 0 {
		m.sel = ""
	}
	m.clampOffset()
}

func (m *Model) clampOffset() {
	rows := m.visibleRows()
	index := m.selIndex()
	if index < 0 || rows <= 0 {
		m.listOffset = 0
		return
	}
	if index < m.listOffset {
		m.listOffset = index
	}
	if index >= m.listOffset+rows {
		m.listOffset = index - rows + 1
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
	height := m.height - promptHeight - statusHeight
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

func (m Model) visibleRows() int {
	rows := m.bodyHeight() - titleHeight
	if rows < 1 {
		return 1
	}
	return rows
}

func (m Model) outputWidth() int {
	width := m.width - sidebarWidth
	if width < 10 {
		return 10
	}
	return width
}

func (m *Model) rebuildOutput() {
	if m.sel == "" {
		m.outputText = ""
		m.output.SetContent("")
		return
	}
	m.outputText = m.wrap(m.linesFor(m.sel))
	m.setContent()
	m.output.GotoBottom()
}

func (m *Model) appendOutput(lines []render.Line) {
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
	m.output.SetContent(m.outputText + m.partialView())
	if atBottom {
		m.output.GotoBottom()
	}
}

func (m Model) partialView() string {
	partial := m.partials[m.sel]
	if partial == "" {
		return ""
	}
	chunk := classStyle(render.ClassText).Width(m.outputWidth()).Render(partial + cursorMark)
	if m.outputText == "" {
		return chunk
	}
	return "\n" + chunk
}

func (m *Model) linesFor(name string) []render.Line {
	item, ok := m.selectedRow()
	if ok && item.name == name && !item.live {
		if cached, seen := m.replays[name]; seen {
			return cached
		}
		lines := m.mgr.Replay(name)
		m.replays[name] = lines
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
		wrapped = append(wrapped, classStyle(line.Class).Width(width).Render(line.Text))
	}
	return strings.Join(wrapped, "\n")
}

func (m Model) View() string {
	if !m.ready {
		return "starting…"
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top, m.sidebarView(),
		lipgloss.JoinVertical(lipgloss.Left, m.barView(), m.outputView()))
	if m.form != nil {
		body = centre(m.width, m.bodyHeight(), m.form.View(m.width))
	}
	if m.picker != nil {
		body = centre(m.width, m.bodyHeight(), m.picker.View(m.width))
	}
	if m.fields != nil {
		body = centre(m.width, m.bodyHeight(), m.fields.View(m.width))
	}
	if m.help != nil {
		body = centre(m.width, m.bodyHeight(), m.help.View(m.width, m.bodyHeight()))
	}
	if m.confirm != "" {
		body = centre(m.width, m.bodyHeight(), m.confirmView())
	}
	return lipgloss.JoinVertical(lipgloss.Left, body, m.promptView(), m.statusView())
}

func (m Model) sidebarView() string {
	rows := make([]string, 0, m.bodyHeight())
	rows = append(rows, titleStyle.Width(sidebarWidth-1).Render("SESSIONS"))

	visible := m.visibleRows()
	for i := m.listOffset; i < len(m.rows) && len(rows) <= visible; i++ {
		rows = append(rows, m.sessionRow(m.rows[i]))
	}
	for len(rows) < m.bodyHeight() {
		rows = append(rows, strings.Repeat(" ", sidebarWidth-1))
	}
	return sidebarStyle.Width(sidebarWidth - 1).Height(m.bodyHeight()).Render(strings.Join(rows, "\n"))
}

func (m Model) sessionRow(item row) string {
	badge := item.label
	if item.queued > 0 {
		badge = fmt.Sprintf("%s+%d", badge, item.queued)
	}
	width := sidebarWidth - 1
	nameWidth := width - len(badge) - 3
	if nameWidth < 3 {
		nameWidth = 3
	}
	marker := "  "
	if item.name == m.sel {
		marker = "▸ "
	}
	text := marker + pad(item.name, nameWidth) + " "
	if item.name == m.sel {
		return selectedRowStyle.Width(width).Render(text + badge)
	}
	return rowStyle.Width(width).Render(text + item.style().Render(badge))
}

func (m Model) outputView() string {
	if len(m.rows) == 0 {
		text := "No sessions yet.\n\nPress n to start one.\nPress ctrl+c to quit."
		if len(m.stored) > 0 {
			text = "Every stored session is archived.\n\nPress A to show them.\nPress n to start a new one."
		}
		return emptyStyle.Width(m.outputWidth()).Height(m.outputHeight()).Render(text)
	}
	return m.output.View()
}

func (m Model) barView() string {
	width := m.outputWidth()
	item, ok := m.selectedRow()
	if !ok {
		return barStyle.Width(width).Render(barMutedStyle.Render(" no session"))
	}

	lefts, rights := barDetails(item), m.barRights(item)
	for _, pair := range barLadder(len(lefts), len(rights)) {
		left := barLeft(item.name, lefts[pair[0]])
		right := rights[pair[1]]
		if gap := width - lipgloss.Width(left) - lipgloss.Width(right); gap >= 0 {
			return barLine(width, left, right, gap)
		}
	}

	right := barRight(item, nil, false)
	room := maxInt(3, width-lipgloss.Width(right)-1)
	left := barLeft(truncate(item.name, room), nil)
	return barLine(width, left, right, maxInt(0, width-lipgloss.Width(left)-lipgloss.Width(right)))
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

func (m Model) barRights(item row) []string {
	stats := m.barStats(item)
	out := make([]string, 0, len(stats)+1)
	for _, group := range stats {
		out = append(out, barRight(item, group, true))
	}
	return append(out, barRight(item, nil, false))
}

func barRight(item row, stats []string, withState bool) string {
	var right string
	if withState {
		right = item.style().Background(barBackground).Render(item.label)
	}
	if len(stats) > 0 {
		right += barMutedStyle.Render(" · " + strings.Join(stats, " · "))
	}
	return right + barCostStyle.Render(fmt.Sprintf(" · $%.4f ", item.cost))
}

func (m Model) barStats(item row) [][]string {
	var stats []string
	if m.showRaw {
		stats = append(stats, "raw")
	}
	if scroll := m.scrollIndicator(); scroll != "" {
		stats = append(stats, scroll)
	}
	if item.queued > 0 {
		stats = append(stats, fmt.Sprintf("+%d queued", item.queued))
	}
	if item.turns > 0 {
		stats = append(stats, plural(item.turns, "turn"))
	}
	if item.last > 0 {
		stats = append(stats, formatDuration(item.last))
	}
	if item.input+item.output > 0 {
		stats = append(stats, fmt.Sprintf("%s in %s out",
			formatCount(item.input), formatCount(item.output)))
	}

	options := make([][]string, 0, len(stats)+1)
	for size := len(stats); size >= 0; size-- {
		options = append(options, stats[:size])
	}
	return options
}

func (m Model) scrollIndicator() string {
	if m.output.AtBottom() {
		return ""
	}
	return fmt.Sprintf("↑ %d%%", int(m.output.ScrollPercent()*100))
}

func barDetails(item row) [][]string {
	if item.model == "" {
		return [][]string{{item.mode}, nil}
	}
	return [][]string{{item.model, item.mode}, {item.model}, nil}
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

func formatCount(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

func formatDuration(d time.Duration) string {
	if d >= time.Second {
		return d.Round(100 * time.Millisecond).String()
	}
	return d.Round(time.Millisecond).String()
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
	if names := completionNames(m.templates, m.prompt.Value()); len(names) > 0 && m.focus == focusPrompt {
		return hintStyle.Render(truncate(strings.Join(names, "  ")+"   tab completes", m.width-2)) +
			"\n" + m.prompt.View()
	}
	if item, ok := m.selectedRow(); ok && !item.running() {
		return hintStyle.Render(label+" — not running, press Enter to resume") + "\n" + m.prompt.View()
	}
	if m.focus == focusPrompt {
		return promptLabelStyle.Render(label+" ⌁ ") + "\n" + m.prompt.View()
	}
	return hintStyle.Render(label+" — press Enter or Tab to type") + "\n" + m.prompt.View()
}

func (m Model) confirmView() string {
	return modalStyle.Render(fmt.Sprintf("Stop session %q?\n\n%s",
		m.confirm, hintStyle.Render("y stop · any other key cancel")))
}

func (m Model) statusView() string {
	if m.errText != "" {
		return statusStyle.Width(m.width).Render(errorStyle.Render(truncate(m.errText, m.width-2)))
	}
	var live, busy, stored int
	for _, item := range m.rows {
		switch {
		case !item.live:
			stored++
		case item.state == session.StateBusy:
			live++
			busy++
		default:
			live++
		}
	}
	parts := []string{
		plural(live, "session"),
		fmt.Sprintf("%d busy", busy),
		fmt.Sprintf("$%.4f", m.mgr.TotalCost()),
	}
	if stored > 0 {
		parts = append(parts, fmt.Sprintf("%d stored", stored))
	}
	if drops := m.sub.Dropped(); drops > 0 {
		parts = append(parts, fmt.Sprintf("%d drops", drops))
	}
	if m.status != "" {
		parts = append(parts, m.status)
	}
	parts = append(parts, statusHints())
	return statusStyle.Width(m.width).Render(truncate(strings.Join(parts, " · "), m.width-2))
}

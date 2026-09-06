package tui

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dextermb/claude-multiplexer/internal/config"
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
	focusDiff
)

type Options struct {
	Manager               *manager.Manager
	Config                config.Config
	ConfigPaths           []string
	ClaudePaths           []string
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

type settingsMsg struct {
	caps         map[string]int
	layouts      map[string]config.Layout
	activeLayout string
	defaults     newSessionDefaults
}

// newSessionDefaults holds the option each field of the new session form opens
// on, after the CLI flags, the settings file, and the built-ins resolve. See
// docs/config/new-session.md.
type newSessionDefaults struct {
	model   string
	mode    string
	effort  string
	control bool
}

// resolveSessionDefaults puts the CLI flag first, then the settings file, then
// the built-in. See docs/config/new-session.md.
func resolveSessionDefaults(opts Options, file config.Config) newSessionDefaults {
	return newSessionDefaults{
		model:   firstNonEmpty(opts.DefaultModel, file.DefaultModel),
		mode:    firstNonEmpty(opts.DefaultPermissionMode, file.DefaultPermissionMode, session.DefaultPermissionMode),
		effort:  file.DefaultEffort,
		control: file.DefaultControl != nil && *file.DefaultControl,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
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

type jobTickMsg struct{}

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
	history      []string
	histIdx      int
	histDraft    string
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
	layoutSwitch *layoutSwitch
	jobsModal    *jobsModal
	pending      string
	seq          *sequence
	seqGen       int
	sel          string
	listOffset   int

	output          viewport.Model
	outputText      string
	shownLines      []render.Line
	expanded        map[int]bool
	capped          []int
	markerAt        map[int]int
	blockStart      map[int]int
	hiddenRows      map[int]int
	blockCursor     int
	caps            map[string]int
	layouts         map[string]config.Layout
	activeLayout    string
	sessionDefaults newSessionDefaults
	layout          config.ResolvedLayout
	content         string
	selection       selRange
	prompt          textarea.Model
	pathMatches     []pathMatch
	pathPicked      int
	pathStem        string
	pathValue       string
	pathBase        string
	form            *form
	confirm         string
	focus           focusArea

	diffs           map[string]diffState
	fileDiffs       map[string]map[string]string
	diffOpen        map[string]map[string]bool
	diffFor         string
	diffPanel       bool
	diffSel         int
	diffScroll      int
	diffWidth       int
	diffHalf        bool
	diffLineNumbers bool
	diffTicking     bool
	sidebarHidden   bool

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

	prompt := textarea.New()
	prompt.Placeholder = "Type a prompt, then press Enter"
	prompt.Prompt = "> "
	prompt.ShowLineNumbers = false
	prompt.CharLimit = 0
	prompt.SetHeight(config.DefaultPromptMin)

	return Model{
		replays:         make(map[string][]render.Line),
		partials:        make(map[string]string),
		queued:          make(map[string][]string),
		todos:           make(map[string][]protocol.Todo),
		questions:       make(map[string]*questionDialog),
		diffs:           make(map[string]diffState),
		fileDiffs:       make(map[string]map[string]string),
		diffOpen:        make(map[string]map[string]bool),
		folded:          make(map[string]bool),
		roots:           make(map[string]string),
		expanded:        make(map[int]bool),
		markerAt:        make(map[int]int),
		blockStart:      make(map[int]int),
		hiddenRows:      make(map[int]int),
		md:              markdown.New(),
		opts:            opts,
		sessionDefaults: resolveSessionDefaults(opts, config.Config{}),
		mgr:             opts.Manager,
		sub:             opts.Manager.Subscribe(manager.DefaultSubscriberBuffer),
		output:          viewport.New(0, 0),
		prompt:          prompt,
		pathPicked:      -1,
		blockCursor:     -1,
		caps:            config.ResolveBlockCaps(config.Config{}),
		layout:          config.DefaultLayout(),
		focus:           focusSidebar,
		mouseOn:         true,
	}
}

func Run(opts Options) error {
	program := tea.NewProgram(New(opts), tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := program.Run()
	return err
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{waitEvent(m.sub), textarea.Blink, reloadStored(m.mgr), m.readSettings()}
	if m.opts.InitialDir != "" {
		cmds = append(cmds, spawnCmd(m.mgr, manager.Spec{Dir: m.opts.InitialDir, Control: m.opts.InitialControl}))
	}
	return tea.Batch(cmds...)
}

func jobTick() tea.Cmd {
	return tea.Tick(jobRefresh, func(time.Time) tea.Msg {
		return jobTickMsg{}
	})
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
	if model.sel != model.diffFor {
		model.diffFor = model.sel
		if refresh := model.diffRefreshCmd(); refresh != nil {
			cmd = tea.Batch(cmd, refresh)
		}
	}
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
	case settingsMsg:
		return m.handleSettings(msg)
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
	case sequenceTimeoutMsg:
		return m.handleSequenceTimeout(msg)
	case jobTickMsg:
		if m.jobsModal == nil {
			return m, nil
		}
		_, cmd := m.jobsModal.Update(msg)
		return m, cmd
	case bashResultMsg:
		return m.handleBash(msg)
	case openedMsg:
		return m.handleOpened(msg)
	case diffMsg:
		return m.handleDiff(msg)
	case fileDiffMsg:
		return m.handleFileDiff(msg)
	case diffTickMsg:
		return m.handleDiffTick()
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

// readSettings reads the settings file away from the main loop. The interface
// reads it again at each notice, so a tool that writes it takes effect at once.
// See docs/config.md.
func (m Model) readSettings() tea.Cmd {
	opts := m.opts
	return func() tea.Msg {
		file, err := config.Load(opts.ConfigPaths...)
		if err != nil {
			return settingsMsg{
				caps:     config.ResolveBlockCaps(config.Config{}),
				defaults: resolveSessionDefaults(opts, config.Config{}),
			}
		}
		merged := config.Resolve(opts.Config, file, config.LoadClaude(opts.ClaudePaths...))
		return settingsMsg{
			caps:         config.ResolveBlockCaps(merged),
			layouts:      merged.Layouts,
			activeLayout: merged.ActiveLayout,
			defaults:     resolveSessionDefaults(opts, merged),
		}
	}
}

func textinputBlink() tea.Cmd {
	return textarea.Blink
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

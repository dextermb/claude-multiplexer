package tui

import (
	"os"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
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
	focusTask
	focusReview
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

type Model struct {
	opts Options
	mgr  *manager.Manager
	sub  *manager.Subscription

	rows           []row
	groups         []group
	lines          []listLine
	peering        bool
	folded         map[string]bool
	roots          map[string]string
	stored         []manager.Meta
	storedLoaded   bool
	cost           float64
	costWindow     string
	greeted        bool
	showArchived   bool
	archivedWindow string
	search         textinput.Model
	searchOn       bool
	replays        map[string][]render.Line
	partials       map[string]string
	queued         map[string][]string
	history        []string
	histIdx        int
	histDraft      string
	todos          map[string][]protocol.Todo
	spinFrame      int
	animating      bool
	md             *markdown.Renderer
	mdMuted        *markdown.Renderer
	showRaw        bool
	showAge        bool
	ageTicking     bool
	templates      []template.Template
	help           *help
	modal          modal
	questions      map[string]*questionDialog
	pending        string
	seq            *sequence
	seqGen         int
	sel            string
	listOffset     int

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
	barSpecs        map[string]config.BarSpec
	barOutputs      map[string]string
	barRuns         map[string]time.Time
	barTicking      bool
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

	diffs           map[string]projectDiff
	fileDiffs       map[string]map[fileKey]string
	diffOpen        map[string]map[fileKey]bool
	diffPanel       bool
	diffSel         int
	diffScroll      int
	diffSize        int
	diffHalf        bool
	diffLineNumbers bool
	diffTicking     bool

	reviewMode        bool
	reviewPending     bool
	reviewFile        int
	reviewHunk        int
	reviewScroll      int
	reviewFocus       reviewSide
	reviewSidebar     bool
	reviewLineNumbers bool

	sidebarHidden bool
	taskScroll    int
	outputFor     string

	width      int
	height     int
	ready      bool
	lastSeq    uint64
	mouseOn    bool
	status     string
	errText    string
	armedQuit  bool
	quitting   bool
	burst      *burst
	burstAware bool
	inBurst    bool
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
		diffs:           make(map[string]projectDiff),
		fileDiffs:       make(map[string]map[fileKey]string),
		diffOpen:        make(map[string]map[fileKey]bool),
		folded:          make(map[string]bool),
		roots:           make(map[string]string),
		expanded:        make(map[int]bool),
		markerAt:        make(map[int]int),
		blockStart:      make(map[int]int),
		hiddenRows:      make(map[int]int),
		md:              markdown.New(),
		mdMuted:         markdown.NewMuted(),
		opts:            opts,
		sessionDefaults: resolveSessionDefaults(opts, config.Config{}),
		mgr:             opts.Manager,
		peering:         len(opts.Manager.PeerNames()) > 0,
		sub:             opts.Manager.Subscribe(manager.DefaultSubscriberBuffer),
		output:          viewport.New(0, 0),
		prompt:          prompt,
		search:          newSearchInput(),
		pathPicked:      -1,
		blockCursor:     -1,
		caps:            config.ResolveBlockCaps(config.Config{}),
		barOutputs:      make(map[string]string),
		barRuns:         make(map[string]time.Time),
		archivedWindow:  config.DefaultLastActive,
		layout:          config.DefaultLayout(),
		focus:           focusSidebar,
		mouseOn:         true,
		burst:           &burst{},
		burstAware:      burstAware,
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

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	prevSel := m.sel
	next, cmd := m.update(msg)
	model, ok := next.(Model)
	if !ok {
		return next, cmd
	}
	model.syncPromptHeight()
	model.syncMentions()
	if refresh := model.followSelection(prevSel); refresh != nil {
		cmd = tea.Batch(cmd, refresh)
	}
	model.clampTaskFocus()
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
	case unqueuedMsg:
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
		if m.modal == nil {
			return m, nil
		}
		return m.routeModal(msg)
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
	case ageTickMsg:
		return m.handleAgeTick()
	case barTickMsg:
		return m.handleBarTick(msg)
	case barOutputMsg:
		return m.handleBarOutput(msg)
	case tea.MouseMsg:
		return m.handleMouse(msg)
	case tea.KeyMsg:
		if isMouseArtifact(msg) {
			return m, nil
		}
		m.inBurst = m.burstAware && m.burst.key(time.Now())
		if msg.Paste {
			return m.handlePaste(string(msg.Runes))
		}
		return m.handleKey(msg)
	}

	var cmd tea.Cmd
	promptFocused := m.focus == focusPrompt || (m.reviewMode && m.reviewFocus == reviewPrompt)
	if promptFocused && m.form == nil {
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

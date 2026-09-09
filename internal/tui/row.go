package tui

import (
	"os"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/dextermb/claude-multiplexer/internal/manager"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

// A section is the top band a row sits in when peering is on: the sessions this
// host runs, the ones it runs for a peer, and the ones that run on a peer. See
// docs/peers.md.
type sectionKind int

const (
	sectionLocal sectionKind = iota
	sectionHosted
	sectionStreamed
)

type row struct {
	name        string
	title       string
	dir         string
	workDir     string
	projectDirs []string
	layout      string
	group       string
	section     sectionKind
	host        string
	hosted      bool
	owner       string
	parent      string
	model       string
	mode        string
	effort      string
	live        bool
	archived    bool
	control     bool
	scheduled   string
	state       session.State
	label       string
	queued      int
	turns       int
	cost        float64
	last        time.Duration
	input       int
	output      int
	context     int
	jobs        int
	jobList     []session.Job
	meta        manager.Meta
}

func rowFromSnapshot(snap session.Snapshot) row {
	return row{
		name:    snap.Name,
		title:   snap.Title,
		dir:     snap.Dir,
		model:   snap.Model,
		mode:    snap.PermissionMode,
		effort:  snap.Effort,
		live:    true,
		state:   snap.State,
		label:   snap.State.String(),
		queued:  snap.Queued,
		turns:   snap.Turns,
		cost:    snap.Cost,
		last:    snap.LastDuration,
		input:   snap.InputTokens,
		output:  snap.OutputTokens,
		context: snap.ContextTokens,
		jobs:    snap.RunningJobs(),
		jobList: snap.Jobs,
	}
}

func rowFromMeta(meta manager.Meta) row {
	label := "stored"
	if meta.Archived {
		label = "archived"
	}
	return row{
		name:        meta.Name,
		title:       meta.Title,
		dir:         meta.Dir,
		model:       meta.Model,
		mode:        meta.PermissionMode,
		effort:      meta.Effort,
		workDir:     meta.WorkingDir,
		projectDirs: meta.WorkingDirs,
		layout:      meta.Layout,
		archived:    meta.Archived,
		control:     meta.Control,
		hosted:      meta.Hosted,
		owner:       meta.Owner,
		scheduled:   meta.Scheduled,
		parent:      meta.Parent,
		label:       label,
		turns:       meta.Turns,
		cost:        meta.Cost,
		input:       meta.InputTokens,
		output:      meta.OutputTokens,
		meta:        meta,
	}
}

func (r row) style() lipgloss.Style {
	if !r.live {
		if r.archived {
			return lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	}
	return stateStyle(r.state)
}

func rowGlyph(item row, frame int) string {
	if !item.live {
		if item.archived {
			return "·"
		}
		return "○"
	}
	switch item.state {
	case session.StateStarting:
		return "◌"
	case session.StateBusy:
		return spinnerFrame(frame)
	case session.StateWaiting:
		return "?"
	default:
		return "●"
	}
}

func (r row) running() bool {
	return r.live && r.state.Live()
}

// remoteHost names the peer a remote session groups under: the host a streamed
// session runs on, or the client a hosted session runs for. It is empty for a
// local session. See docs/tui/sessions.md.
func (r row) remoteHost() string {
	if r.host != "" {
		return r.host
	}
	if r.hosted {
		return r.owner
	}
	return ""
}

// displayName is the title when the session has one, else the name.
func (r row) displayName() string {
	if r.title != "" {
		return r.title
	}
	return r.name
}

// openDir is the directory the interface opens for a session: the working
// directory a tool of that session set, and the directory it started in when
// there is none, or when the one it set is gone. See docs/mcp/tools.md.
func (r row) openDir() string {
	if r.workDir != "" {
		if info, err := os.Stat(r.workDir); err == nil && info.IsDir() {
			return r.workDir
		}
	}
	if len(r.projectDirs) > 0 {
		return r.projectDirs[0]
	}
	return r.dir
}

// diffDirs is the set of directories the diff panel groups the changes by: the
// project directories when the session has a project, else the one directory it
// opens. See docs/tui/diff.md.
func (r row) diffDirs() []string {
	if len(r.projectDirs) > 0 {
		return r.projectDirs
	}
	return []string{r.openDir()}
}

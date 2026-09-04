package tui

import (
	"os"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/dextermb/claude-multiplexer/internal/manager"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

type row struct {
	name     string
	title    string
	dir      string
	workDir  string
	group    string
	parent   string
	model    string
	mode     string
	effort   string
	live     bool
	archived bool
	control  bool
	state    session.State
	label    string
	queued   int
	turns    int
	cost     float64
	last     time.Duration
	input    int
	output   int
	context  int
	jobs     int
	jobList  []session.Job
	meta     manager.Meta
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
		name:     meta.Name,
		title:    meta.Title,
		dir:      meta.Dir,
		model:    meta.Model,
		mode:     meta.PermissionMode,
		effort:   meta.Effort,
		workDir:  meta.WorkingDir,
		archived: meta.Archived,
		control:  meta.Control,
		parent:   meta.Parent,
		label:    label,
		turns:    meta.Turns,
		cost:     meta.Cost,
		input:    meta.InputTokens,
		output:   meta.OutputTokens,
		meta:     meta,
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

// displayName is the title when the session has one, else the name.
func (r row) displayName() string {
	if r.title != "" {
		return r.title
	}
	return r.name
}

// openDir is the directory the interface opens for a session: the working
// directory a tool of that session set, and the directory it started in when
// there is none, or when the one it set is gone. See docs/mcp.md.
func (r row) openDir() string {
	if r.workDir != "" {
		if info, err := os.Stat(r.workDir); err == nil && info.IsDir() {
			return r.workDir
		}
	}
	return r.dir
}

package tui

import (
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/dextermb/claude-multiplexer/internal/manager"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

type row struct {
	name     string
	dir      string
	model    string
	mode     string
	live     bool
	archived bool
	state    session.State
	label    string
	queued   int
	turns    int
	cost     float64
	last     time.Duration
	input    int
	output   int
	meta     manager.Meta
}

func rowFromSnapshot(snap session.Snapshot) row {
	return row{
		name:   snap.Name,
		dir:    snap.Dir,
		model:  snap.Model,
		mode:   snap.PermissionMode,
		live:   true,
		state:  snap.State,
		label:  snap.State.String(),
		queued: snap.Queued,
		turns:  snap.Turns,
		cost:   snap.Cost,
		last:   snap.LastDuration,
		input:  snap.InputTokens,
		output: snap.OutputTokens,
	}
}

func rowFromMeta(meta manager.Meta) row {
	label := "stored"
	if meta.Archived {
		label = "archived"
	}
	return row{
		name:     meta.Name,
		dir:      meta.Dir,
		model:    meta.Model,
		mode:     meta.PermissionMode,
		archived: meta.Archived,
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
		return lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	}
	return stateStyle(r.state)
}

func (r row) running() bool {
	return r.live && r.state.Live()
}

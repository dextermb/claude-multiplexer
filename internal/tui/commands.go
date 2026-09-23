package tui

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dextermb/claude-multiplexer/internal/manager"
	"github.com/dextermb/claude-multiplexer/internal/peer"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

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
		cost, window := mgr.WindowCost()
		return storedMsg{metas: mgr.Stored(), cost: cost, window: window}
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

func resumeControlCmd(mgr *manager.Manager, name string, control bool) tea.Cmd {
	return func() tea.Msg {
		next, err := mgr.ResumeWithControl(context.Background(), name, control)
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

func archiveManyCmd(mgr *manager.Manager, names []string, skipped int) tea.Cmd {
	return func() tea.Msg {
		for _, name := range names {
			if err := mgr.Archive(name, true); err != nil {
				return archivedManyMsg{err: err}
			}
		}
		return archivedManyMsg{count: len(names), skipped: skipped}
	}
}

func spawnCmd(mgr *manager.Manager, spec manager.Spec) tea.Cmd {
	return func() tea.Msg {
		name, err := mgr.Spawn(context.Background(), spec)
		return spawnedMsg{name: name, err: err}
	}
}

// attachCmd starts a session on a peer and streams it in, so a remote session
// appears in the sidebar like a local one. See docs/peers.md.
func attachCmd(mgr *manager.Manager, host string, spec manager.Spec) tea.Cmd {
	return func() tea.Msg {
		name, err := mgr.AttachRemoteByName(host, peer.CreateSpec{
			Dir:            spec.Dir,
			TempDir:        strings.TrimSpace(spec.Dir) == "",
			Name:           spec.Name,
			Model:          spec.Model,
			PermissionMode: spec.PermissionMode,
			Effort:         spec.Effort,
		})
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

func unqueueCmd(mgr *manager.Manager, name string) tea.Cmd {
	return func() tea.Msg {
		_, err := mgr.Unqueue(name)
		return unqueuedMsg{name: name, err: err}
	}
}

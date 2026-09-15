package manager

import (
	"errors"
	"fmt"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

// ErrContextHeld is the failure when a prompt reaches a session the governor
// holds. The human clears the hold. See docs/sessions/context.md.
var ErrContextHeld = errors.New("manager: the session is held, because its context is full")

// The actions contextAction takes. Notify raises a notice and nothing else.
// Hold also refuses a new prompt until the human clears it.
const (
	ContextActionNotify = "notify"
	ContextActionHold   = "hold"
)

// contextGovernor is the settings of the governor, read once per crossing.
type contextGovernor struct {
	warn   int
	act    int
	action string
}

// readContextGovernor reads the governor settings. It reports false when the
// governor is off, which is the default. See docs/sessions/context.md.
func (m *Manager) readContextGovernor() (contextGovernor, bool) {
	cfg, err := config.Load(m.opts.ConfigPaths...)
	if err != nil {
		return contextGovernor{}, false
	}
	var g contextGovernor
	if cfg.ContextWarnPercent != nil {
		g.warn = *cfg.ContextWarnPercent
	}
	if cfg.ContextActPercent != nil {
		g.act = *cfg.ContextActPercent
	}
	g.action = cfg.ContextAction
	if g.action != ContextActionHold {
		g.action = ContextActionNotify
	}
	if g.warn < 1 && g.act < 1 {
		return contextGovernor{}, false
	}
	return g, true
}

// contextFill is the percent of the context window the session fills now. It
// reports false when the window of the model is not known.
func contextFill(snap session.Snapshot) (int, bool) {
	limit := session.ContextWindow(snap.Model)
	if limit < 1 || snap.ContextTokens < 1 {
		return 0, false
	}
	return snap.ContextTokens * 100 / limit, true
}

// maybeContextNotice raises the governor notices as the context of a session
// fills, and sets the hold at the act threshold. It raises each notice once per
// crossing, and it does no work while the fill is unchanged. See
// docs/sessions/context.md.
func (m *Manager) maybeContextNotice(item *entry, snap session.Snapshot) {
	item.ctxMu.Lock()
	if snap.ContextTokens == item.ctxLast {
		item.ctxMu.Unlock()
		return
	}
	item.ctxLast = snap.ContextTokens
	warned, acted := item.ctxWarned, item.ctxActed
	item.ctxMu.Unlock()

	if warned && acted {
		return
	}
	fill, ok := contextFill(snap)
	if !ok {
		return
	}
	g, on := m.readContextGovernor()
	if !on {
		return
	}

	if g.act > 0 && fill >= g.act && !acted {
		item.ctxMu.Lock()
		item.ctxActed = true
		item.ctxWarned = true
		if g.action == ContextActionHold {
			item.ctxHeld = true
		}
		item.ctxMu.Unlock()
		m.notify(snap.Name, contextNotice(snap.Name, fill, g.action), true)
		return
	}
	if g.warn > 0 && fill >= g.warn && !warned {
		item.ctxMu.Lock()
		item.ctxWarned = true
		item.ctxMu.Unlock()
		m.notify(snap.Name, fmt.Sprintf("%s context is %d%% full", snap.Name, fill), true)
	}
}

func contextNotice(name string, fill int, action string) string {
	if action == ContextActionHold {
		return fmt.Sprintf("%s is held: context is %d%% full", name, fill)
	}
	return fmt.Sprintf("%s context is %d%% full", name, fill)
}

// ContextHeld reports whether the governor holds a session.
func (m *Manager) ContextHeld(name string) bool {
	item, err := m.entry(name)
	if err != nil {
		return false
	}
	item.ctxMu.Lock()
	defer item.ctxMu.Unlock()
	return item.ctxHeld
}

// ClearContextHold lets a held session take a prompt again. The thresholds stay
// crossed, so the governor does not raise the same notice twice. See
// docs/sessions/context.md.
func (m *Manager) ClearContextHold(name string) error {
	item, err := m.entry(name)
	if err != nil {
		return err
	}
	item.ctxMu.Lock()
	held := item.ctxHeld
	item.ctxHeld = false
	item.ctxMu.Unlock()
	if held {
		m.notify(name, name+" is no longer held", true)
	}
	return nil
}

// HeldSessions names every live session the governor holds, so the sidebar
// marks them.
func (m *Manager) HeldSessions() map[string]bool {
	m.mu.Lock()
	names := make(map[string]*entry, len(m.entries))
	for name, item := range m.entries {
		names[name] = item
	}
	m.mu.Unlock()

	out := make(map[string]bool, len(names))
	for name, item := range names {
		item.ctxMu.Lock()
		if item.ctxHeld {
			out[name] = true
		}
		item.ctxMu.Unlock()
	}
	return out
}

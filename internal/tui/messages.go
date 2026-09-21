package tui

import (
	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/keys"
	"github.com/dextermb/claude-multiplexer/internal/manager"
)

type eventMsg manager.Event

type busClosedMsg struct{}

type spawnedMsg struct {
	name string
	err  error
}

type storedMsg struct {
	metas  []manager.Meta
	cost   float64
	window string
}

type settingsMsg struct {
	caps           map[string]int
	layouts        map[string]config.Layout
	activeLayout   string
	bars           map[string]config.BarSpec
	defaults       newSessionDefaults
	archivedWindow string
	keys           keys.Keymap
	keysOK         bool
	keyNote        string
}

type stoppedMsg struct {
	name string
	err  error
}

type interruptedMsg struct {
	name string
	err  error
}

type unqueuedMsg struct {
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

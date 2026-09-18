package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

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

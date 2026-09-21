package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dextermb/claude-multiplexer/internal/commands"
	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/keys"
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
			km, ok, note := resolveKeymap(nil)
			return settingsMsg{
				caps:           config.ResolveBlockCaps(config.Config{}),
				bars:           resolveBarSpecs(nil),
				defaults:       resolveSessionDefaults(opts, config.Config{}),
				archivedWindow: config.ArchivedWindow(""),
				keys:           km,
				keysOK:         ok,
				keyNote:        note,
			}
		}
		merged := config.Resolve(opts.Config, file, config.LoadClaude(opts.ClaudePaths...))
		km, ok, note := resolveKeymap(merged.Keybindings)
		cmds, cok, cnote := resolveCommands(merged.Commands, km)
		return settingsMsg{
			caps:           config.ResolveBlockCaps(merged),
			layouts:        merged.Layouts,
			activeLayout:   merged.ActiveLayout,
			bars:           resolveBarSpecs(merged.Bars),
			defaults:       resolveSessionDefaults(opts, merged),
			archivedWindow: config.ArchivedWindow(merged.ArchivedWindow),
			keys:           km,
			keysOK:         ok,
			keyNote:        note,
			commands:       cmds,
			commandsOK:     cok,
			commandNote:    cnote,
		}
	}
}

// resolveCommands builds the command table from the settings, validated against
// the resolved keymap. It reports whether the commands are valid, and a note for
// the status bar from any refusal. See docs/config/commands.md.
func resolveCommands(cmds []config.Command, km keys.Keymap) (commands.Resolved, bool, string) {
	resolved, errs := commands.Resolve(cmds, km)
	if len(errs) > 0 {
		return resolved, false, "commands: " + errs[0].Error()
	}
	return resolved, true, ""
}

// resolveKeymap builds the keymap from the user bindings. It reports whether the
// bindings are valid, and a note for the status bar from any error or warning.
// See docs/config/keybindings.md.
func resolveKeymap(kb *config.Keybindings) (keys.Keymap, bool, string) {
	km, warnings, errs := keys.LoadKeymap(kb)
	if len(errs) > 0 {
		return km, false, "keybindings: " + errs[0].Error()
	}
	if len(warnings) > 0 {
		return km, true, "keybindings: " + warnings[0].String()
	}
	return km, true, ""
}

// resolveBarSpecs resolves the composition of the two bars against the embedded
// defaults, so the interface holds a ready spec for each. See docs/config/bars.md.
func resolveBarSpecs(bars *config.Bars) map[string]config.BarSpec {
	out := make(map[string]config.BarSpec, 2)
	for _, bar := range []string{config.BarSession, config.BarStatus} {
		if spec, err := config.ResolveBarSpec(bars, bar); err == nil {
			out[bar] = spec
		}
	}
	return out
}

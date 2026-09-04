// Package open builds the command that opens a directory, in the file manager
// or in the editor. See docs/config.md.
package open

import (
	"errors"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/dextermb/claude-multiplexer/internal/config"
)

// ErrNoEditor says that no source named an editor.
var ErrNoEditor = errors.New("no editor: set --editor, $EDITOR, or the config file")

// Target is one program to start, and whether it wants the terminal.
type Target struct {
	Command  string
	Args     []string
	Terminal bool
}

// terminalEditors are the editors that draw in the terminal, by base name.
var terminalEditors = map[string]bool{
	"ed":    true,
	"emacs": true,
	"helix": true,
	"hx":    true,
	"joe":   true,
	"kak":   true,
	"micro": true,
	"nano":  true,
	"nvim":  true,
	"vi":    true,
	"vim":   true,
	"vis":   true,
}

// IsTerminalEditor reports whether a command line names a known terminal
// editor. It reads the base name, so a full path still matches.
func IsTerminalEditor(command string) bool {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return false
	}
	name := filepath.Base(fields[0])
	return terminalEditors[strings.TrimSuffix(name, ".exe")]
}

// Editor builds the command that opens dir in the editor.
func Editor(cfg config.Config, dir string) (Target, error) {
	fields := strings.Fields(cfg.Editor)
	if len(fields) == 0 {
		return Target{}, ErrNoEditor
	}
	terminal := IsTerminalEditor(cfg.Editor)
	if cfg.EditorTerminal != nil {
		terminal = *cfg.EditorTerminal
	}
	return Target{
		Command:  fields[0],
		Args:     append(fields[1:], dir),
		Terminal: terminal,
	}, nil
}

// FileManager builds the command that opens dir in the file manager of the
// platform.
func FileManager(dir string) Target {
	switch runtime.GOOS {
	case "darwin":
		return Target{Command: "open", Args: []string{dir}}
	case "windows":
		return Target{Command: "explorer", Args: []string{dir}}
	default:
		return Target{Command: "xdg-open", Args: []string{dir}}
	}
}

// Package config names the directories of the program, reads the settings file,
// and resolves a setting from the flag, the environment, that file, and the
// settings of Claude Code. See docs/config.md.
package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const FileName = "config.json"

// DefaultBlockCap is the rows a block draws in the session pane before the pane
// caps it. See docs/tui/output.md.
const DefaultBlockCap = 20

// Names holds the settings directories of the program, in the order they are
// read. The first one is where a new installation writes.
var Names = []string{"claude-multiplexer", "multiplexer", "multiplexier"}

// RootNames holds the state directories under the home directory, in the order
// they are read. The first one is where a new installation writes. Only these
// two ever held a session, so only these two are read. See docs/manager.md.
var RootNames = []string{".claude-multiplexer", ".multiplexier"}

// DefaultRoot gives the state directory to use when --root names none: the
// first of RootNames that is there, and the first name when none of them is.
func DefaultRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	for _, name := range RootNames {
		path := filepath.Join(home, name)
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path, nil
		}
	}
	return filepath.Join(home, RootNames[0]), nil
}

type Config struct {
	Editor         string `json:"editor,omitempty"`
	EditorTerminal *bool  `json:"editorTerminal,omitempty"`
	// BlockCap is the rows a block draws before the pane caps it. Zero caps
	// nothing, and nil takes DefaultBlockCap.
	BlockCap *int `json:"blockCap,omitempty"`
}

// BlockCapOrDefault reads the cap out of the settings, and gives the default
// when the settings name none.
func BlockCapOrDefault(cfg Config) int {
	if cfg.BlockCap == nil {
		return DefaultBlockCap
	}
	return *cfg.BlockCap
}

// Paths lists the settings files to look for, in order.
func Paths() []string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		dir = filepath.Join(home, ".config")
	}
	out := make([]string, 0, len(Names))
	for _, name := range Names {
		out = append(out, filepath.Join(dir, name, FileName))
	}
	return out
}

// Target names the file to write: the first one that is there, and the first
// path when none of them is.
func Target(paths ...string) string {
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

// Active names the settings file that is read now: the first one that is there,
// and an empty string when none of them is.
func Active(paths ...string) string {
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// Write puts the settings in path, and makes the directory when it is missing.
func Write(path string, cfg Config) error {
	if path == "" {
		return errors.New("config: no path to write")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

// Load reads the first file that is there. A file that is not there gives an
// empty configuration, and a file that does not parse gives an error.
func Load(paths ...string) (Config, error) {
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return Config{}, err
		}
		if strings.TrimSpace(string(data)) == "" {
			return Config{}, nil
		}
		var cfg Config
		if err := json.Unmarshal(data, &cfg); err != nil {
			return Config{}, errors.New(path + ": " + err.Error())
		}
		return cfg, nil
	}
	return Config{}, nil
}

// The fields that Clear takes.
const (
	FieldEditor   = "editor"
	FieldTerminal = "terminal"
	FieldBoth     = "both"
)

// Clear takes a field out of the settings. An empty field clears both.
func Clear(cfg Config, field string) (Config, error) {
	switch field {
	case FieldEditor:
		cfg.Editor = ""
	case FieldTerminal:
		cfg.EditorTerminal = nil
	case FieldBoth, "":
		cfg.Editor = ""
		cfg.EditorTerminal = nil
	default:
		return Config{}, errors.New("config: the field must be " + FieldEditor + ", " + FieldTerminal + ", or " + FieldBoth)
	}
	return cfg, nil
}

// Resolve puts the flags first, then the environment, then the settings file,
// then the settings of Claude Code.
func Resolve(flags, file, claude Config) Config {
	out := file
	if out.Editor == "" {
		out.Editor = claude.Editor
	}
	if out.EditorTerminal == nil {
		out.EditorTerminal = claude.EditorTerminal
	}
	if editor := firstOf(flags.Editor, os.Getenv("VISUAL"), os.Getenv("EDITOR")); editor != "" {
		out.Editor = editor
	}
	if flags.EditorTerminal != nil {
		out.EditorTerminal = flags.EditorTerminal
	}
	if flags.BlockCap != nil {
		out.BlockCap = flags.BlockCap
	}
	return out
}

func firstOf(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

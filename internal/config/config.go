// Package config reads the settings file, and resolves a setting from the
// flag, the environment, that file, and the settings of Claude Code. See
// docs/config.md.
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

// Names holds both spellings of the program, in the order they are read.
var Names = []string{"multiplexer", "multiplexier"}

type Config struct {
	Editor         string `json:"editor,omitempty"`
	EditorTerminal *bool  `json:"editorTerminal,omitempty"`
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

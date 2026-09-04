// Package config reads the settings file, and resolves a setting from the
// flag, the environment, and that file. See docs/config.md.
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
	Editor         string `json:"editor"`
	EditorTerminal *bool  `json:"editorTerminal"`
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

// Resolve puts the flags first, then the environment, then the file.
func Resolve(flags, file Config) Config {
	out := file
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

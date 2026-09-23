package update

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// State is the cached result of the last update check. It lives in the state
// directory, so a check runs at most once an hour and a dismissal lasts a day.
// See docs/version-updates.md.
type State struct {
	LastCheck      time.Time `json:"lastCheck"`
	Tag            string    `json:"tag,omitempty"`
	URL            string    `json:"url,omitempty"`
	PublishedAt    time.Time `json:"publishedAt,omitzero"`
	DismissedUntil time.Time `json:"dismissedUntil,omitzero"`
}

// StatePath names the state file inside the root state directory.
func StatePath(root string) string {
	return filepath.Join(root, "update-check.json")
}

// LoadState reads the state file. A missing file is not an error; it gives a
// zero State.
func LoadState(path string) (State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return State{}, nil
		}
		return State{}, err
	}
	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return State{}, err
	}
	return st, nil
}

// SaveState writes the state file, and makes the directory when it is missing.
func SaveState(path string, st State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

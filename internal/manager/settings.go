package manager

import (
	"encoding/json"
	"errors"

	"github.com/dextermb/claude-multiplexer/internal/config"
)

// SetEditor writes the editor settings, so a session can name the editor the
// human opens a directory with. See docs/config.md.
func (m *Manager) SetEditor(editor string, terminal *bool) (string, error) {
	path := config.Target(m.opts.ConfigPaths...)
	if path == "" {
		return "", errors.New("manager: no settings file to write")
	}
	current, err := config.Load(path)
	if err != nil {
		return "", err
	}
	if editor != "" {
		current.Editor = editor
	}
	if terminal != nil {
		current.EditorTerminal = terminal
	}
	if err := config.Write(path, current); err != nil {
		return "", err
	}
	return path, nil
}

// UnsetEditor takes the editor settings out of the settings file. It reports
// whether the file held anything to take out. See docs/config.md.
func (m *Manager) UnsetEditor(field string) (string, bool, error) {
	path := config.Target(m.opts.ConfigPaths...)
	if path == "" {
		return "", false, errors.New("manager: no settings file to write")
	}
	current, err := config.Load(path)
	if err != nil {
		return "", false, err
	}
	next, err := config.Clear(current, field)
	if err != nil {
		return "", false, err
	}
	if config.SameEditor(next, current) {
		return path, false, nil
	}
	if err := config.Write(path, next); err != nil {
		return "", false, err
	}
	return path, true, nil
}

// SetConfig sets one settings key by a dot-notation path, so a session can write
// any settings-file value. It validates the change against the schema before it
// writes. See docs/config.md.
func (m *Manager) SetConfig(path string, value json.RawMessage) (string, error) {
	file := config.Target(m.opts.ConfigPaths...)
	if file == "" {
		return "", errors.New("manager: no settings file to write")
	}
	current, err := config.Load(file)
	if err != nil {
		return "", err
	}
	next, err := config.SetPath(current, path, value)
	if err != nil {
		return "", err
	}
	if err := config.Write(file, next); err != nil {
		return "", err
	}
	return file, nil
}

// UnsetConfig removes one settings key by a dot-notation path. It reports whether
// the key was there to remove. See docs/config.md.
func (m *Manager) UnsetConfig(path string) (string, bool, error) {
	file := config.Target(m.opts.ConfigPaths...)
	if file == "" {
		return "", false, errors.New("manager: no settings file to write")
	}
	current, err := config.Load(file)
	if err != nil {
		return "", false, err
	}
	next, changed, err := config.UnsetPath(current, path)
	if err != nil {
		return "", false, err
	}
	if !changed {
		return file, false, nil
	}
	if err := config.Write(file, next); err != nil {
		return "", false, err
	}
	return file, true, nil
}

// SetBlockCap writes the rows a block draws before the pane caps it, so a
// session can change how much of a large result the human sees. An empty bucket
// sets the default for every type; a bucket sets that one type. A nil rows caps
// nothing (a bucket only). See docs/config.md.
func (m *Manager) SetBlockCap(bucket string, rows *int) (string, error) {
	if rows != nil && *rows < 0 {
		return "", errors.New("manager: the block cap must be zero or more")
	}
	if bucket == "" && rows == nil {
		return "", errors.New("manager: the default block cap needs a row count")
	}
	if bucket != "" && !config.ValidBucket(bucket) {
		return "", errors.New("manager: unknown block type " + bucket)
	}
	path := config.Target(m.opts.ConfigPaths...)
	if path == "" {
		return "", errors.New("manager: no settings file to write")
	}
	current, err := config.Load(path)
	if err != nil {
		return "", err
	}
	if bucket == "" {
		current.BlockCap = rows
	} else {
		if current.BlockCaps == nil {
			current.BlockCaps = map[string]*int{}
		}
		current.BlockCaps[bucket] = rows
	}
	if err := config.Write(path, current); err != nil {
		return "", err
	}
	return path, nil
}

// UnsetBlockCap takes a block cap out of the settings file. An empty bucket
// clears the default, so the pane returns to config.DefaultBlockCap; a bucket
// clears that type, so it takes the default again. It reports whether the file
// held the cap. See docs/config.md.
func (m *Manager) UnsetBlockCap(bucket string) (string, bool, error) {
	if bucket != "" && !config.ValidBucket(bucket) {
		return "", false, errors.New("manager: unknown block type " + bucket)
	}
	path := config.Target(m.opts.ConfigPaths...)
	if path == "" {
		return "", false, errors.New("manager: no settings file to write")
	}
	current, err := config.Load(path)
	if err != nil {
		return "", false, err
	}
	if bucket == "" {
		if current.BlockCap == nil {
			return path, false, nil
		}
		current.BlockCap = nil
	} else {
		if _, ok := current.BlockCaps[bucket]; !ok {
			return path, false, nil
		}
		delete(current.BlockCaps, bucket)
		if len(current.BlockCaps) == 0 {
			current.BlockCaps = nil
		}
	}
	if err := config.Write(path, current); err != nil {
		return "", false, err
	}
	return path, true, nil
}

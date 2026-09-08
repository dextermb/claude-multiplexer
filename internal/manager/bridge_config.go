package manager

import (
	"encoding/json"
	"strconv"

	"github.com/dextermb/claude-multiplexer/internal/config"
)

func (b *bridge) SetWorkingDir(path, by string) (string, error) {
	full, err := b.m.SetWorkingDir(by, path)
	if err != nil {
		return "", err
	}
	b.m.notify(by, by+" set its working directory to "+full, true)
	return full, nil
}

func (b *bridge) UnsetWorkingDir(by string) (bool, error) {
	changed, err := b.m.UnsetWorkingDir(by)
	if err != nil {
		return false, err
	}
	if changed {
		b.m.notify(by, by+" cleared its working directory", true)
	}
	return changed, nil
}

func (b *bridge) Project(session string) ([]string, error) { return b.m.Project(session) }

func (b *bridge) SetProject(paths []string, by string) ([]string, error) {
	dirs, err := b.m.SetProject(by, paths)
	if err != nil {
		return nil, err
	}
	b.m.notify(by, by+" set its project to "+projectSummary(dirs), true)
	return dirs, nil
}

func (b *bridge) AddProjectDir(path, by string) ([]string, error) {
	dirs, err := b.m.AddProjectDir(by, path)
	if err != nil {
		return nil, err
	}
	b.m.notify(by, by+" added "+path+" to its project", true)
	return dirs, nil
}

func (b *bridge) RemoveProjectDir(path, by string) ([]string, error) {
	dirs, err := b.m.RemoveProjectDir(by, path)
	if err != nil {
		return nil, err
	}
	b.m.notify(by, by+" removed "+path+" from its project", true)
	return dirs, nil
}

func (b *bridge) ClearProject(by string) (bool, error) {
	changed, err := b.m.ClearProject(by)
	if err != nil {
		return false, err
	}
	if changed {
		b.m.notify(by, by+" cleared its project", true)
	}
	return changed, nil
}

func projectSummary(dirs []string) string {
	if len(dirs) == 0 {
		return "no directories"
	}
	return strconv.Itoa(len(dirs)) + " directories"
}

func (b *bridge) SetEditor(editor string, terminal *bool, by string) (string, error) {
	path, err := b.m.SetEditor(editor, terminal)
	if err != nil {
		return "", err
	}
	b.m.notify(by, by+" "+editorNotice(editor, terminal), false)
	return path, nil
}

func (b *bridge) UnsetEditor(field, by string) (string, bool, error) {
	path, changed, err := b.m.UnsetEditor(field)
	if err != nil {
		return "", false, err
	}
	if changed {
		b.m.notify(by, by+" cleared "+clearedNotice(field), false)
	}
	return path, changed, nil
}

func (b *bridge) SetConfig(path string, value json.RawMessage, by string) (string, error) {
	file, err := b.m.SetConfig(path, value)
	if err != nil {
		return "", err
	}
	b.m.notify(by, by+" set "+path+" in the settings", false)
	return file, nil
}

func (b *bridge) UnsetConfig(path, by string) (string, bool, error) {
	file, changed, err := b.m.UnsetConfig(path)
	if err != nil {
		return "", false, err
	}
	if changed {
		b.m.notify(by, by+" cleared "+path+" from the settings", false)
	}
	return file, changed, nil
}

func (b *bridge) SetBlockCap(bucket string, rows *int, by string) (string, error) {
	path, err := b.m.SetBlockCap(bucket, rows)
	if err != nil {
		return "", err
	}
	b.m.notify(by, by+" "+blockCapNotice(bucket, rows), false)
	return path, nil
}

func (b *bridge) UnsetBlockCap(bucket, by string) (string, bool, error) {
	path, changed, err := b.m.UnsetBlockCap(bucket)
	if err != nil {
		return "", false, err
	}
	if changed {
		b.m.notify(by, by+" "+blockCapClearedNotice(bucket), false)
	}
	return path, changed, nil
}

func blockCapNotice(bucket string, rows *int) string {
	if bucket == "" {
		if rows == nil || *rows == 0 {
			return "turned the block cap off"
		}
		return "set the block cap to " + strconv.Itoa(*rows) + " rows"
	}
	if rows == nil {
		return "turned the " + bucket + " block cap off"
	}
	if *rows == 0 {
		return "set the " + bucket + " block cap to the marker only"
	}
	return "set the " + bucket + " block cap to " + strconv.Itoa(*rows) + " rows"
}

func blockCapClearedNotice(bucket string) string {
	if bucket == "" {
		return "cleared the block cap"
	}
	return "cleared the " + bucket + " block cap"
}

func clearedNotice(field string) string {
	switch field {
	case config.FieldEditor:
		return "the editor"
	case config.FieldTerminal:
		return "the terminal flag of the editor"
	default:
		return "the editor settings"
	}
}

func editorNotice(editor string, terminal *bool) string {
	if editor == "" {
		if terminal != nil && *terminal {
			return "made the editor a terminal editor"
		}
		return "made the editor a window editor"
	}
	return "set the editor to " + editor
}

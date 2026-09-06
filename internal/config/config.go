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

// DefaultQuestionCap is the lines the question modal draws of one option label
// or one option description before it caps them. See docs/tui/input.md.
const DefaultQuestionCap = 2

// The buckets a block cap keys by. A block takes the bucket of its first line.
// The question buckets cap the question modal, not the pane. See
// docs/tui/output.md and docs/tui/input.md.
const (
	BucketPrompt              = "prompt"
	BucketMessage             = "message"
	BucketTool                = "tool"
	BucketMeta                = "meta"
	BucketBash                = "bash"
	BucketError               = "error"
	BucketQuestionOption      = "question_option"
	BucketQuestionDescription = "question_description"
)

// Buckets lists the block-cap buckets, in the order the docs name them.
var Buckets = []string{
	BucketPrompt, BucketMessage, BucketTool, BucketMeta, BucketBash, BucketError,
	BucketQuestionOption, BucketQuestionDescription,
}

// bucketDefaults holds the buckets that take a cap of their own when the
// settings name none, in place of the global BlockCap default.
var bucketDefaults = map[string]int{
	BucketQuestionOption:      DefaultQuestionCap,
	BucketQuestionDescription: DefaultQuestionCap,
}

// ValidBucket reports whether name is a block-cap bucket.
func ValidBucket(name string) bool {
	for _, b := range Buckets {
		if b == name {
			return true
		}
	}
	return false
}

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
	// BlockCaps caps a block by its bucket. A number draws that many rows (0
	// draws only the marker), null never caps, and an absent bucket takes
	// BlockCap. See docs/config.md.
	BlockCaps map[string]*int `json:"blockCaps,omitempty"`
	// Layouts holds the named interface layouts, keyed by name. See
	// docs/config.md.
	Layouts map[string]Layout `json:"layouts,omitempty"`
	// ActiveLayout names the layout every session takes, unless the session
	// names its own. An empty string takes the built-in defaults.
	ActiveLayout string `json:"activeLayout,omitempty"`
	// DefaultModel, DefaultPermissionMode, DefaultEffort, and DefaultControl set
	// the option each field of the new session form opens on. See
	// docs/config/new-session.md.
	DefaultModel          string `json:"defaultModel,omitempty"`
	DefaultPermissionMode string `json:"defaultPermissionMode,omitempty"`
	DefaultEffort         string `json:"defaultEffort,omitempty"`
	DefaultControl        *bool  `json:"defaultControl,omitempty"`
}

// BlockCapOrDefault reads the cap out of the settings, and gives the default
// when the settings name none.
func BlockCapOrDefault(cfg Config) int {
	if cfg.BlockCap == nil {
		return DefaultBlockCap
	}
	return *cfg.BlockCap
}

// BlockCapFor resolves the cap for one bucket to a row count with two sentinels:
// -1 never caps, 0 draws only the marker, and a positive number draws that many
// rows. A bucket with its own entry wins; a question bucket then takes
// DefaultQuestionCap, and any other bucket takes the global BlockCap default.
// See docs/config.md.
func BlockCapFor(cfg Config, bucket string) int {
	if v, ok := cfg.BlockCaps[bucket]; ok {
		if v == nil || *v < 0 {
			return -1
		}
		return *v
	}
	if def, ok := bucketDefaults[bucket]; ok {
		if def > 0 {
			return def
		}
		return -1
	}
	if g := BlockCapOrDefault(cfg); g > 0 {
		return g
	}
	return -1
}

// ResolveBlockCaps resolves every bucket to its row count, so the pane holds one
// map of resolved caps. See docs/tui/output.md.
func ResolveBlockCaps(cfg Config) map[string]int {
	caps := make(map[string]int, len(Buckets))
	for _, b := range Buckets {
		caps[b] = BlockCapFor(cfg, b)
	}
	return caps
}

// The built-in layout dimensions, used when no layout sets one. See
// docs/config.md and docs/tui.md.
const (
	DefaultSidebarWidth = 26
	DefaultTaskWidth    = 32
	DefaultDiffWidth    = 32
	DefaultPromptMin    = 1
	DefaultPromptMax    = 4
)

// Layout holds the interface dimensions a named layout sets. A nil field takes
// the built-in default, so a layout may set only some of them. See
// docs/config.md.
type Layout struct {
	PromptMin    *int `json:"promptMin,omitempty"`
	PromptMax    *int `json:"promptMax,omitempty"`
	SidebarWidth *int `json:"sidebarWidth,omitempty"`
	TaskWidth    *int `json:"taskWidth,omitempty"`
	DiffWidth    *int `json:"diffWidth,omitempty"`
}

// ResolvedLayout holds the dimensions after resolution, each one a concrete
// number the interface reads.
type ResolvedLayout struct {
	PromptMin    int
	PromptMax    int
	SidebarWidth int
	TaskWidth    int
	DiffWidth    int
}

// DefaultLayout gives the built-in dimensions.
func DefaultLayout() ResolvedLayout {
	return ResolvedLayout{
		PromptMin:    DefaultPromptMin,
		PromptMax:    DefaultPromptMax,
		SidebarWidth: DefaultSidebarWidth,
		TaskWidth:    DefaultTaskWidth,
		DiffWidth:    DefaultDiffWidth,
	}
}

// ResolveLayout overlays the global layout, then the session layout, onto the
// built-in defaults. A name that no layout holds falls through to the next
// level, so a deleted layout never breaks a session. See docs/config.md.
func ResolveLayout(layouts map[string]Layout, global, session string) ResolvedLayout {
	out := DefaultLayout()
	overlay := func(name string) {
		layout, ok := layouts[name]
		if name == "" || !ok {
			return
		}
		if layout.PromptMin != nil {
			out.PromptMin = *layout.PromptMin
		}
		if layout.PromptMax != nil {
			out.PromptMax = *layout.PromptMax
		}
		if layout.SidebarWidth != nil {
			out.SidebarWidth = *layout.SidebarWidth
		}
		if layout.TaskWidth != nil {
			out.TaskWidth = *layout.TaskWidth
		}
		if layout.DiffWidth != nil {
			out.DiffWidth = *layout.DiffWidth
		}
	}
	overlay(global)
	overlay(session)
	return out.sane()
}

// sane keeps every dimension inside a readable range that does not depend on the
// terminal size. The interface clamps each width again to the terminal it draws
// in. See docs/tui.md.
func (r ResolvedLayout) sane() ResolvedLayout {
	if r.PromptMin < 1 {
		r.PromptMin = 1
	}
	if r.PromptMax < r.PromptMin {
		r.PromptMax = r.PromptMin
	}
	if r.SidebarWidth < DefaultSidebarWidth-16 {
		r.SidebarWidth = DefaultSidebarWidth - 16
	}
	if r.TaskWidth < 16 {
		r.TaskWidth = 16
	}
	if r.DiffWidth < 16 {
		r.DiffWidth = 16
	}
	return r
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

// SameEditor reports whether two settings hold the same editor and terminal
// flag. Clear touches only these two fields, so UnsetEditor asks whether it
// changed anything.
func SameEditor(a, b Config) bool {
	if a.Editor != b.Editor {
		return false
	}
	if (a.EditorTerminal == nil) != (b.EditorTerminal == nil) {
		return false
	}
	return a.EditorTerminal == nil || *a.EditorTerminal == *b.EditorTerminal
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
	if len(flags.BlockCaps) > 0 {
		merged := make(map[string]*int, len(out.BlockCaps)+len(flags.BlockCaps))
		for bucket, rows := range out.BlockCaps {
			merged[bucket] = rows
		}
		for bucket, rows := range flags.BlockCaps {
			merged[bucket] = rows
		}
		out.BlockCaps = merged
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

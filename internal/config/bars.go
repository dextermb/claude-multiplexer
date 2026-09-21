package config

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// The default composition of the two status bars. The multiplexer reads these
// to build the default bars, and the MCP tool get_bar_defaults returns them, so
// the render, the tool, and the doc read one source. See docs/config/bars.md.
//
//go:embed bars/session.json bars/status.json
var barDefaults embed.FS

// The two bars a settings file may compose.
const (
	BarSession = "session"
	BarStatus  = "status"
)

// DefaultBarRefresh is the interval a custom element re-runs its script when it
// names none. See docs/config/bars.md.
const DefaultBarRefresh = 3 * time.Second

// MinBarRefresh is the floor a refresh interval takes, so a script cannot run
// faster than this.
const MinBarRefresh = 500 * time.Millisecond

// Bars holds the composition of the two status bars. A nil bar, or a nil side,
// takes the embedded default. See docs/config/bars.md.
type Bars struct {
	Session *BarSpec `json:"session,omitempty"`
	Status  *BarSpec `json:"status,omitempty"`
}

// BarSpec is the ordered elements of one bar. A nil side takes the default of
// that side, and an empty side ([]) draws nothing.
type BarSpec struct {
	Left  []BarElement `json:"left,omitempty"`
	Right []BarElement `json:"right,omitempty"`
}

// BarElement is one element of a bar. A bare string in JSON is a built-in id. An
// object is a custom element that runs a script. Script marks it custom, and
// Label names it in the payload and the notices. See docs/config/bars.md.
type BarElement struct {
	ID      string `json:"id,omitempty"`
	Script  string `json:"script,omitempty"`
	Label   string `json:"label,omitempty"`
	Refresh string `json:"refresh,omitempty"`
	Style   string `json:"style,omitempty"`
}

// UnmarshalJSON reads a bare string as the id, and an object as a custom
// element, so the settings file may write either form.
func (e *BarElement) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "\"") {
		var id string
		if err := json.Unmarshal(data, &id); err != nil {
			return err
		}
		*e = BarElement{ID: id}
		return nil
	}
	type raw BarElement
	var out raw
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&out); err != nil {
		return err
	}
	*e = BarElement(out)
	return nil
}

// Custom reports whether an element runs a script, rather than draws a built-in.
func (e BarElement) Custom() bool {
	return strings.TrimSpace(e.Script) != ""
}

// The built-in element ids, per bar and side. An id outside its set is an error
// at load. See docs/config/bars.md.
var builtinBarIDs = map[string]map[string][]string{
	BarSession: {
		"left":  {"name", "control", "model", "mode", "effort", "workItem"},
		"right": {"state", "diff", "pr", "context", "raw", "scroll", "jobs", "queued", "tokens", "cache", "cost"},
	},
	BarStatus: {
		"left":  {"sessions", "busy", "cost", "status"},
		"right": {"hints"},
	},
}

// barScriptRunners maps a script extension to the command that runs it. The
// config validates the extension, and the interface runs the command. See
// docs/config/bars.md.
var barScriptRunners = map[string][]string{
	".go": {"go", "run"},
	".py": {"python3"},
	".sh": {"bash"},
}

// BarScriptRunner gives the command and its leading arguments for a script path,
// by the file extension, and false when the extension is not one it runs.
func BarScriptRunner(path string) ([]string, bool) {
	cmd, ok := barScriptRunners[strings.ToLower(filepath.Ext(path))]
	if !ok {
		return nil, false
	}
	out := make([]string, len(cmd))
	copy(out, cmd)
	return out, true
}

// DefaultBarSpec parses the embedded default of one bar: BarSession or
// BarStatus.
func DefaultBarSpec(bar string) (BarSpec, error) {
	data, err := DefaultBarJSON(bar)
	if err != nil {
		return BarSpec{}, err
	}
	var spec BarSpec
	if err := json.Unmarshal([]byte(data), &spec); err != nil {
		return BarSpec{}, err
	}
	return spec, nil
}

// DefaultBarJSON returns the embedded default of one bar as text, for the MCP
// tool and the doc.
func DefaultBarJSON(bar string) (string, error) {
	if bar != BarSession && bar != BarStatus {
		return "", errors.New("config: unknown bar " + bar)
	}
	data, err := barDefaults.ReadFile("bars/" + bar + ".json")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ResolveBarSpec resolves the composition of one bar: the configured side when
// the settings name it, and the embedded default of that side when they do not.
func ResolveBarSpec(bars *Bars, bar string) (BarSpec, error) {
	def, err := DefaultBarSpec(bar)
	if err != nil {
		return BarSpec{}, err
	}
	user := barSide(bars, bar)
	if user == nil {
		return def, nil
	}
	out := def
	if user.Left != nil {
		out.Left = user.Left
	}
	if user.Right != nil {
		out.Right = user.Right
	}
	return out, nil
}

func barSide(bars *Bars, bar string) *BarSpec {
	if bars == nil {
		return nil
	}
	switch bar {
	case BarSession:
		return bars.Session
	case BarStatus:
		return bars.Status
	default:
		return nil
	}
}

// BarRefresh gives the refresh interval of a custom element: the parsed value,
// the default when it names none, and the floor when it names a shorter one.
func (e BarElement) BarRefresh() time.Duration {
	if strings.TrimSpace(e.Refresh) == "" {
		return DefaultBarRefresh
	}
	d, err := time.ParseDuration(e.Refresh)
	if err != nil || d <= 0 {
		return DefaultBarRefresh
	}
	if d < MinBarRefresh {
		return MinBarRefresh
	}
	return d
}

// ValidateBars checks the bars block: each built-in id against its bar and side,
// each script extension, and each refresh. A bad value stops the program at
// start, so a typing mistake is not silent. See docs/config/bars.md.
func ValidateBars(bars *Bars) error {
	if bars == nil {
		return nil
	}
	for _, bar := range []string{BarSession, BarStatus} {
		spec := barSide(bars, bar)
		if spec == nil {
			continue
		}
		if err := validateBarSide(bar, "left", spec.Left); err != nil {
			return err
		}
		if err := validateBarSide(bar, "right", spec.Right); err != nil {
			return err
		}
	}
	return nil
}

func validateBarSide(bar, side string, elements []BarElement) error {
	for _, e := range elements {
		if e.Custom() {
			if err := validateCustomElement(bar, side, e); err != nil {
				return err
			}
			continue
		}
		if !validBuiltinID(bar, side, e.ID) {
			return fmt.Errorf("config: %q is not a built-in element of bars.%s.%s", e.ID, bar, side)
		}
	}
	return nil
}

func validateCustomElement(bar, side string, e BarElement) error {
	if _, ok := BarScriptRunner(e.Script); !ok {
		return fmt.Errorf("config: the script %q of bars.%s.%s has no known extension (.go, .py, or .sh)", e.Script, bar, side)
	}
	if strings.TrimSpace(e.Refresh) != "" {
		if _, err := time.ParseDuration(e.Refresh); err != nil {
			return fmt.Errorf("config: the refresh %q of a bars.%s.%s element is not a duration", e.Refresh, bar, side)
		}
	}
	return nil
}

func validBuiltinID(bar, side, id string) bool {
	for _, name := range builtinBarIDs[bar][side] {
		if name == id {
			return true
		}
	}
	return false
}

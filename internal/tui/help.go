package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type binding struct {
	group string
	keys  string
	what  string
	brief string
}

var bindings = []binding{
	{"Sessions", "n  ctrl+n", "Start a new session", "n new"},
	{"Sessions", "t  ctrl+p", "Open the preset prompts", "t preset"},
	{"Sessions", "enter", "Type into a session, or resume one that is not running", ""},
	{"Sessions", "r", "Resume the selected session", "r resume"},
	{"Sessions", "a", "Archive the selected session, or bring it back", "a archive"},
	{"Sessions", "A", "Show or hide archived sessions", ""},
	{"Sessions", "x  ctrl+x", "Stop the selected session, after a confirmation", "x stop"},

	{"Moving", "j  k  up  down", "Move through the list, or scroll the output", ""},
	{"Moving", "tab", "Move to the next pane", ""},
	{"Moving", "esc", "Leave the prompt or the output, or close a dialog", ""},

	{"The prompt", "enter", "Send what you typed", ""},
	{"The prompt", "ctrl+j", "Add a new line inside the prompt", ""},
	{"The prompt", "tab", "Complete a /preset name, or move on", ""},
	{"The prompt", "/", "Start a preset name, and see what matches", ""},

	{"The output", "u  d", "Scroll half a pane", ""},
	{"The output", "pgup  pgdown", "Scroll a whole pane", ""},
	{"The output", "g  G  home  end", "Go to the top, and to the bottom", ""},
	{"The output", "m", "Switch between rendered markdown and raw text", ""},

	{"The new session form", "tab", "Complete the path, or move to the next field", ""},
	{"The new session form", "shift+tab", "Walk the suggested paths, or move back a field", ""},

	{"The list and the output", "?", "Show this list", "? keys"},
	{"The list and the output", "q", "Stop every session, and quit", "q quit"},

	{"Everywhere", "ctrl+t", "Turn the mouse on or off", ""},
	{"Everywhere", "ctrl+c", "Clear the prompt. Press it again to quit", ""},
}

func statusHints() string {
	var out []string
	for _, item := range bindings {
		if item.brief != "" {
			out = append(out, item.brief)
		}
	}
	return strings.Join(out, " · ")
}

type help struct {
	filter textinput.Model
	offset int
}

func newHelp() *help {
	filter := textinput.New()
	filter.Placeholder = "search the keys"
	filter.Prompt = "> "
	filter.CharLimit = 40
	filter.Width = 30
	filter.Focus()
	return &help{filter: filter}
}

func (h *help) Update(msg tea.Msg) (bool, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc", "enter":
			return false, nil
		case "up", "ctrl+k":
			if h.offset > 0 {
				h.offset--
			}
			return true, nil
		case "down", "ctrl+j":
			h.offset++
			return true, nil
		}
	}
	var cmd tea.Cmd
	h.filter, cmd = h.filter.Update(msg)
	h.offset = 0
	return true, cmd
}

func (h *help) rows(width int) []string {
	needle := strings.ToLower(strings.TrimSpace(h.filter.Value()))
	var (
		out   []string
		group string
	)
	for _, item := range bindings {
		if needle != "" &&
			!strings.Contains(strings.ToLower(item.keys), needle) &&
			!strings.Contains(strings.ToLower(item.what), needle) &&
			!strings.Contains(strings.ToLower(item.group), needle) {
			continue
		}
		if item.group != group {
			group = item.group
			out = append(out, titleStyle.Render(group))
		}
		out = append(out, "  "+fieldLabelStyle.Render(pad(item.keys, 17))+truncate(item.what, width-21))
	}
	return out
}

func (h *help) View(width, height int) string {
	inner := width - 8
	if inner < 40 {
		inner = 40
	}
	window := height - 10
	if window < 3 {
		window = 3
	}

	rows := h.rows(inner)
	if h.offset > len(rows)-window {
		h.offset = len(rows) - window
	}
	if h.offset < 0 {
		h.offset = 0
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("Keys"))
	b.WriteString("\n\n")
	b.WriteString(h.filter.View())
	b.WriteString("\n\n")

	if len(rows) == 0 {
		b.WriteString(hintStyle.Render("No key matches that."))
	}
	end := h.offset + window
	if end > len(rows) {
		end = len(rows)
	}
	for _, row := range rows[h.offset:end] {
		b.WriteString(row + "\n")
	}

	footer := "esc close"
	if len(rows) > window {
		footer = "↑↓ scroll · " + footer
	}
	b.WriteString("\n" + hintStyle.Render(footer))
	return modalStyle.Width(inner).Render(b.String())
}

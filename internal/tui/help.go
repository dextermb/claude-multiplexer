package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type binding struct {
	group  string
	target string
	keys   string
	what   string
	brief  string
}

var bindings = []binding{
	{"Quick keys", "", "n  ctrl+n", "Start a new session", "n new"},
	{"Quick keys", "", "t  ctrl+p", "Open the preset prompts", "t preset"},

	{"The targets", "", "s  ctrl+s", "Act on the selected session", "s session"},
	{"The targets", "", "l  ctrl+l", "Act on the list", "l list"},
	{"The targets", "", "o  ctrl+o", "Act on the output pane", "o output"},

	{"The session (s)", "s", "s c", "Start a new session", "c new"},
	{"The session (s)", "s", "s t", "Open the preset prompts", "t preset"},
	{"The session (s)", "s", "s r", "Resume the selected session", "r resume"},
	{"The session (s)", "s", "s n", "Rename the selected session", "n name"},
	{"The session (s)", "s", "s a", "Archive the selected session, or bring it back", "a archive"},
	{"The session (s)", "s", "s x", "Stop the selected session, after a confirmation", "x stop"},
	{"The session (s)", "s", "s j", "Show the background jobs of the selected session", "j jobs"},
	{"The session (s)", "s", "s f", "Open the working directory in the file manager", "f folder"},
	{"The session (s)", "s", "s d", "Show the working-tree diff of the selected session", "d diff"},
	{"The session (s)", "s", "s E", "Open the working directory in the editor", "E editor"},
	{"The session (s)", "s", "s m", "Change the model", "m model"},
	{"The session (s)", "s", "s e", "Change the effort (the thinking budget)", "e effort"},
	{"The session (s)", "s", "s p", "Change the permission mode", "p mode"},

	{"The list (l)", "l", "l f", "Fold or unfold the group of the selected session", "f fold"},
	{"The list (l)", "l", "l F", "Fold every group but this one", "F others"},
	{"The list (l)", "l", "l u", "Unfold every group", "u unfold"},
	{"The list (l)", "l", "l a", "Show or hide the archived sessions", "a archived"},
	{"The list (l)", "l", "l t", "Hide or show the sidebar", "t sidebar"},
	{"The list (l)", "l", "l c", "Hide the sidebar, for more pane width", "c hide"},
	{"The list (l)", "l", "l e", "Show the sidebar again", "e show"},

	{"The output pane (o)", "o", "o m", "Switch between rendered markdown and raw text", "m markdown"},
	{"The output pane (o)", "o", "o l", "Open the layouts, to switch between them", "l layouts"},

	{"The diff panel (d)", "d", "d +", "Widen the diff panel", "+ wider"},
	{"The diff panel (d)", "d", "d -", "Narrow the diff panel", "- narrower"},
	{"The diff panel (d)", "d", "d /", "Show the panel at half the screen, or the set width", "/ half"},
	{"The diff panel (d)", "d", "d n", "Show or hide the line numbers", "n numbers"},
	{"The diff panel (d)", "", "j  k", "Step through an open diff, then to the next or previous file", ""},
	{"The diff panel (d)", "", "g  G", "Go to the top or bottom of the open diff, or of the file list", ""},
	{"The diff panel (d)", "", "}  {", "Jump to the next or previous empty line of an open diff", ""},

	{"Moving", "", "j  k  up  down", "Move through the list, or scroll the output", ""},
	{"Moving", "", "tab", "Move to the next pane", ""},
	{"Moving", "", "enter", "Type into a session, or resume one that is not running", ""},
	{"Moving", "", "esc", "Leave the prompt or the output, close a dialog, or cancel a sequence", ""},

	{"The prompt", "", "enter", "Send what you typed", ""},
	{"The prompt", "", "ctrl+j", "Add a new line inside the prompt", ""},
	{"The prompt", "", "tab", "Complete a /preset name, or move on", ""},
	{"The prompt", "", "/", "Start a preset name, and see what matches", ""},

	{"The output", "", "u  d", "Scroll half a pane", ""},
	{"The output", "", "pgup  pgdown", "Scroll a whole pane", ""},
	{"The output", "", "g  G  home  end", "Go to the top, and to the bottom", ""},
	{"The output", "", "[  ]", "Move the cursor between the capped blocks", ""},
	{"The output", "", "enter  space", "Open or close the block under the cursor", ""},
	{"The output", "", "i", "Move to the prompt", ""},

	{"The new session form", "", "tab", "Complete the path, or move to the next field", ""},
	{"The new session form", "", "shift+tab", "Walk the suggested paths, or move back a field", ""},

	{"The list and the output", "", "?", "Show this list", "? keys"},
	{"The list and the output", "", "q", "Stop every session, and quit", "q quit"},

	{"Everywhere", "", "ctrl+t", "Turn the mouse on or off", ""},
	{"Everywhere", "", "ctrl+c", "Clear the prompt. Press it again to quit", ""},
}

// statusHints lists the keys that work on their own, for the status bar.
func statusHints() string {
	var out []string
	for _, item := range bindings {
		if item.target == "" && item.brief != "" {
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
	inner := modalInner(width)
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

package tui

import (
	"fmt"
	"strings"
	"unicode"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/keys"
)

// helpEntry is one key in the key list: its keys, what it does, and what runs
// when you press enter on it.
type helpEntry struct {
	group   string
	keys    string
	what    string
	action  keys.Action
	command *config.Command
}

// help is the key list: every key in up to three columns, a filter, and a
// cursor that runs the key under it. See docs/tui/keys.md.
type help struct {
	filter textinput.Model
	cursor int
	offset int
	// rows and cols are the rows the columns show and the number of columns at
	// the last draw, so the arrow keys move in the grid the screen shows.
	rows int
	cols int
	grid [][]int
}

const (
	keyColumn          = 12
	keyListColumnWidth = 64
)

func newHelp() *help {
	filter := newTextInput()
	filter.Placeholder = "type to filter · a key or a word"
	filter.Prompt = "/ "
	filter.CharLimit = 40
	filter.SetWidth(40)
	filter.Focus()
	return &help{filter: filter}
}

// Update moves the cursor, edits the filter, and closes the list. It returns the
// entry to run when you press enter on one.
func (h *help) Update(msg tea.Msg, km keys.Keymap, cmds []config.Command) (bool, *helpEntry, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		entries := h.entries(km, cmds)
		switch key.String() {
		case "esc":
			return false, nil, nil
		case "enter":
			if h.cursor < len(entries) {
				e := entries[h.cursor]
				return false, &e, nil
			}
			return false, nil, nil
		case "up", "ctrl+k":
			h.moveTo(h.cursor - 1)
			return true, nil, nil
		case "down", "ctrl+j":
			h.moveTo(minInt(h.cursor+1, len(entries)-1))
			return true, nil, nil
		case "left":
			h.side(-1)
			return true, nil, nil
		case "right":
			h.side(1)
			return true, nil, nil
		}
	}
	var cmd tea.Cmd
	before := h.filter.Value()
	h.filter, cmd = h.filter.Update(msg)
	if h.filter.Value() != before {
		h.cursor, h.offset = 0, 0
	}
	return true, nil, cmd
}

func (h *help) moveTo(i int) {
	if i < 0 {
		i = 0
	}
	h.cursor = i
}

// side moves the cursor to the column beside it, to the row nearest the one it
// is on.
func (h *help) side(delta int) {
	col, row := h.place()
	next := col + delta
	if next < 0 || next >= len(h.grid) || len(h.grid[next]) == 0 {
		return
	}
	target := h.grid[next][len(h.grid[next])-1]
	for _, at := range h.grid[next] {
		if _, r := h.placeOf(at); r >= row {
			target = at
			break
		}
	}
	h.cursor = target
}

// entries lists the keys the filter lets through, in the order of the table.
func (h *help) entries(km keys.Keymap, cmds []config.Command) []helpEntry {
	needle := strings.ToLower(strings.TrimSpace(h.filter.Value()))
	match := func(parts ...string) bool {
		if needle == "" {
			return true
		}
		for _, part := range parts {
			if strings.Contains(strings.ToLower(part), needle) {
				return true
			}
		}
		return false
	}
	var out []helpEntry
	for _, item := range bindings {
		shown := item.keys
		if item.action != "" {
			shown = displayKeys(km, item.action)
		}
		if match(shown, item.what, item.group) {
			out = append(out, helpEntry{group: item.group, keys: shown, what: item.what, action: item.action})
		}
	}
	for i := range cmds {
		c := cmds[i]
		if match(c.Keys, c.Label, "your commands") {
			out = append(out, helpEntry{group: "Your commands", keys: c.Keys, what: c.Label, command: &c})
		}
	}
	return out
}

// keyColumns splits the entries into columns at group edges, so each column holds
// about the same number of rows. Each column is a list of lines: -1 for a group
// label, else the index of an entry.
func keyColumns(entries []helpEntry, cols int) [][]int {
	type group struct{ from, to int }
	var groups []group
	for i, e := range entries {
		if i == 0 || e.group != entries[i-1].group {
			groups = append(groups, group{from: i, to: i})
		}
		groups[len(groups)-1].to = i + 1
	}
	total := 0
	for _, g := range groups {
		total += g.to - g.from + 2
	}
	target := (total + cols - 1) / maxInt(1, cols)
	out := make([][]int, 1, cols)
	for _, g := range groups {
		col := &out[len(out)-1]
		if len(*col) > 0 && len(*col)+g.to-g.from+1 > target && len(out) < cols {
			out = append(out, nil)
			col = &out[len(out)-1]
		}
		if len(*col) > 0 {
			*col = append(*col, -2)
		}
		*col = append(*col, -1)
		for i := g.from; i < g.to; i++ {
			*col = append(*col, i)
		}
	}
	return out
}

func (h *help) placeOf(entry int) (col, row int) {
	for c, lines := range h.grid {
		for r, at := range lines {
			if at == entry {
				return c, r
			}
		}
	}
	return 0, 0
}

func (h *help) place() (col, row int) { return h.placeOf(h.cursor) }

// runnable says whether enter can run an entry from the list: a two-key action,
// a global key, or one of your commands.
func runnable(e helpEntry) bool {
	if e.command != nil {
		return true
	}
	if _, ok := chordActions[e.action]; ok {
		return true
	}
	return e.action != "" && keys.ContextOf(e.action) == keys.CtxGlobal
}

// runHelpEntry runs the key you chose in the key list. A key that works only in
// one pane cannot run from here, so the status bar says where it works.
func (m Model) runHelpEntry(e helpEntry) (tea.Model, tea.Cmd) {
	switch {
	case e.command != nil:
		return m.runCommand(*e.command)
	case chordActions[e.action] != nil:
		return chordActions[e.action](m)
	case e.action != "" && keys.ContextOf(e.action) == keys.CtxGlobal:
		return m.runGlobal(e.action)
	}
	m.status = "press " + e.keys + " where it works: " + strings.ToLower(e.group)
	return m, nil
}

func sentence(text string) string {
	runes := []rune(text)
	if len(runes) == 0 {
		return text
	}
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func (h *help) View(km keys.Keymap, cmds []config.Command, width, height int) string {
	inner := modalInner(width)
	entries := h.entries(km, cmds)
	if h.cursor >= len(entries) {
		h.cursor = maxInt(0, len(entries)-1)
	}
	room := inner - 4
	cols := minInt(3, maxInt(1, (room+1)/keyListColumnWidth))
	h.cols = cols
	h.grid = keyColumns(entries, cols)
	colWidth := (room - (cols - 1)) / cols

	window := maxInt(3, height-10)
	longest := 0
	for _, lines := range h.grid {
		longest = maxInt(longest, len(lines))
	}
	_, row := h.place()
	if row < h.offset {
		h.offset = row
	}
	if row >= h.offset+window {
		h.offset = row - window + 1
	}
	h.offset = maxInt(0, minInt(h.offset, longest-window))
	h.rows = minInt(window, longest)

	var b strings.Builder
	b.WriteString(titleStyle.Render("Keys") + hintStyle.Render(fmt.Sprintf("(%d)", len(entries))))
	b.WriteString("\n\n")
	b.WriteString(h.filter.View())
	b.WriteString("\n\n")

	if len(entries) == 0 {
		b.WriteString(hintStyle.Render("No key matches that."))
	} else {
		columns := make([]string, cols)
		for c := 0; c < cols; c++ {
			w := colWidth
			if c == cols-1 {
				w = room - (cols-1)*(colWidth+1)
			}
			var lines []string
			if c < len(h.grid) {
				for r := h.offset; r < h.offset+h.rows; r++ {
					line := ""
					if r < len(h.grid[c]) {
						line = h.line(entries, h.grid[c], r, w)
					}
					lines = append(lines, lipgloss.NewStyle().Width(w).MaxWidth(w).Render(line))
				}
			}
			columns[c] = strings.Join(lines, "\n")
		}
		divider := strings.TrimSuffix(strings.Repeat(ruleStyle.Render("│")+"\n", h.rows), "\n")
		parts := []string{columns[0]}
		for c := 1; c < cols; c++ {
			parts = append(parts, divider, columns[c])
		}
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, parts...))
	}

	footer := "↑↓ move · enter run the key · esc close"
	if cols > 1 {
		footer = "↑↓ move · ←→ column · enter run the key · esc close"
	}
	b.WriteString("\n\n" + hintStyle.Render(footer))
	return modalStyle.Width(inner + 2).Render(b.String())
}

// line draws one line of a column: a group label set in a rule, a blank line
// between groups, or a key and what it does, inverted under the cursor.
func (h *help) line(entries []helpEntry, lines []int, r, width int) string {
	at := lines[r]
	switch at {
	case -2:
		return ""
	case -1:
		return ruleLabel(entries[lines[r+1]].group, width, false)
	}
	e := entries[at]
	text := " " + pad(e.keys, keyColumn) + truncate(sentence(e.what), width-keyColumn-4)
	if at == h.cursor {
		mark := "  "
		if runnable(e) {
			mark = " →"
		}
		return invertStyle.Width(width).Render(pad(text, width-2) + mark)
	}
	keyText, whatText := keyStyle, fgStyle(colMuted)
	if e.action == keys.SessionStop {
		keyText, whatText = fgStyle(colDanger), fgStyle(colDanger)
	}
	return keyText.Render(" "+pad(e.keys, keyColumn)) + whatText.Render(truncate(sentence(e.what), width-keyColumn-4))
}

package tui

import (
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

const findContext = 2

// promptMark starts the first row of a prompt, so a needle that starts with
// promptPrefix steps from prompt to prompt. See docs/tui/output.md.
const (
	promptMark   = "› "
	promptPrefix = ">"
)

// findState searches the rendered rows of the output pane. See
// docs/tui/keys/navigation.md.
type findState struct {
	on      bool
	input   textinput.Model
	raw     string
	needle  string
	prompts bool
	hits    []int
	row     int
	origin  int
	note    string
}

func newFindState() findState {
	in := newTextInput()
	in.Prompt = "/"
	in.CharLimit = 64
	return findState{input: in, row: -1}
}

// live reports whether a needle narrows the pane, so the step keys work and the
// status row shows the count.
func (f findState) live() bool { return f.needle != "" || f.prompts }

func (f findState) index() int {
	for i, row := range f.hits {
		if row == f.row {
			return i
		}
	}
	return -1
}

// parseFind splits what you typed into the needle and whether it reads prompt
// rows only.
func parseFind(raw string) (string, bool) {
	prompts := strings.HasPrefix(raw, promptPrefix)
	needle := strings.TrimPrefix(raw, promptPrefix)
	return strings.ToLower(strings.TrimSpace(needle)), prompts
}

// findHits lists the rows that hold the needle, in row order. A prompt search
// reads only the rows the renderer marked as the head of a prompt.
func findHits(content, needle string, prompts bool) []int {
	if needle == "" && !prompts {
		return nil
	}
	var out []int
	for row, text := range plainLines(content) {
		if prompts && !strings.HasPrefix(text, promptMark) {
			continue
		}
		if needle != "" && !strings.Contains(strings.ToLower(text), needle) {
			continue
		}
		out = append(out, row)
	}
	return out
}

func (m Model) openFind() (tea.Model, tea.Cmd) {
	m.find.on = true
	m.find.origin = m.output.YOffset()
	m.find.row = -1
	m.find.note = ""
	m.find.raw = ""
	m.find.needle = ""
	m.find.prompts = false
	m.find.hits = nil
	m.find.input.SetValue("")
	m.find.input.Focus()
	return m, textinput.Blink
}

// clearFind drops the needle, so the pane stops marking a row and the step keys
// go back to what they ran before.
func (m *Model) clearFind() {
	m.find.on = false
	m.find.input.Blur()
	m.find.input.SetValue("")
	m.find.raw = ""
	m.find.needle = ""
	m.find.prompts = false
	m.find.hits = nil
	m.find.row = -1
	m.find.note = ""
	m.setContent()
}

// findKey handles a key while the find box has the focus. Every key rescans the
// pane and moves to the nearest hit above where the box opened, so the search
// is incremental.
func (m Model) findKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		offset := m.find.origin
		m.clearFind()
		m.output.SetYOffset(offset)
		return m, nil
	case "enter":
		m.find.on = false
		m.find.input.Blur()
		if !m.find.live() {
			m.clearFind()
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.find.input, cmd = m.find.input.Update(msg)
	m.find.raw = m.find.input.Value()
	m.find.needle, m.find.prompts = parseFind(m.find.raw)
	m.find.hits = findHits(m.content, m.find.needle, m.find.prompts)
	m.find.row = -1
	m.seek(m.find.origin, true)
	return m, cmd
}

// stepFind moves to the next hit. The search runs backwards, so findNext steps
// to the older hit and findPrev steps to the newer one.
func (m Model) stepFind(forward bool) (tea.Model, tea.Cmd) {
	from := m.find.row
	if from < 0 {
		from = m.output.YOffset()
	}
	m.seek(from, !forward)
	return m, nil
}

// seek moves to the first hit above from, or below it when back is false, and
// wraps around the pane when it runs out of hits.
func (m *Model) seek(from int, back bool) {
	if len(m.find.hits) == 0 {
		m.find.row = -1
		m.find.note = "no match"
		m.setContent()
		return
	}
	row, wrapped := seekRow(m.find.hits, from, back)
	m.find.row = row
	m.find.note = ""
	if wrapped {
		m.find.note = "wrapped"
	}
	m.showRow(row)
	m.setContent()
}

func seekRow(hits []int, from int, back bool) (int, bool) {
	if back {
		for i := len(hits) - 1; i >= 0; i-- {
			if hits[i] < from {
				return hits[i], false
			}
		}
		return hits[len(hits)-1], true
	}
	for _, row := range hits {
		if row > from {
			return row, false
		}
	}
	return hits[0], true
}

// showRow scrolls the pane so the row is in sight, with a little of the text
// above it for context.
func (m *Model) showRow(row int) {
	offset := row - findContext
	if offset < 0 {
		offset = 0
	}
	m.output.SetYOffset(offset)
}

// refreshFind rescans the hits after the pane is drawn again, and holds the
// current hit on its row, so new output at the bottom does not move it.
func (m *Model) refreshFind() {
	if !m.find.live() {
		m.find.hits = nil
		m.find.row = -1
		return
	}
	row := m.find.row
	m.find.hits = findHits(m.content, m.find.needle, m.find.prompts)
	if row < 0 || len(m.find.hits) == 0 {
		return
	}
	m.find.row = -1
	for i := len(m.find.hits) - 1; i >= 0; i-- {
		if m.find.hits[i] <= row {
			m.find.row = m.find.hits[i]
			return
		}
	}
	m.find.row = m.find.hits[0]
}

// findPaint marks the row of the current hit, the same as the review screen
// marks its hunk. See docs/tui/review.md.
func (m Model) findPaint(content string) string {
	if m.find.row < 0 {
		return content
	}
	rows := strings.Split(content, "\n")
	if m.find.row >= len(rows) {
		return content
	}
	plain := plainLines(content)
	width := m.textWidth()
	rows[m.find.row] = invertStyle.Width(width).MaxWidth(width).Render(trimRight(plain[m.find.row]))
	return strings.Join(rows, "\n")
}

// findStatus draws the find box, or the count of the live needle, in the status
// row.
func (m Model) findStatus() (string, bool) {
	if m.find.on {
		return m.find.input.View(), true
	}
	if !m.find.live() || m.focus != focusOutput {
		return "", false
	}
	label := statusKeyStyle.Render("/" + m.find.raw)
	return label + statusMutedStyle.Render("  "+m.findCount()), true
}

func (m Model) findCount() string {
	if len(m.find.hits) == 0 {
		return "no match"
	}
	count := strconv.Itoa(len(m.find.hits))
	at := m.find.index()
	if at < 0 {
		return count + " hits"
	}
	out := strconv.Itoa(at+1) + "/" + count
	if m.find.note != "" {
		out += " " + m.find.note
	}
	return out
}

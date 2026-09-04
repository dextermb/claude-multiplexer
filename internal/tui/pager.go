package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/dextermb/claude-multiplexer/internal/render"
)

// pager opens a collapsed result. It first lists every result that carries a
// whole body, then pages the one the user chooses. See docs/tui/output.md.
type pager struct {
	entries []pagerEntry
	cursor  int
	showing bool
	vp      viewport.Model
	width   int
	height  int
}

type pagerEntry struct {
	label string
	body  string
}

func collectExpandables(lines []render.Line) []pagerEntry {
	var out []pagerEntry
	for _, line := range lines {
		if line.Full == "" {
			continue
		}
		out = append(out, pagerEntry{label: strings.TrimSpace(line.Text), body: line.Full})
	}
	return out
}

func newPager(entries []pagerEntry, width, height int) *pager {
	p := &pager{entries: entries, width: width, height: height}
	if len(entries) > 0 {
		p.cursor = len(entries) - 1
	}
	return p
}

// Update returns whether the pager stays open.
func (p *pager) Update(msg tea.Msg) (bool, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return true, nil
	}
	if p.showing {
		return p.scroll(key), nil
	}
	switch key.String() {
	case "esc", "q":
		return false, nil
	case "up", "k":
		p.move(-1)
	case "down", "j":
		p.move(1)
	case "enter":
		p.open()
	}
	return true, nil
}

func (p *pager) scroll(key tea.KeyMsg) bool {
	switch key.String() {
	case "esc", "q":
		p.showing = false
	case "up", "k":
		p.vp.LineUp(1)
	case "down", "j":
		p.vp.LineDown(1)
	case "u", "ctrl+u":
		p.vp.HalfViewUp()
	case "d", "ctrl+d":
		p.vp.HalfViewDown()
	case "g", "home":
		p.vp.GotoTop()
	case "G", "end":
		p.vp.GotoBottom()
	case "pgup":
		p.vp.ViewUp()
	case "pgdown":
		p.vp.ViewDown()
	}
	return true
}

func (p *pager) move(delta int) {
	if len(p.entries) == 0 {
		return
	}
	p.cursor = (p.cursor + delta + len(p.entries)) % len(p.entries)
}

func (p *pager) open() {
	if len(p.entries) == 0 {
		return
	}
	vp := viewport.New(p.innerWidth(), p.docHeight())
	vp.SetContent(p.entries[p.cursor].body)
	p.vp = vp
	p.showing = true
}

func (p *pager) innerWidth() int {
	return modalInner(p.width)
}

func (p *pager) docHeight() int {
	height := p.height - 10
	if height < 3 {
		height = 3
	}
	return height
}

func (p *pager) View(width, height int) string {
	p.width, p.height = width, height
	inner := p.innerWidth()

	if p.showing {
		p.vp.Width = inner
		p.vp.Height = p.docHeight()
		var b strings.Builder
		b.WriteString(titleStyle.Render(truncate(p.entries[p.cursor].label, inner-2)))
		b.WriteString("\n\n")
		b.WriteString(p.vp.View())
		b.WriteString("\n\n" + hintStyle.Render("↑↓ scroll · g/G ends · esc back"))
		return modalStyle.Width(inner).Render(b.String())
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("Open a result"))
	b.WriteString("\n\n")
	rowWidth := inner - 4
	for i, entry := range p.entries {
		row := truncate(entry.label, rowWidth-2)
		if i == p.cursor {
			b.WriteString(selectedRowStyle.Width(rowWidth).Render("▸ " + row))
		} else {
			b.WriteString(rowStyle.Width(rowWidth).Render("  " + row))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n" + hintStyle.Render("↑↓ move · enter open · esc close"))
	return modalStyle.Width(inner).Render(b.String())
}

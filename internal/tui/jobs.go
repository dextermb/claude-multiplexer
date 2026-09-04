package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dextermb/claude-multiplexer/internal/session"
)

// jobsModal lists the background jobs of one session, running first, in start
// order. It reads the jobs the row already holds. See docs/tui/sessions.md.
type jobsModal struct {
	name   string
	jobs   []session.Job
	vp     viewport.Model
	width  int
	height int
	ready  bool
}

func newJobsModal(name string, jobs []session.Job, width, height int) *jobsModal {
	return &jobsModal{name: name, jobs: orderJobs(jobs), width: width, height: height}
}

func orderJobs(jobs []session.Job) []session.Job {
	out := make([]session.Job, 0, len(jobs))
	for _, job := range jobs {
		if job.Status.Running() {
			out = append(out, job)
		}
	}
	for _, job := range jobs {
		if !job.Status.Running() {
			out = append(out, job)
		}
	}
	return out
}

// Update returns whether the modal stays open.
func (m *jobsModal) Update(msg tea.Msg) (bool, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return true, nil
	}
	switch key.String() {
	case "esc", "q", "J":
		return false, nil
	case "up", "k":
		m.vp.LineUp(1)
	case "down", "j":
		m.vp.LineDown(1)
	case "u", "ctrl+u":
		m.vp.HalfViewUp()
	case "d", "ctrl+d":
		m.vp.HalfViewDown()
	case "g", "home":
		m.vp.GotoTop()
	case "G", "end":
		m.vp.GotoBottom()
	case "pgup":
		m.vp.ViewUp()
	case "pgdown":
		m.vp.ViewDown()
	}
	return true, nil
}

func (m *jobsModal) innerWidth() int {
	inner := m.width - 8
	if inner < 40 {
		inner = 40
	}
	return inner
}

func (m *jobsModal) docHeight() int {
	height := m.height - 8
	if height < 3 {
		height = 3
	}
	return height
}

func (m *jobsModal) View(width, height int) string {
	m.width, m.height = width, height
	inner := m.innerWidth()

	if !m.ready {
		m.vp = viewport.New(inner, m.docHeight())
		m.ready = true
	}
	m.vp.Width = inner
	m.vp.Height = m.docHeight()
	m.vp.SetContent(m.body(inner))

	var b strings.Builder
	b.WriteString(titleStyle.Render("Background jobs · " + m.name))
	b.WriteString("\n\n")
	b.WriteString(m.vp.View())
	b.WriteString("\n\n" + hintStyle.Render("↑↓ scroll · esc close"))
	return modalStyle.Width(inner).Render(b.String())
}

func (m *jobsModal) body(width int) string {
	if len(m.jobs) == 0 {
		return rowMutedStyle.Render("No background jobs yet.")
	}
	var b strings.Builder
	for i, job := range m.jobs {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(jobLine(job, width))
	}
	return b.String()
}

func jobLine(job session.Job, width int) string {
	head := jobStyle(job.Status).Render(fmt.Sprintf("%s %s", jobGlyph(job.Status), job.Status.String()))
	desc := job.Description
	if desc == "" {
		desc = job.ID
	}
	room := width - lipgloss.Width(head) - 3
	if room < 1 {
		room = 1
	}
	return head + rowMutedStyle.Render(" · ") + rowStyle.Render(truncate(desc, room))
}

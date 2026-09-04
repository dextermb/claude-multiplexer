package tui

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dextermb/claude-multiplexer/internal/session"
)

// jobRefresh is how often an open job re-reads its output file.
const jobRefresh = 500 * time.Millisecond

// jobsModal lists the background jobs of one session, running first, in start
// order, and opens the output of the one you choose. See docs/tui/sessions.md.
type jobsModal struct {
	name    string
	jobs    []session.Job
	cursor  int
	showing bool
	body    string
	failed  error
	vp      viewport.Model
	width   int
	height  int
	ready   bool
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

// setJobs takes the current jobs of the session, and holds the cursor on the
// job it already names, because a job that stops moves down the list.
func (m *jobsModal) setJobs(jobs []session.Job) {
	var id string
	if job, ok := m.current(); ok {
		id = job.ID
	}
	m.jobs = orderJobs(jobs)
	m.cursor = 0
	for i, job := range m.jobs {
		if job.ID == id {
			m.cursor = i
			break
		}
	}
	if m.showing {
		m.refresh()
	}
}

func (m *jobsModal) current() (session.Job, bool) {
	if m.cursor < 0 || m.cursor >= len(m.jobs) {
		return session.Job{}, false
	}
	return m.jobs[m.cursor], true
}

// Update returns whether the modal stays open.
func (m *jobsModal) Update(msg tea.Msg) (bool, tea.Cmd) {
	if _, ok := msg.(jobTickMsg); ok {
		if !m.showing {
			return true, nil
		}
		m.refresh()
		return true, jobTick()
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return true, nil
	}
	if m.showing {
		return true, m.scroll(key)
	}
	switch key.String() {
	case "esc", "q", "J":
		return false, nil
	case "up", "k":
		m.move(-1)
	case "down", "j":
		m.move(1)
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		m.cursor = len(m.jobs) - 1
	case "enter":
		return true, m.open()
	}
	return true, nil
}

func (m *jobsModal) scroll(key tea.KeyMsg) tea.Cmd {
	switch key.String() {
	case "esc", "q":
		m.showing = false
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
	return nil
}

func (m *jobsModal) move(delta int) {
	if len(m.jobs) == 0 {
		return
	}
	m.cursor = (m.cursor + delta + len(m.jobs)) % len(m.jobs)
}

func (m *jobsModal) open() tea.Cmd {
	if _, ok := m.current(); !ok {
		return nil
	}
	m.vp = viewport.New(m.contentWidth(), m.docHeight())
	m.showing = true
	m.refresh()
	m.vp.GotoTop()
	return jobTick()
}

// refresh re-reads the output file, and keeps the view at the end when it
// already sits there, so a running job scrolls itself.
func (m *jobsModal) refresh() {
	job, ok := m.current()
	if !ok {
		m.showing = false
		return
	}
	follow := m.vp.AtBottom()
	body, err := session.ReadOutput(job)
	m.body, m.failed = body, err
	if errors.Is(err, session.ErrNoOutputPath) {
		m.failed = nil
	}
	m.vp.SetContent(m.detail(job, m.contentWidth()))
	if follow {
		m.vp.GotoBottom()
	}
}

func (m *jobsModal) innerWidth() int {
	return modalInner(m.width)
}

// contentWidth is the room inside the box, because modalStyle.Width counts its
// own padding and the viewport does not wrap a line that overruns it.
func (m *jobsModal) contentWidth() int {
	width := m.innerWidth() - 4
	if width < 10 {
		width = 10
	}
	return width
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
		m.vp = viewport.New(m.contentWidth(), m.docHeight())
		m.ready = true
	}
	m.vp.Width = m.contentWidth()
	m.vp.Height = m.docHeight()

	var b strings.Builder
	if job, ok := m.current(); m.showing && ok {
		m.vp.SetContent(m.detail(job, m.contentWidth()))
		b.WriteString(titleStyle.Render(truncate(jobTitle(job), inner-2)))
		b.WriteString("\n\n")
		b.WriteString(m.vp.View())
		b.WriteString("\n\n" + hintStyle.Render("↑↓ scroll · g/G ends · esc back"))
		return modalStyle.Width(inner).Render(b.String())
	}

	m.vp.SetContent(m.list(m.contentWidth()))
	b.WriteString(titleStyle.Render("Background jobs · " + m.name))
	b.WriteString("\n\n")
	b.WriteString(m.vp.View())
	b.WriteString("\n\n" + hintStyle.Render("↑↓ move · enter open · esc close"))
	return modalStyle.Width(inner).Render(b.String())
}

func (m *jobsModal) list(width int) string {
	if len(m.jobs) == 0 {
		return rowMutedStyle.Render("No background jobs yet.")
	}
	var b strings.Builder
	for i, job := range m.jobs {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(jobLine(job, width-2, i == m.cursor))
	}
	return b.String()
}

func jobLine(job session.Job, width int, selected bool) string {
	head := jobStyle(job.Status).Render(fmt.Sprintf("%s %s", jobGlyph(job.Status), job.Status.String()))
	mark, textStyle := "  ", rowStyle
	if selected {
		mark, textStyle = "▸ ", selectedRowStyle
	}
	room := width - lipgloss.Width(head) - 5
	if room < 1 {
		room = 1
	}
	return mark + head + rowMutedStyle.Render(" · ") + textStyle.Render(truncate(jobLabel(job), room))
}

func jobLabel(job session.Job) string {
	if job.Description != "" {
		return job.Description
	}
	if job.Command != "" {
		return job.Command
	}
	return job.ID
}

func jobTitle(job session.Job) string {
	return jobGlyph(job.Status) + " " + job.Status.String() + " · " + jobLabel(job)
}

const jobLabelWidth = 9

func (m *jobsModal) detail(job session.Job, width int) string {
	valueWidth := width - jobLabelWidth
	if valueWidth < 8 {
		valueWidth = 8
	}
	var b strings.Builder
	for _, field := range jobHead(job) {
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
			rowMutedStyle.Width(jobLabelWidth).Render(field.label),
			rowStyle.Width(valueWidth).Render(field.value)))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(rowMutedStyle.Render(strings.Repeat("─", width)))
	b.WriteString("\n\n")
	b.WriteString(m.outputBody(job))
	return b.String()
}

func (m *jobsModal) outputBody(job session.Job) string {
	if m.failed != nil {
		return errorStyle.Render("The output could not be read: " + m.failed.Error())
	}
	if job.OutputPath == "" {
		return rowMutedStyle.Render("Waiting for the output file.")
	}
	if strings.TrimSpace(m.body) == "" {
		return rowMutedStyle.Render("No output yet.")
	}
	return strings.TrimRight(m.body, "\n")
}

type jobField struct{ label, value string }

func jobHead(job session.Job) []jobField {
	out := []jobField{{"status", job.Status.String()}}
	if job.Command != "" {
		out = append(out, jobField{"command", job.Command})
	}
	if job.TaskType != "" {
		out = append(out, jobField{"type", job.TaskType})
	}
	if !job.StartedAt.IsZero() {
		out = append(out, jobField{"started", job.StartedAt.Format("15:04:05")})
	}
	if !job.EndedAt.IsZero() {
		out = append(out, jobField{"ended", job.EndedAt.Format("15:04:05")})
	}
	if job.Summary != "" {
		out = append(out, jobField{"summary", job.Summary})
	}
	if job.OutputPath != "" {
		out = append(out, jobField{"file", job.OutputPath})
	}
	return out
}

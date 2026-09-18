package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dextermb/claude-multiplexer/internal/git"
	"github.com/dextermb/claude-multiplexer/internal/manager"
	"github.com/dextermb/claude-multiplexer/internal/render"
)

// reviewSide is the pane the focus sits in while the review screen is open: the
// diff, the explanation thread, or the prompt for a follow-up. See
// docs/tui/review.md.
type reviewSide int

const (
	reviewDiff reviewSide = iota
	reviewExplain
	reviewPrompt
)

// explainState is the running thread of one session's review, and whether a
// turn is in flight. See docs/tui/review.md.
type explainState struct {
	pending bool
	thread  []explainTurn
}

// explainTurn is one exchange of the thread: the ask the review sent, the
// committed reply text, and the streaming tail before the turn ends.
type explainTurn struct {
	ask     string
	text    string
	partial string
}

// reviewSelected is the s R action. It opens the review screen for the selected
// session, hides the sidebar, and reads the diff. See docs/tui/review.md.
func (m Model) reviewSelected() (tea.Model, tea.Cmd) {
	if _, ok := m.selectedRow(); !ok {
		return m, nil
	}
	m.reviewMode = true
	m.reviewFile = 0
	m.reviewHunk = 0
	m.reviewScroll = 0
	m.explainScroll = 0
	m.reviewFocus = reviewDiff
	m.reviewSidebar = !m.sidebarHidden
	m.sidebarHidden = true
	m.focus = focusReview
	m.prompt.Blur()
	m.status = "review — j/k hunk · }/{ file · e explain · E file · tab pane · esc close"
	cmds := []tea.Cmd{m.diffRefreshCmd()}
	if !m.diffTicking {
		m.diffTicking = true
		cmds = append(cmds, diffTick())
	}
	if load := m.reviewLoadCmd(); load != nil {
		cmds = append(cmds, load)
	}
	return m, tea.Batch(cmds...)
}

// leaveReview closes the review screen and brings the sidebar back if it was
// shown before.
func (m Model) leaveReview() (tea.Model, tea.Cmd) {
	m.reviewMode = false
	m.prompt.Blur()
	if m.reviewSidebar {
		m.sidebarHidden = false
	}
	m.focus = focusOutput
	m.status = ""
	m.output.Width = m.outputWidth()
	m.output.Height = m.outputHeight()
	m.rebuildOutput()
	return m, nil
}

// reviewLoadCmd reads the diff of the selected file when it is not cached, so
// the review screen can split it into hunks.
func (m Model) reviewLoadCmd() tea.Cmd {
	entries := m.diffEntries()
	if m.reviewFile < 0 || m.reviewFile >= len(entries) {
		return nil
	}
	e := entries[m.reviewFile]
	if _, ok := m.fileDiffs[m.sel][e.key()]; ok {
		return nil
	}
	return fileDiffCmd(m.sel, e.dir, e.file.Path)
}

// reviewFileHunks is the hunks of the selected file, or nil when the diff is
// not cached yet or the file has no text change.
func (m Model) reviewFileHunks() []git.Hunk {
	entries := m.diffEntries()
	if m.reviewFile < 0 || m.reviewFile >= len(entries) {
		return nil
	}
	text, ok := m.fileDiffs[m.sel][entries[m.reviewFile].key()]
	if !ok {
		return nil
	}
	return git.Hunks(text)
}

func (m Model) reviewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.reviewFocus == reviewPrompt {
		return m.reviewPromptKey(msg)
	}
	switch msg.String() {
	case "esc":
		return m.leaveReview()
	case "tab":
		return m.reviewCycleFocus()
	case "e":
		return m.reviewExplainHunk()
	case "E":
		return m.reviewExplainFile()
	}
	if m.reviewFocus == reviewExplain {
		return m.reviewExplainScrollKey(msg)
	}
	return m.reviewDiffKey(msg)
}

func (m Model) reviewDiffKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		return m.reviewMoveHunk(1)
	case "k", "up":
		return m.reviewMoveHunk(-1)
	case "}", "shift+]":
		return m.reviewMoveFile(1)
	case "{", "shift+[":
		return m.reviewMoveFile(-1)
	case "g", "home":
		return m.reviewGotoFile(0)
	case "G", "end":
		return m.reviewGotoFile(len(m.diffEntries()) - 1)
	case "pgup":
		m.reviewScroll -= m.reviewPage()
		m.clampReviewScroll()
		return m, nil
	case "pgdown":
		m.reviewScroll += m.reviewPage()
		m.clampReviewScroll()
		return m, nil
	}
	return m, nil
}

func (m Model) reviewExplainScrollKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		m.explainScroll++
	case "k", "up":
		m.explainScroll--
	case "g", "home":
		m.explainScroll = 0
	case "G", "end":
		m.explainScroll = len(m.reviewExplainContent(m.reviewExplainWidth()))
	case "pgup":
		m.explainScroll -= m.reviewPage()
	case "pgdown":
		m.explainScroll += m.reviewPage()
	}
	m.clampExplainScroll()
	return m, nil
}

func (m Model) reviewPromptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.reviewFocus = reviewDiff
		m.prompt.Blur()
		return m, nil
	case "enter":
		text := strings.TrimSpace(m.prompt.Value())
		if text == "" {
			return m, nil
		}
		m.prompt.Reset()
		return m.reviewSend(text, text)
	}
	var cmd tea.Cmd
	m.prompt, cmd = m.prompt.Update(msg)
	return m, cmd
}

func (m Model) reviewCycleFocus() (tea.Model, tea.Cmd) {
	switch m.reviewFocus {
	case reviewDiff:
		m.reviewFocus = reviewExplain
		return m, nil
	case reviewExplain:
		m.reviewFocus = reviewPrompt
		m.prompt.Focus()
		return m, textarea.Blink
	default:
		m.reviewFocus = reviewDiff
		m.prompt.Blur()
		return m, nil
	}
}

// reviewMoveHunk moves the selected hunk within the file, and rolls to the next
// or the previous file at the ends.
func (m Model) reviewMoveHunk(delta int) (tea.Model, tea.Cmd) {
	hunks := m.reviewFileHunks()
	next := m.reviewHunk + delta
	if next < 0 {
		if m.reviewFile > 0 {
			return m.enterFile(m.reviewFile-1, true)
		}
		m.reviewHunk = 0
	} else if next >= len(hunks) {
		if m.reviewFile < len(m.diffEntries())-1 {
			return m.enterFile(m.reviewFile+1, false)
		}
		if len(hunks) > 0 {
			m.reviewHunk = len(hunks) - 1
		}
	} else {
		m.reviewHunk = next
	}
	m.ensureReviewHunkVisible()
	return m, nil
}

func (m Model) reviewMoveFile(delta int) (tea.Model, tea.Cmd) {
	return m.enterFile(m.reviewFile+delta, false)
}

func (m Model) reviewGotoFile(index int) (tea.Model, tea.Cmd) {
	return m.enterFile(index, false)
}

// enterFile selects a file. When last is set, it selects the file's last hunk,
// so k from the top of a file lands on the bottom of the one above.
func (m Model) enterFile(index int, last bool) (tea.Model, tea.Cmd) {
	entries := m.diffEntries()
	if index < 0 || index >= len(entries) {
		return m, nil
	}
	m.reviewFile = index
	m.reviewHunk = 0
	if load := m.reviewLoadCmd(); load != nil {
		if last {
			m.reviewHunk = 0
		}
		m.ensureReviewHunkVisible()
		return m, load
	}
	if last {
		if hunks := m.reviewFileHunks(); len(hunks) > 0 {
			m.reviewHunk = len(hunks) - 1
		}
	}
	m.ensureReviewHunkVisible()
	return m, nil
}

// reviewExplainHunk is the e action. It asks the live session to explain the
// selected hunk, named as path:start-end. See docs/tui/review.md.
func (m Model) reviewExplainHunk() (tea.Model, tea.Cmd) {
	entries := m.diffEntries()
	if m.reviewFile < 0 || m.reviewFile >= len(entries) {
		return m, nil
	}
	path := entries[m.reviewFile].file.Path
	hunks := m.reviewFileHunks()
	if m.reviewHunk < 0 || m.reviewHunk >= len(hunks) {
		return m.reviewExplainFile()
	}
	ask, prompt := hunkExplainPrompt(path, hunks[m.reviewHunk])
	return m.reviewSend(ask, prompt)
}

// hunkExplainPrompt names the hunk as path:start-end and asks the session to
// read the file for the context, so the prompt carries no diff text. See
// docs/tui/review.md.
func hunkExplainPrompt(path string, h git.Hunk) (ask, prompt string) {
	loc := fmt.Sprintf("%s:%d-%d", path, h.NewStart, h.NewEnd())
	ask = "explain " + loc
	prompt = fmt.Sprintf("Explain the change at %s. Keep the explanation short. "+
		"The change is against origin/HEAD. Read the file, or run `git diff origin/HEAD -- %s`, "+
		"for the surrounding context.", loc, path)
	return ask, prompt
}

// reviewExplainFile is the E action. It asks the live session to explain the
// whole selected file.
func (m Model) reviewExplainFile() (tea.Model, tea.Cmd) {
	entries := m.diffEntries()
	if m.reviewFile < 0 || m.reviewFile >= len(entries) {
		return m, nil
	}
	path := entries[m.reviewFile].file.Path
	ask, prompt := fileExplainPrompt(path)
	return m.reviewSend(ask, prompt)
}

// fileExplainPrompt names the whole file, with no line range.
func fileExplainPrompt(path string) (ask, prompt string) {
	ask = "explain " + path
	prompt = fmt.Sprintf("Explain the change to %s. Keep the explanation short. "+
		"The change is against origin/HEAD. Read the file, or run `git diff origin/HEAD -- %s`, "+
		"for the surrounding context.", path, path)
	return ask, prompt
}

// reviewSend appends a turn to the thread and sends the prompt to the live
// session. A busy session queues the prompt, the same as any prompt. See
// docs/tui/review.md.
func (m Model) reviewSend(ask, prompt string) (tea.Model, tea.Cmd) {
	item, ok := m.selectedRow()
	if !ok {
		return m, nil
	}
	if item.readOnly {
		m.errText = readOnlyStatus
		return m, nil
	}
	if !item.running() {
		m.errText = "this session is not running — press Enter to resume it"
		return m, nil
	}
	st := m.explain[m.sel]
	st.thread = append(st.thread, explainTurn{ask: ask})
	st.pending = true
	m.explain[m.sel] = st
	if err := m.mgr.Send(m.sel, prompt); err != nil {
		m.errText = err.Error()
		st.pending = false
		m.explain[m.sel] = st
		return m, nil
	}
	m.errText = ""
	m.queued[m.sel] = append(m.queued[m.sel], prompt)
	m.explainScroll = len(m.reviewExplainContent(m.reviewExplainWidth()))
	m.clampExplainScroll()
	m.refresh()
	return m, m.ensureAnimating()
}

// captureExplain mirrors the reply of the reviewed session into the last thread
// turn while a turn is pending. It runs only while the review screen is open.
// See docs/tui/review.md.
func (m *Model) captureExplain(ev manager.Event, turnEnded bool) {
	if !m.reviewMode || ev.Session != m.sel {
		return
	}
	st, ok := m.explain[ev.Session]
	if !ok || len(st.thread) == 0 || !st.pending {
		return
	}
	last := len(st.thread) - 1
	for _, line := range ev.Lines {
		if line.Class != render.ClassText || line.Text == "" {
			continue
		}
		if st.thread[last].text != "" {
			st.thread[last].text += "\n"
		}
		st.thread[last].text += line.Text
	}
	st.thread[last].partial = ev.Partial
	if turnEnded {
		st.pending = false
	}
	m.explain[ev.Session] = st
}

func (m *Model) ensureReviewHunkVisible() {
	_, sel := m.reviewDiffContent(m.reviewDiffWidth())
	if sel < 0 {
		m.clampReviewScroll()
		return
	}
	height := m.reviewHeight()
	if sel < m.reviewScroll {
		m.reviewScroll = sel
	}
	if sel >= m.reviewScroll+height {
		m.reviewScroll = sel - height + 1
	}
	m.clampReviewScroll()
}

func (m *Model) clampReviewScroll() {
	lines, _ := m.reviewDiffContent(m.reviewDiffWidth())
	m.reviewScroll = clampScroll(m.reviewScroll, len(lines), m.reviewHeight())
}

func (m *Model) clampExplainScroll() {
	m.explainScroll = clampScroll(m.explainScroll, len(m.reviewExplainContent(m.reviewExplainWidth())), m.reviewHeight())
}

func (m Model) reviewPage() int {
	page := m.reviewHeight() - 2
	if page < 1 {
		page = 1
	}
	return page
}

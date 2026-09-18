package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dextermb/claude-multiplexer/internal/git"
)

// reviewSide is the pane the focus sits in while the review screen is open: the
// diff, the explanation, or the prompt for a follow-up. The explanation is the
// session output pane, so it carries the block cursor. See docs/tui/review.md.
type reviewSide int

const (
	reviewDiff reviewSide = iota
	reviewExplain
	reviewPrompt
)

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
	m.reviewFocus = reviewDiff
	m.reviewSidebar = !m.sidebarHidden
	m.sidebarHidden = true
	m.focus = focusReview
	m.prompt.Blur()
	m.status = "review — j/k hunk · }/{ file · e explain · E file · tab pane · esc close"
	m.output.Width = m.outputWidth()
	m.output.Height = m.outputHeight()
	m.rebuildOutput()
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
		return m.reviewExplainKey(msg)
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

// reviewExplainKey drives the explanation, which is the session output pane. It
// scrolls the pane and moves the block cursor to open a capped block. See
// docs/tui/output.md and docs/tui/review.md.
func (m Model) reviewExplainKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", " ":
		if m.blockCursor >= 0 {
			m.toggleBlock(m.blockCursor)
		}
	case "]":
		m.moveBlockCursor(1)
	case "[":
		m.moveBlockCursor(-1)
	case "j", "down":
		m.output.ScrollDown(1)
	case "k", "up":
		m.output.ScrollUp(1)
	case "u", "ctrl+u":
		m.output.HalfPageUp()
	case "d", "ctrl+d":
		m.output.HalfPageDown()
	case "pgup":
		m.output.PageUp()
	case "pgdown":
		m.output.PageDown()
	case "g", "home":
		m.output.GotoTop()
	case "G", "end":
		m.output.GotoBottom()
	}
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
		return m.reviewSend(text)
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
	return m.reviewSend(hunkExplainPrompt(path, hunks[m.reviewHunk]))
}

// reviewExplainFile is the E action. It asks the live session to explain the
// whole selected file.
func (m Model) reviewExplainFile() (tea.Model, tea.Cmd) {
	entries := m.diffEntries()
	if m.reviewFile < 0 || m.reviewFile >= len(entries) {
		return m, nil
	}
	return m.reviewSend(fileExplainPrompt(entries[m.reviewFile].file.Path))
}

// hunkExplainPrompt names the hunk as path:start-end and asks the session to
// read the file for the context, so the prompt carries no diff text. See
// docs/tui/review.md.
func hunkExplainPrompt(path string, h git.Hunk) string {
	loc := fmt.Sprintf("%s:%d-%d", path, h.NewStart, h.NewEnd())
	return fmt.Sprintf("Explain the change at %s. Keep the explanation short. "+
		"The change is against origin/HEAD. Read the file, or run `git diff origin/HEAD -- %s`, "+
		"for the surrounding context.", loc, path)
}

// fileExplainPrompt names the whole file, with no line range.
func fileExplainPrompt(path string) string {
	return fmt.Sprintf("Explain the change to %s. Keep the explanation short. "+
		"The change is against origin/HEAD. Read the file, or run `git diff origin/HEAD -- %s`, "+
		"for the surrounding context.", path, path)
}

// reviewSend sends the prompt to the live session, and moves the explanation to
// the newest line so the reply shows. A busy session queues the prompt, the same
// as any prompt. See docs/tui/review.md.
func (m Model) reviewSend(prompt string) (tea.Model, tea.Cmd) {
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
	next, cmd := m.dispatch(prompt)
	model := next.(Model)
	model.output.GotoBottom()
	return model, cmd
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

func (m Model) reviewPage() int {
	page := m.reviewHeight() - 2
	if page < 1 {
		page = 1
	}
	return page
}

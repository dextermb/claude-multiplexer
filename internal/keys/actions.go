// Package keys resolves key presses to actions, from the built-in defaults
// overlaid with the user bindings. See docs/config/keybindings.md.
package keys

import "strings"

// Context is where a key press is read: a two-key target, or one pane.
type Context string

// Action is a stable identity for one thing the interface does, as
// "<context>.<name>".
type Action string

const (
	CtxTarget     Context = "targets"
	CtxSession    Context = "session"
	CtxList       Context = "list"
	CtxOutput     Context = "output"
	CtxDiff       Context = "diff"
	CtxGlobal     Context = "global"
	CtxOutputPane Context = "outputPane"
	CtxSidebar    Context = "sidebar"
	CtxPrompt     Context = "prompt"
	CtxTask       Context = "task"
	CtxDiffPane   Context = "diffPane"
	CtxReview     Context = "review"
)

const (
	TargetSession Action = "targets.session"
	TargetList    Action = "targets.list"
	TargetOutput  Action = "targets.output"
	TargetDiff    Action = "targets.diff"

	SessionNew        Action = "session.new"
	SessionPresets    Action = "session.presets"
	SessionResume     Action = "session.resume"
	SessionRename     Action = "session.rename"
	SessionArchive    Action = "session.archive"
	SessionArchiveAll Action = "session.archiveAll"
	SessionStop       Action = "session.stop"
	SessionJobs       Action = "session.jobs"
	SessionFocusTasks Action = "session.focusTasks"
	SessionFiles      Action = "session.files"
	SessionDiff       Action = "session.diff"
	SessionReview     Action = "session.review"
	SessionEditor     Action = "session.editor"
	SessionModel      Action = "session.model"
	SessionEffort     Action = "session.effort"
	SessionMode       Action = "session.mode"
	SessionControl    Action = "session.control"
	SessionClearHold  Action = "session.clearHold"

	ListFold       Action = "list.fold"
	ListFoldOthers Action = "list.foldOthers"
	ListUnfold     Action = "list.unfold"
	ListArchived   Action = "list.archived"
	ListSearch     Action = "list.search"
	ListSidebar    Action = "list.sidebar"
	ListCollapse   Action = "list.collapse"
	ListExpand     Action = "list.expand"

	OutputMarkdown Action = "output.markdown"
	OutputLayouts  Action = "output.layouts"
	OutputAge      Action = "output.age"

	DiffWider    Action = "diff.wider"
	DiffNarrower Action = "diff.narrower"
	DiffHalf     Action = "diff.half"
	DiffNumbers  Action = "diff.numbers"
	DiffPr       Action = "diff.pr"
	DiffAllPrs   Action = "diff.allPrs"

	GlobalNewSession  Action = "global.newSession"
	GlobalPresets     Action = "global.presets"
	GlobalToggleMouse Action = "global.toggleMouse"
	GlobalQuit        Action = "global.quit"
	GlobalFocusNext   Action = "global.focusNext"
	GlobalPageUp      Action = "global.pageUp"
	GlobalPageDown    Action = "global.pageDown"

	OutputPaneOpenBlock   Action = "outputPane.openBlock"
	OutputPaneToggleBlock Action = "outputPane.toggleBlock"
	OutputPaneBlockNext   Action = "outputPane.blockNext"
	OutputPaneBlockPrev   Action = "outputPane.blockPrev"
	OutputPaneToPrompt    Action = "outputPane.toPrompt"
	OutputPaneUp          Action = "outputPane.up"
	OutputPaneDown        Action = "outputPane.down"
	OutputPaneHalfUp      Action = "outputPane.halfUp"
	OutputPaneHalfDown    Action = "outputPane.halfDown"
	OutputPaneTop         Action = "outputPane.top"
	OutputPaneBottom      Action = "outputPane.bottom"

	SidebarUp    Action = "sidebar.up"
	SidebarDown  Action = "sidebar.down"
	SidebarEnter Action = "sidebar.enter"

	PromptSend        Action = "prompt.send"
	PromptNewline     Action = "prompt.newline"
	PromptUnqueueLast Action = "prompt.unqueueLast"

	TaskUp        Action = "task.up"
	TaskDown      Action = "task.down"
	TaskHalfUp    Action = "task.halfUp"
	TaskHalfDown  Action = "task.halfDown"
	TaskPageUp    Action = "task.pageUp"
	TaskPageDown  Action = "task.pageDown"
	TaskTop       Action = "task.top"
	TaskBottom    Action = "task.bottom"
	TaskFocusNext Action = "task.focusNext"

	DiffPaneUp        Action = "diffPane.up"
	DiffPaneDown      Action = "diffPane.down"
	DiffPaneLeft      Action = "diffPane.left"
	DiffPaneRight     Action = "diffPane.right"
	DiffPaneLineUp    Action = "diffPane.lineUp"
	DiffPaneLineDown  Action = "diffPane.lineDown"
	DiffPanePageUp    Action = "diffPane.pageUp"
	DiffPanePageDown  Action = "diffPane.pageDown"
	DiffPaneTop       Action = "diffPane.top"
	DiffPaneBottom    Action = "diffPane.bottom"
	DiffPaneJumpDown  Action = "diffPane.jumpDown"
	DiffPaneJumpUp    Action = "diffPane.jumpUp"
	DiffPaneToggle    Action = "diffPane.toggle"
	DiffPaneFocusNext Action = "diffPane.focusNext"

	ReviewHunkNext    Action = "review.hunkNext"
	ReviewHunkPrev    Action = "review.hunkPrev"
	ReviewFileNext    Action = "review.fileNext"
	ReviewFilePrev    Action = "review.filePrev"
	ReviewTop         Action = "review.top"
	ReviewBottom      Action = "review.bottom"
	ReviewPageUp      Action = "review.pageUp"
	ReviewPageDown    Action = "review.pageDown"
	ReviewNumbers     Action = "review.numbers"
	ReviewExplainHunk Action = "review.explainHunk"
	ReviewExplainFile Action = "review.explainFile"
	ReviewFocusNext   Action = "review.focusNext"
)

// Def is one action in the catalogue: its context, its default keys, and a
// short description for a displacement warning.
type Def struct {
	Action  Action
	Context Context
	Keys    []string
	Desc    string
}

// Defaults is the built-in catalogue, in a stable order. Every field of
// config.Keybindings maps to exactly one entry here; a test holds the parity.
var Defaults = []Def{
	{TargetSession, CtxTarget, []string{"s", "ctrl+s"}, "the session target"},
	{TargetList, CtxTarget, []string{"l", "ctrl+l"}, "the list target"},
	{TargetOutput, CtxTarget, []string{"o", "ctrl+o"}, "the output target"},
	{TargetDiff, CtxTarget, []string{"d"}, "the diff target"},

	{SessionNew, CtxSession, []string{"c"}, "session: start a new session"},
	{SessionPresets, CtxSession, []string{"t"}, "session: open the preset prompts"},
	{SessionResume, CtxSession, []string{"r"}, "session: resume the selected session"},
	{SessionRename, CtxSession, []string{"n"}, "session: rename the selected session"},
	{SessionArchive, CtxSession, []string{"a"}, "session: archive the selected session"},
	{SessionArchiveAll, CtxSession, []string{"A"}, "session: archive every session attached to the control session"},
	{SessionStop, CtxSession, []string{"x"}, "session: stop the selected session"},
	{SessionJobs, CtxSession, []string{"j"}, "session: show the background jobs"},
	{SessionFocusTasks, CtxSession, []string{"k"}, "session: focus the task and job panel"},
	{SessionFiles, CtxSession, []string{"f"}, "session: open the working directories in the file manager"},
	{SessionDiff, CtxSession, []string{"d"}, "session: show the working-tree diff"},
	{SessionReview, CtxSession, []string{"R"}, "session: open the code review screen"},
	{SessionEditor, CtxSession, []string{"E"}, "session: open the working directories in the editor"},
	{SessionModel, CtxSession, []string{"m"}, "session: change the model"},
	{SessionEffort, CtxSession, []string{"e"}, "session: change the effort"},
	{SessionMode, CtxSession, []string{"p"}, "session: change the permission mode"},
	{SessionControl, CtxSession, []string{"C"}, "session: turn control on or off"},
	{SessionClearHold, CtxSession, []string{"h"}, "session: clear the context hold"},

	{ListFold, CtxList, []string{"f"}, "list: fold or unfold the group"},
	{ListFoldOthers, CtxList, []string{"F"}, "list: fold every group but this one"},
	{ListUnfold, CtxList, []string{"u"}, "list: unfold every group"},
	{ListArchived, CtxList, []string{"a"}, "list: show or hide the archived sessions"},
	{ListSearch, CtxList, []string{"s"}, "list: search the sessions"},
	{ListSidebar, CtxList, []string{"t"}, "list: hide or show the sidebar"},
	{ListCollapse, CtxList, []string{"c"}, "list: hide the sidebar"},
	{ListExpand, CtxList, []string{"e"}, "list: show the sidebar again"},

	{OutputMarkdown, CtxOutput, []string{"m"}, "output: switch markdown and raw text"},
	{OutputLayouts, CtxOutput, []string{"l"}, "output: open the layouts"},
	{OutputAge, CtxOutput, []string{"a"}, "output: show or hide the age of each block"},

	{DiffWider, CtxDiff, []string{"+"}, "diff: widen the diff panel"},
	{DiffNarrower, CtxDiff, []string{"-"}, "diff: narrow the diff panel"},
	{DiffHalf, CtxDiff, []string{"/"}, "diff: half the screen, or the set width"},
	{DiffNumbers, CtxDiff, []string{"n"}, "diff: show or hide the line numbers"},
	{DiffPr, CtxDiff, []string{"p"}, "diff: open the pull request of the current file"},
	{DiffAllPrs, CtxDiff, []string{"P"}, "diff: open the pull request of every code base"},

	{GlobalNewSession, CtxGlobal, []string{"n", "ctrl+n"}, "start a new session"},
	{GlobalPresets, CtxGlobal, []string{"t", "ctrl+p"}, "open the preset prompts"},
	{GlobalToggleMouse, CtxGlobal, []string{"ctrl+t"}, "turn the mouse on or off"},
	{GlobalQuit, CtxGlobal, []string{"q"}, "stop every session, and quit"},
	{GlobalFocusNext, CtxGlobal, []string{"tab"}, "move to the next pane"},
	{GlobalPageUp, CtxGlobal, []string{"pgup"}, "scroll the output up a page"},
	{GlobalPageDown, CtxGlobal, []string{"pgdown"}, "scroll the output down a page"},

	{OutputPaneOpenBlock, CtxOutputPane, []string{"enter"}, "output: open the block, or move to the prompt"},
	{OutputPaneToggleBlock, CtxOutputPane, []string{" "}, "output: open or close the block"},
	{OutputPaneBlockNext, CtxOutputPane, []string{"]"}, "output: move to the next capped block"},
	{OutputPaneBlockPrev, CtxOutputPane, []string{"["}, "output: move to the previous capped block"},
	{OutputPaneToPrompt, CtxOutputPane, []string{"i"}, "output: move to the prompt"},
	{OutputPaneUp, CtxOutputPane, []string{"up", "k"}, "output: scroll up one line"},
	{OutputPaneDown, CtxOutputPane, []string{"down", "j"}, "output: scroll down one line"},
	{OutputPaneHalfUp, CtxOutputPane, []string{"u", "ctrl+u"}, "output: scroll up half a pane"},
	{OutputPaneHalfDown, CtxOutputPane, []string{"d", "ctrl+d"}, "output: scroll down half a pane"},
	{OutputPaneTop, CtxOutputPane, []string{"g", "home"}, "output: go to the top"},
	{OutputPaneBottom, CtxOutputPane, []string{"G", "end"}, "output: go to the bottom"},

	{SidebarUp, CtxSidebar, []string{"up", "k"}, "list: move up"},
	{SidebarDown, CtxSidebar, []string{"down", "j"}, "list: move down"},
	{SidebarEnter, CtxSidebar, []string{"enter", "i"}, "list: type into a session, or resume it"},

	{PromptSend, CtxPrompt, []string{"enter"}, "prompt: send what you typed"},
	{PromptNewline, CtxPrompt, []string{"ctrl+j"}, "prompt: add a new line"},
	{PromptUnqueueLast, CtxPrompt, []string{"backspace"}, "prompt: remove the newest queued prompt"},

	{TaskUp, CtxTask, []string{"up", "k"}, "task panel: scroll up one line"},
	{TaskDown, CtxTask, []string{"down", "j"}, "task panel: scroll down one line"},
	{TaskHalfUp, CtxTask, []string{"u", "ctrl+u"}, "task panel: scroll up half a panel"},
	{TaskHalfDown, CtxTask, []string{"d", "ctrl+d"}, "task panel: scroll down half a panel"},
	{TaskPageUp, CtxTask, []string{"pgup"}, "task panel: scroll up a page"},
	{TaskPageDown, CtxTask, []string{"pgdown"}, "task panel: scroll down a page"},
	{TaskTop, CtxTask, []string{"g", "home"}, "task panel: go to the top"},
	{TaskBottom, CtxTask, []string{"G", "end"}, "task panel: go to the bottom"},
	{TaskFocusNext, CtxTask, []string{"tab"}, "task panel: move to the next pane"},

	{DiffPaneUp, CtxDiffPane, []string{"k"}, "diff panel: step up, or up a row in the grid"},
	{DiffPaneDown, CtxDiffPane, []string{"j"}, "diff panel: step down, or down a row in the grid"},
	{DiffPaneLeft, CtxDiffPane, []string{"h"}, "diff panel: move left in the grid"},
	{DiffPaneRight, CtxDiffPane, []string{"l"}, "diff panel: move right in the grid"},
	{DiffPaneLineUp, CtxDiffPane, []string{"up"}, "diff panel: scroll up one line"},
	{DiffPaneLineDown, CtxDiffPane, []string{"down"}, "diff panel: scroll down one line"},
	{DiffPanePageUp, CtxDiffPane, []string{"pgup"}, "diff panel: scroll up a page"},
	{DiffPanePageDown, CtxDiffPane, []string{"pgdown"}, "diff panel: scroll down a page"},
	{DiffPaneTop, CtxDiffPane, []string{"g"}, "diff panel: go to the top"},
	{DiffPaneBottom, CtxDiffPane, []string{"G"}, "diff panel: go to the bottom"},
	{DiffPaneJumpDown, CtxDiffPane, []string{"}", "shift+]"}, "diff panel: jump to the next empty line"},
	{DiffPaneJumpUp, CtxDiffPane, []string{"{", "shift+["}, "diff panel: jump to the previous empty line"},
	{DiffPaneToggle, CtxDiffPane, []string{"enter", " "}, "diff panel: open or close the file"},
	{DiffPaneFocusNext, CtxDiffPane, []string{"tab"}, "diff panel: move to the next pane"},

	{ReviewHunkNext, CtxReview, []string{"j", "down"}, "review: step to the next hunk"},
	{ReviewHunkPrev, CtxReview, []string{"k", "up"}, "review: step to the previous hunk"},
	{ReviewFileNext, CtxReview, []string{"}", "shift+]"}, "review: jump to the next file"},
	{ReviewFilePrev, CtxReview, []string{"{", "shift+["}, "review: jump to the previous file"},
	{ReviewTop, CtxReview, []string{"g", "home"}, "review: go to the first file"},
	{ReviewBottom, CtxReview, []string{"G", "end"}, "review: go to the last file"},
	{ReviewPageUp, CtxReview, []string{"pgup"}, "review: scroll up a page"},
	{ReviewPageDown, CtxReview, []string{"pgdown"}, "review: scroll down a page"},
	{ReviewNumbers, CtxReview, []string{"n"}, "review: show or hide the line numbers"},
	{ReviewExplainHunk, CtxReview, []string{"e"}, "review: explain the selected hunk"},
	{ReviewExplainFile, CtxReview, []string{"E"}, "review: explain the whole file"},
	{ReviewFocusNext, CtxReview, []string{"tab"}, "review: move between the diff, explanation, and prompt"},
}

// TargetContext maps a target action to the context of its second key.
func TargetContext(a Action) (Context, bool) {
	switch a {
	case TargetSession:
		return CtxSession, true
	case TargetList:
		return CtxList, true
	case TargetOutput:
		return CtxOutput, true
	case TargetDiff:
		return CtxDiff, true
	}
	return "", false
}

// TargetForContext maps a chord context back to its target action.
func TargetForContext(ctx Context) (Action, bool) {
	switch ctx {
	case CtxSession:
		return TargetSession, true
	case CtxList:
		return TargetList, true
	case CtxOutput:
		return TargetOutput, true
	case CtxDiff:
		return TargetDiff, true
	}
	return "", false
}

// ContextOf is the context part of an action id.
func ContextOf(a Action) Context {
	if i := strings.IndexByte(string(a), '.'); i >= 0 {
		return Context(a[:i])
	}
	return ""
}

// InContext lists the catalogue actions of a context, in catalogue order.
func InContext(ctx Context) []Action {
	var out []Action
	for _, d := range Defaults {
		if d.Context == ctx {
			out = append(out, d.Action)
		}
	}
	return out
}

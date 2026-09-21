package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dextermb/claude-multiplexer/internal/keys"
)

// binding is one help row. When action is set, the keys column comes from the
// keymap, so a rebind follows; otherwise keys is the literal text. See
// docs/config/keybindings.md.
type binding struct {
	group  string
	keys   string
	action keys.Action
	what   string
}

var bindings = []binding{
	{group: "Quick keys", action: keys.GlobalNewSession, what: "Start a new session"},
	{group: "Quick keys", action: keys.GlobalPresets, what: "Open the preset prompts"},

	{group: "The targets", action: keys.TargetSession, what: "Act on the selected session"},
	{group: "The targets", action: keys.TargetList, what: "Act on the list"},
	{group: "The targets", action: keys.TargetOutput, what: "Act on the output pane"},

	{group: "The session (s)", action: keys.SessionNew, what: "Start a new session"},
	{group: "The session (s)", action: keys.SessionPresets, what: "Open the preset prompts"},
	{group: "The session (s)", action: keys.SessionResume, what: "Resume the selected session"},
	{group: "The session (s)", action: keys.SessionRename, what: "Rename the selected session"},
	{group: "The session (s)", action: keys.SessionArchive, what: "Archive the selected session, or bring it back"},
	{group: "The session (s)", action: keys.SessionStop, what: "Stop the selected session, after a confirmation"},
	{group: "The session (s)", action: keys.SessionJobs, what: "Show the background jobs of the selected session"},
	{group: "The session (s)", action: keys.SessionFocusTasks, what: "Move the focus to the task and job panel"},
	{group: "The session (s)", action: keys.SessionFiles, what: "Open the working directories in the file manager"},
	{group: "The session (s)", action: keys.SessionDiff, what: "Show the working-tree diff of the selected session"},
	{group: "The session (s)", action: keys.SessionReview, what: "Open the code review screen for the selected session"},
	{group: "The session (s)", action: keys.SessionEditor, what: "Open the working directories in the editor"},
	{group: "The session (s)", action: keys.SessionModel, what: "Change the model"},
	{group: "The session (s)", action: keys.SessionEffort, what: "Change the effort (the thinking budget)"},
	{group: "The session (s)", action: keys.SessionMode, what: "Change the permission mode"},
	{group: "The session (s)", action: keys.SessionControl, what: "Turn control on or off for the session"},
	{group: "The session (s)", action: keys.SessionClearHold, what: "Clear the context hold, so the session takes a prompt again"},

	{group: "The list (l)", action: keys.ListFold, what: "Fold or unfold the group of the selected session"},
	{group: "The list (l)", action: keys.ListFoldOthers, what: "Fold every group but this one"},
	{group: "The list (l)", action: keys.ListUnfold, what: "Unfold every group"},
	{group: "The list (l)", action: keys.ListArchived, what: "Show or hide the archived sessions"},
	{group: "The list (l)", action: keys.ListSearch, what: "Search the sessions in the list"},
	{group: "The list (l)", action: keys.ListSidebar, what: "Hide or show the sidebar"},
	{group: "The list (l)", action: keys.ListCollapse, what: "Hide the sidebar, for more pane width"},
	{group: "The list (l)", action: keys.ListExpand, what: "Show the sidebar again"},

	{group: "The output pane (o)", action: keys.OutputMarkdown, what: "Switch between rendered markdown and raw text"},
	{group: "The output pane (o)", action: keys.OutputLayouts, what: "Open the layouts, to switch between them"},
	{group: "The output pane (o)", action: keys.OutputAge, what: "Show or hide the age of each block at the right edge"},

	{group: "The diff panel (d)", action: keys.DiffWider, what: "Widen the diff panel"},
	{group: "The diff panel (d)", action: keys.DiffNarrower, what: "Narrow the diff panel"},
	{group: "The diff panel (d)", action: keys.DiffHalf, what: "Show the panel at half the screen, or the set width"},
	{group: "The diff panel (d)", action: keys.DiffNumbers, what: "Show or hide the line numbers"},
	{group: "The diff panel (d)", action: keys.DiffPr, what: "Open the pull request of the current file's code base"},
	{group: "The diff panel (d)", action: keys.DiffAllPrs, what: "Open the pull request of every code base"},
	{group: "The diff panel (d)", action: keys.DiffPaneDown, what: "Step through an open diff, or move in the file grid"},
	{group: "The diff panel (d)", action: keys.DiffPaneJumpDown, what: "Jump to the next empty line of an open diff"},

	{group: "The review screen (s R)", action: keys.ReviewHunkNext, what: "Step through the hunks, and roll to the next file"},
	{group: "The review screen (s R)", action: keys.ReviewFileNext, what: "Jump to the next file"},
	{group: "The review screen (s R)", action: keys.ReviewExplainHunk, what: "Explain the selected hunk"},
	{group: "The review screen (s R)", action: keys.ReviewExplainFile, what: "Explain the whole file"},
	{group: "The review screen (s R)", action: keys.ReviewNumbers, what: "Show or hide the line numbers of the diff"},
	{group: "The review screen (s R)", action: keys.ReviewFocusNext, what: "Move between the diff, the explanation, and the prompt"},

	{group: "The task panel", action: keys.TaskDown, what: "Scroll the task and job panel"},
	{group: "The task panel", action: keys.TaskHalfDown, what: "Scroll half a panel"},
	{group: "The task panel", action: keys.TaskPageDown, what: "Scroll a whole panel"},
	{group: "The task panel", action: keys.TaskTop, what: "Go to the top; the paired key goes to the bottom"},

	{group: "Moving", action: keys.SidebarDown, what: "Move through the list"},
	{group: "Moving", action: keys.GlobalFocusNext, what: "Move to the next pane"},
	{group: "Moving", action: keys.SidebarEnter, what: "Type into a session, or resume one that is not running"},
	{group: "Moving", keys: "esc", what: "Leave the prompt or the output, close a dialog, or cancel a sequence"},

	{group: "The prompt", action: keys.PromptSend, what: "Send what you typed"},
	{group: "The prompt", action: keys.PromptNewline, what: "Add a new line inside the prompt"},
	{group: "The prompt", action: keys.PromptUnqueueLast, what: "On an empty prompt, remove the newest queued prompt"},
	{group: "The prompt", keys: "tab", what: "Complete a /preset name, or move on"},
	{group: "The prompt", keys: "up  down", what: "Recall an older or a newer prompt you sent"},
	{group: "The prompt", keys: "shift+tab", what: "Walk the paths that match an @ word"},

	{group: "The output", action: keys.OutputPaneHalfDown, what: "Scroll half a pane"},
	{group: "The output", action: keys.OutputPaneBlockNext, what: "Move the cursor between the capped blocks"},
	{group: "The output", action: keys.OutputPaneOpenBlock, what: "Open or close the block under the cursor"},
	{group: "The output", action: keys.OutputPaneToPrompt, what: "Move to the prompt"},

	{group: "The list and the output", keys: "?", what: "Show this list"},
	{group: "The list and the output", action: keys.GlobalQuit, what: "Stop every session, and quit"},

	{group: "Everywhere", action: keys.GlobalToggleMouse, what: "Turn the mouse on or off"},
	{group: "Everywhere", keys: "ctrl+c", what: "Clear the prompt. Press it again to quit"},
}

// briefWord is the status-bar word for an action, next to its key.
var briefWord = map[keys.Action]string{
	keys.GlobalNewSession: "new",
	keys.GlobalPresets:    "preset",
	keys.GlobalQuit:       "quit",
	keys.TargetSession:    "session",
	keys.TargetList:       "list",
	keys.TargetOutput:     "output",

	keys.SessionNew: "new", keys.SessionPresets: "preset", keys.SessionResume: "resume",
	keys.SessionRename: "name", keys.SessionArchive: "archive", keys.SessionStop: "stop",
	keys.SessionJobs: "jobs", keys.SessionFocusTasks: "tasks", keys.SessionFiles: "folder",
	keys.SessionDiff: "diff", keys.SessionReview: "review", keys.SessionEditor: "editor",
	keys.SessionModel: "model", keys.SessionEffort: "effort", keys.SessionMode: "mode",
	keys.SessionControl: "control", keys.SessionClearHold: "hold",

	keys.ListFold: "fold", keys.ListFoldOthers: "others", keys.ListUnfold: "unfold",
	keys.ListArchived: "archived", keys.ListSearch: "search", keys.ListSidebar: "sidebar",
	keys.ListCollapse: "hide", keys.ListExpand: "show",

	keys.OutputMarkdown: "markdown", keys.OutputLayouts: "layouts", keys.OutputAge: "age",

	keys.DiffWider: "wider", keys.DiffNarrower: "narrower", keys.DiffHalf: "half",
	keys.DiffNumbers: "numbers", keys.DiffPr: "pr", keys.DiffAllPrs: "all prs",
}

// firstKey is the first key an action answers to.
func firstKey(km keys.Keymap, a keys.Action) string {
	if ks := km.Keys(a); len(ks) > 0 {
		return ks[0]
	}
	return ""
}

// brief is the status-bar hint for an action: its key and its word.
func brief(km keys.Keymap, a keys.Action) string {
	word, ok := briefWord[a]
	if !ok {
		return ""
	}
	key := firstKey(km, a)
	if key == "" {
		return ""
	}
	return key + " " + word
}

// displayKeys is the keys column of a help row: a chord shows its target and its
// action key, every other action shows its own keys.
func displayKeys(km keys.Keymap, a keys.Action) string {
	ctx := keys.ContextOf(a)
	if target, ok := keys.TargetForContext(ctx); ok {
		return firstKey(km, target) + " " + strings.Join(km.Keys(a), " ")
	}
	return strings.Join(km.Keys(a), "  ")
}

// statusHints lists the keys that work on their own, for the status bar.
func (m Model) statusHints() string {
	order := []keys.Action{
		keys.GlobalNewSession, keys.GlobalPresets,
		keys.TargetSession, keys.TargetList, keys.TargetOutput,
	}
	var out []string
	for _, a := range order {
		if b := brief(m.keys, a); b != "" {
			out = append(out, b)
		}
	}
	out = append(out, "? keys")
	if b := brief(m.keys, keys.GlobalQuit); b != "" {
		out = append(out, b)
	}
	return strings.Join(out, " · ")
}

// sequenceHints lists the action keys of a target, for the status bar.
func (m Model) sequenceHints(target keys.Action) string {
	ctx, ok := keys.TargetContext(target)
	if !ok {
		return ""
	}
	var out []string
	for _, a := range keys.InContext(ctx) {
		if b := brief(m.keys, a); b != "" {
			out = append(out, b)
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

func (h *help) rows(km keys.Keymap, width int) []string {
	needle := strings.ToLower(strings.TrimSpace(h.filter.Value()))
	var (
		out   []string
		group string
	)
	for _, item := range bindings {
		shown := item.keys
		if item.action != "" {
			shown = displayKeys(km, item.action)
		}
		if needle != "" &&
			!strings.Contains(strings.ToLower(shown), needle) &&
			!strings.Contains(strings.ToLower(item.what), needle) &&
			!strings.Contains(strings.ToLower(item.group), needle) {
			continue
		}
		if item.group != group {
			group = item.group
			out = append(out, titleStyle.Render(group))
		}
		out = append(out, "  "+fieldLabelStyle.Render(pad(shown, 17))+truncate(item.what, width-21))
	}
	return out
}

func (h *help) View(km keys.Keymap, width, height int) string {
	inner := modalInner(width)
	window := height - 10
	if window < 3 {
		window = 3
	}

	rows := h.rows(km, inner)
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

package config

// Keybindings rebinds the keys of the interface. Each field names the keys of
// one action; an empty field keeps the built-in default. The keys ?, esc, and
// ctrl+c are reserved and cannot be rebound. See docs/config/keybindings.md.
type Keybindings struct {
	Targets    *TargetKeys     `json:"targets,omitempty"`
	Session    *SessionKeys    `json:"session,omitempty"`
	List       *ListKeys       `json:"list,omitempty"`
	Output     *OutputKeys     `json:"output,omitempty"`
	Diff       *DiffKeys       `json:"diff,omitempty"`
	Global     *GlobalKeys     `json:"global,omitempty"`
	OutputPane *OutputPaneKeys `json:"outputPane,omitempty"`
	Sidebar    *SidebarKeys    `json:"sidebar,omitempty"`
	Prompt     *PromptKeys     `json:"prompt,omitempty"`
	Task       *TaskKeys       `json:"task,omitempty"`
	DiffPane   *DiffPaneKeys   `json:"diffPane,omitempty"`
	Review     *ReviewKeys     `json:"review,omitempty"`
}

// TargetKeys are the first keys of a two-key sequence.
type TargetKeys struct {
	Session []string `json:"session,omitempty"`
	List    []string `json:"list,omitempty"`
	Output  []string `json:"output,omitempty"`
	Diff    []string `json:"diff,omitempty"`
}

// SessionKeys are the second keys after the session target.
type SessionKeys struct {
	New        []string `json:"new,omitempty"`
	Presets    []string `json:"presets,omitempty"`
	Resume     []string `json:"resume,omitempty"`
	Rename     []string `json:"rename,omitempty"`
	Archive    []string `json:"archive,omitempty"`
	Stop       []string `json:"stop,omitempty"`
	Jobs       []string `json:"jobs,omitempty"`
	FocusTasks []string `json:"focusTasks,omitempty"`
	Files      []string `json:"files,omitempty"`
	Diff       []string `json:"diff,omitempty"`
	Review     []string `json:"review,omitempty"`
	Editor     []string `json:"editor,omitempty"`
	Model      []string `json:"model,omitempty"`
	Effort     []string `json:"effort,omitempty"`
	Mode       []string `json:"mode,omitempty"`
	Control    []string `json:"control,omitempty"`
	ClearHold  []string `json:"clearHold,omitempty"`
}

// ListKeys are the second keys after the list target.
type ListKeys struct {
	Fold       []string `json:"fold,omitempty"`
	FoldOthers []string `json:"foldOthers,omitempty"`
	Unfold     []string `json:"unfold,omitempty"`
	Archived   []string `json:"archived,omitempty"`
	Search     []string `json:"search,omitempty"`
	Sidebar    []string `json:"sidebar,omitempty"`
	Collapse   []string `json:"collapse,omitempty"`
	Expand     []string `json:"expand,omitempty"`
}

// OutputKeys are the second keys after the output target.
type OutputKeys struct {
	Markdown []string `json:"markdown,omitempty"`
	Layouts  []string `json:"layouts,omitempty"`
	Age      []string `json:"age,omitempty"`
}

// DiffKeys are the second keys after the diff target.
type DiffKeys struct {
	Wider    []string `json:"wider,omitempty"`
	Narrower []string `json:"narrower,omitempty"`
	Half     []string `json:"half,omitempty"`
	Numbers  []string `json:"numbers,omitempty"`
	Pr       []string `json:"pr,omitempty"`
	AllPrs   []string `json:"allPrs,omitempty"`
}

// GlobalKeys are the keys that work on their own.
type GlobalKeys struct {
	NewSession  []string `json:"newSession,omitempty"`
	Presets     []string `json:"presets,omitempty"`
	ToggleMouse []string `json:"toggleMouse,omitempty"`
	Quit        []string `json:"quit,omitempty"`
	FocusNext   []string `json:"focusNext,omitempty"`
	PageUp      []string `json:"pageUp,omitempty"`
	PageDown    []string `json:"pageDown,omitempty"`
}

// OutputPaneKeys move and scroll the output pane.
type OutputPaneKeys struct {
	OpenBlock   []string `json:"openBlock,omitempty"`
	ToggleBlock []string `json:"toggleBlock,omitempty"`
	BlockNext   []string `json:"blockNext,omitempty"`
	BlockPrev   []string `json:"blockPrev,omitempty"`
	ToPrompt    []string `json:"toPrompt,omitempty"`
	Up          []string `json:"up,omitempty"`
	Down        []string `json:"down,omitempty"`
	HalfUp      []string `json:"halfUp,omitempty"`
	HalfDown    []string `json:"halfDown,omitempty"`
	Top         []string `json:"top,omitempty"`
	Bottom      []string `json:"bottom,omitempty"`
}

// SidebarKeys move through the list.
type SidebarKeys struct {
	Up    []string `json:"up,omitempty"`
	Down  []string `json:"down,omitempty"`
	Enter []string `json:"enter,omitempty"`
}

// PromptKeys act on the prompt box.
type PromptKeys struct {
	Send        []string `json:"send,omitempty"`
	Newline     []string `json:"newline,omitempty"`
	UnqueueLast []string `json:"unqueueLast,omitempty"`
}

// TaskKeys scroll the task and job panel.
type TaskKeys struct {
	Up        []string `json:"up,omitempty"`
	Down      []string `json:"down,omitempty"`
	HalfUp    []string `json:"halfUp,omitempty"`
	HalfDown  []string `json:"halfDown,omitempty"`
	PageUp    []string `json:"pageUp,omitempty"`
	PageDown  []string `json:"pageDown,omitempty"`
	Top       []string `json:"top,omitempty"`
	Bottom    []string `json:"bottom,omitempty"`
	FocusNext []string `json:"focusNext,omitempty"`
}

// DiffPaneKeys move through the diff panel.
type DiffPaneKeys struct {
	Up        []string `json:"up,omitempty"`
	Down      []string `json:"down,omitempty"`
	Left      []string `json:"left,omitempty"`
	Right     []string `json:"right,omitempty"`
	LineUp    []string `json:"lineUp,omitempty"`
	LineDown  []string `json:"lineDown,omitempty"`
	PageUp    []string `json:"pageUp,omitempty"`
	PageDown  []string `json:"pageDown,omitempty"`
	Top       []string `json:"top,omitempty"`
	Bottom    []string `json:"bottom,omitempty"`
	JumpDown  []string `json:"jumpDown,omitempty"`
	JumpUp    []string `json:"jumpUp,omitempty"`
	Toggle    []string `json:"toggle,omitempty"`
	FocusNext []string `json:"focusNext,omitempty"`
}

// ReviewKeys act on the code review screen.
type ReviewKeys struct {
	HunkNext    []string `json:"hunkNext,omitempty"`
	HunkPrev    []string `json:"hunkPrev,omitempty"`
	FileNext    []string `json:"fileNext,omitempty"`
	FilePrev    []string `json:"filePrev,omitempty"`
	Top         []string `json:"top,omitempty"`
	Bottom      []string `json:"bottom,omitempty"`
	PageUp      []string `json:"pageUp,omitempty"`
	PageDown    []string `json:"pageDown,omitempty"`
	Numbers     []string `json:"numbers,omitempty"`
	ExplainHunk []string `json:"explainHunk,omitempty"`
	ExplainFile []string `json:"explainFile,omitempty"`
	FocusNext   []string `json:"focusNext,omitempty"`
}

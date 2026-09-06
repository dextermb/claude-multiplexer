package mcp

type renameIn struct {
	Title string `json:"title" jsonschema:"the new display title for this session; an empty string clears it"`
}

type listIn struct {
	LiveOnly bool `json:"live_only,omitempty" jsonschema:"true to leave out the sessions that are not running now"`
}

type messagesIn struct {
	Session string `json:"session" jsonschema:"the name of the session to read"`
	Limit   int    `json:"limit,omitempty" jsonschema:"how many messages to return, newest last; 20 by default"`
}

type sendIn struct {
	Session string `json:"session" jsonschema:"the name of the session to prompt"`
	Text    string `json:"text" jsonschema:"the prompt to queue for that session"`
}

type targetIn struct {
	Session string `json:"session" jsonschema:"the name of the session to act on"`
}

type archiveIn struct {
	Session string `json:"session" jsonschema:"the name of the session to act on"`
	Restore bool   `json:"restore,omitempty" jsonschema:"true to bring an archived session back into the list"`
}

type createIn struct {
	Path string `json:"path" jsonschema:"the directory the new session works in"`
	Name string `json:"name,omitempty" jsonschema:"an optional name for the new session"`
}

type listJobsIn struct {
	Session string `json:"session,omitempty" jsonschema:"the session to read the jobs of; empty means this session"`
}

type stopJobIn struct {
	Session string `json:"session,omitempty" jsonschema:"the session that owns the job; empty means this session"`
	Job     string `json:"job" jsonschema:"the id of the background job to stop"`
}

type setEditorIn struct {
	Editor   string `json:"editor,omitempty" jsonschema:"the command that opens a directory, such as 'code -n' or 'nvim'; leave it out to keep the editor that is set"`
	Terminal *bool  `json:"terminal,omitempty" jsonschema:"true when that editor draws in the terminal (vim, nvim, helix), false when it has its own window (code, zed); leave it out to keep what is set"`
}

type setEditorOut struct {
	OK      bool   `json:"ok"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

type unsetEditorIn struct {
	Field string `json:"field,omitempty" jsonschema:"which setting to clear: 'editor' for the editor, 'terminal' for the terminal flag, or 'both'; 'both' by default"`
}

type unsetEditorOut struct {
	OK      bool   `json:"ok"`
	Path    string `json:"path"`
	Changed bool   `json:"changed"`
	Message string `json:"message"`
}

type templatePathIn struct {
	Session string `json:"session,omitempty" jsonschema:"the session to read the template directories of; empty means this session"`
}

type setBlockCapIn struct {
	Type      string `json:"type,omitempty" jsonschema:"the block type to cap: prompt, message, tool, meta, bash, error, question_option, or question_description; empty sets the default for every other pane type"`
	Rows      *int   `json:"rows,omitempty" jsonschema:"the rows a block of this type draws before the pane caps it and offers to open the rest; 0 draws only the marker, so the human opens the block to read it"`
	Unlimited bool   `json:"unlimited,omitempty" jsonschema:"true never caps this type, so the block always draws in full; do not give rows as well"`
}

type setBlockCapOut struct {
	OK      bool   `json:"ok"`
	Path    string `json:"path"`
	Type    string `json:"type,omitempty"`
	Rows    *int   `json:"rows,omitempty"`
	Message string `json:"message"`
}

type unsetBlockCapIn struct {
	Type string `json:"type,omitempty" jsonschema:"the block type to clear: prompt, message, tool, meta, bash, error, question_option, or question_description; empty clears the default"`
}

type unsetBlockCapOut struct {
	OK      bool   `json:"ok"`
	Path    string `json:"path"`
	Changed bool   `json:"changed"`
	Message string `json:"message"`
}

type setWorkingDirIn struct {
	Path string `json:"path" jsonschema:"the directory this session works in now, such as '.worktrees/feature'; a relative path is resolved against the directory the session started in"`
}

type workingDirOut struct {
	OK      bool   `json:"ok"`
	Path    string `json:"path,omitempty"`
	Changed bool   `json:"changed"`
	Message string `json:"message"`
}

type okOut struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

type sendOut struct {
	OK      bool   `json:"ok"`
	Queued  int    `json:"queued"`
	Message string `json:"message"`
}

type listOut struct {
	Sessions []Session `json:"sessions"`
}

type messagesOut struct {
	Session  string    `json:"session"`
	Messages []Message `json:"messages"`
}

type createOut struct {
	OK      bool   `json:"ok"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

type listJobsOut struct {
	Session string `json:"session"`
	Jobs    []Job  `json:"jobs"`
}

type stopJobOut struct {
	OK      bool   `json:"ok"`
	Queued  int    `json:"queued"`
	Message string `json:"message"`
}

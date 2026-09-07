package mcp

type setConfigIn struct {
	Path  string `json:"path" jsonschema:"the settings key to set, as a dot path, such as 'editor', 'blockCap', 'blockCaps.tool', or 'layouts.wide.sidebarSize'"`
	Value any    `json:"value" jsonschema:"the JSON value to write, such as \"nvim\", 40, true, or null"`
}

type unsetConfigIn struct {
	Path string `json:"path" jsonschema:"the settings key to remove, as a dot path, such as 'blockCap' or 'layouts.wide'"`
}

type setConfigOut struct {
	OK      bool   `json:"ok"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

type unsetConfigOut struct {
	OK      bool   `json:"ok"`
	Path    string `json:"path"`
	Changed bool   `json:"changed"`
	Message string `json:"message"`
}

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

type listProjectIn struct {
	Session string `json:"session,omitempty" jsonschema:"the session to read the project of; empty means this session"`
}

type projectDirIn struct {
	Path string `json:"path" jsonschema:"a directory to add to or remove from this session's project; a relative path is resolved against the directory the session started in"`
}

type setProjectIn struct {
	Paths []string `json:"paths" jsonschema:"the whole ordered set of directories of this session's project; a relative path is resolved against the directory the session started in"`
}

type projectOut struct {
	OK      bool     `json:"ok"`
	Dirs    []string `json:"dirs"`
	Changed bool     `json:"changed"`
	Message string   `json:"message"`
}

type listLayoutsIn struct {
	Session string `json:"session,omitempty" jsonschema:"the session to read the active layout of; empty means this session"`
}

type saveLayoutIn struct {
	Name         string  `json:"name" jsonschema:"the name of the layout to create or replace"`
	PromptMin    *int    `json:"promptMin,omitempty" jsonschema:"the least rows the prompt bar draws; leave it out to keep the current value"`
	PromptMax    *int    `json:"promptMax,omitempty" jsonschema:"the most rows the prompt bar grows to; leave it out to keep the current value"`
	SidebarSize  *int    `json:"sidebarSize,omitempty" jsonschema:"the columns of the session list sidebar; leave it out to keep the current value"`
	TaskSize     *int    `json:"taskSize,omitempty" jsonschema:"the columns of the task and background job panel; leave it out to keep the current value"`
	DiffSize     *int    `json:"diffSize,omitempty" jsonschema:"the size of the diff panel: columns on left or right, rows on top or bottom; leave it out to keep the current value"`
	DiffPosition *string `json:"diffPosition,omitempty" jsonschema:"the side the diff panel draws on: left, right, top, or bottom; leave it out to keep the current value"`
}

type saveLayoutOut struct {
	OK      bool   `json:"ok"`
	Path    string `json:"path"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

type deleteLayoutIn struct {
	Name string `json:"name" jsonschema:"the name of the layout to remove"`
}

type deleteLayoutOut struct {
	OK      bool   `json:"ok"`
	Path    string `json:"path"`
	Changed bool   `json:"changed"`
	Message string `json:"message"`
}

type setLayoutIn struct {
	Name  string `json:"name" jsonschema:"the name of the layout to activate"`
	Scope string `json:"scope,omitempty" jsonschema:"'session' sets this session, 'all' sets the global default; 'session' by default"`
}

type setLayoutOut struct {
	OK      bool   `json:"ok"`
	Path    string `json:"path,omitempty"`
	Scope   string `json:"scope"`
	Message string `json:"message"`
}

type unsetLayoutIn struct {
	Scope string `json:"scope,omitempty" jsonschema:"'session' clears this session, 'all' clears the global default; 'session' by default"`
}

type unsetLayoutOut struct {
	OK      bool   `json:"ok"`
	Path    string `json:"path,omitempty"`
	Scope   string `json:"scope"`
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

type createScheduleIn struct {
	Cron           string `json:"cron" jsonschema:"a 5-field cron expression in local time, such as '*/5 * * * *' for every 5 minutes or '0 9 * * 1-5' for 09:00 on weekdays"`
	Dir            string `json:"dir" jsonschema:"the directory the run works in; a relative path is resolved against the directory the multiplexer runs in"`
	Prompt         string `json:"prompt" jsonschema:"the prompt the schedule sends on each run"`
	Name           string `json:"name,omitempty" jsonschema:"an optional name for the schedule; the multiplexer derives one from the directory when it is empty"`
	Session        string `json:"session,omitempty" jsonschema:"leave empty to start a fresh session each run; give a session name to reuse one session, so it keeps its memory across runs"`
	Model          string `json:"model,omitempty" jsonschema:"the model of the session the run starts; the default model when it is empty"`
	PermissionMode string `json:"permission_mode,omitempty" jsonschema:"the permission mode of the session the run starts; the default when it is empty"`
	Effort         string `json:"effort,omitempty" jsonschema:"the effort level of the session the run starts"`
	Control        bool   `json:"control,omitempty" jsonschema:"true gives the session the control grant, so its own tools can drive other sessions"`
}

type scheduleOut struct {
	OK       bool     `json:"ok"`
	Schedule Schedule `json:"schedule"`
	Message  string   `json:"message"`
}

type listSchedulesOut struct {
	Schedules []Schedule `json:"schedules"`
}

type deleteScheduleIn struct {
	Name string `json:"name" jsonschema:"the name of the schedule to remove"`
}

type deleteScheduleOut struct {
	OK      bool   `json:"ok"`
	Changed bool   `json:"changed"`
	Message string `json:"message"`
}

type setScheduleEnabledIn struct {
	Name    string `json:"name" jsonschema:"the name of the schedule to change"`
	Enabled bool   `json:"enabled" jsonschema:"true runs the schedule on its cron; false pauses it, and keeps it on disk"`
}

type runScheduleIn struct {
	Name string `json:"name" jsonschema:"the name of the schedule to run now, whatever its cron says"`
}

type runScheduleOut struct {
	OK      bool   `json:"ok"`
	Session string `json:"session"`
	Message string `json:"message"`
}

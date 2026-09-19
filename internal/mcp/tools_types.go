package mcp

import (
	"time"

	"github.com/dextermb/claude-multiplexer/internal/config"
)

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

type configureJiraIn struct {
	Token string `json:"token" jsonschema:"the Jira API token or service-account key"`
	Email string `json:"email,omitempty" jsonschema:"the account email; set it for a personal token (Basic auth), and leave it empty for a service-account key (bearer)"`
	URL   string `json:"url,omitempty" jsonschema:"the MCP endpoint, or empty for the default https://mcp.atlassian.com/v2/mcp"`
}

type configureLinearIn struct {
	Token string `json:"token" jsonschema:"the Linear API key"`
	URL   string `json:"url,omitempty" jsonschema:"the MCP endpoint, or empty for the default https://mcp.linear.app/mcp"`
}

type configureWorkItemOut struct {
	OK       bool   `json:"ok"`
	Provider string `json:"provider"`
	Path     string `json:"path"`
	Message  string `json:"message"`
}

type getWorkItemIn struct {
	Session string `json:"session,omitempty" jsonschema:"the session to read, or empty for this session"`
}

type setWorkItemIn struct {
	Provider string `json:"provider,omitempty" jsonschema:"the provider: jira or linear; leave empty when only one is configured"`
	Key      string `json:"key" jsonschema:"the work-item key, such as PROJ-123 for Jira or ENG-45 for Linear"`
}

type setWorkItemStatusIn struct {
	Status string `json:"status" jsonschema:"the target status, which must be one of the statuses the platform offers for this item"`
}

type workItemOut struct {
	OK      bool     `json:"ok"`
	Linked  bool     `json:"linked"`
	Item    WorkItem `json:"item"`
	Message string   `json:"message"`
}

type workItemStatusesOut struct {
	OK       bool             `json:"ok"`
	Statuses []WorkItemStatus `json:"statuses"`
	Message  string           `json:"message"`
}

type configureGitHubIn struct {
	Token string `json:"token,omitempty" jsonschema:"a GitHub token with repo read scope; leave empty to use the gh CLI login"`
	Mode  string `json:"mode,omitempty" jsonschema:"the transport: auto, api, or cli; empty is auto (api with a token, else the gh CLI)"`
	URL   string `json:"url,omitempty" jsonschema:"the GraphQL endpoint, or empty for the default https://api.github.com/graphql"`
}

type configureGitLabIn struct {
	Token string `json:"token,omitempty" jsonschema:"a GitLab token with read_api scope; leave empty to use the glab CLI login"`
	Mode  string `json:"mode,omitempty" jsonschema:"the transport: auto, api, or cli; empty is auto (api with a token, else the glab CLI)"`
	URL   string `json:"url,omitempty" jsonschema:"the GraphQL endpoint, or empty for the default https://gitlab.com/api/graphql"`
}

type configurePROut struct {
	OK       bool   `json:"ok"`
	Provider string `json:"provider"`
	Path     string `json:"path"`
	Message  string `json:"message"`
}

type getPRIn struct {
	Session string `json:"session,omitempty" jsonschema:"the session to read, or empty for this session"`
}

type pullRequestOut struct {
	OK      bool          `json:"ok"`
	Count   int           `json:"count"`
	PRs     []PullRequest `json:"prs"`
	Message string        `json:"message"`
}

type listIn struct {
	Stopped    bool   `json:"stopped,omitempty" jsonschema:"true to also return the sessions that are stored and not running now"`
	Archived   bool   `json:"archived,omitempty" jsonschema:"true to also return the archived sessions"`
	LastActive string `json:"last_active,omitempty" jsonschema:"how recently a stored or archived session was last active to still return it: 1d, 1w, 1m, 1y, or unset for no limit; 1d by default; running sessions are always returned"`
}

// The windows last_active accepts. The vocabulary lives in config, so the MCP
// tools and the TUI archived list share one set. See docs/config.md.
const (
	LastActiveDay   = config.LastActiveDay
	LastActiveWeek  = config.LastActiveWeek
	LastActiveMonth = config.LastActiveMonth
	LastActiveYear  = config.LastActiveYear
	LastActiveUnset = config.LastActiveUnset
)

// DefaultLastActive is the window a call takes when it names none.
const DefaultLastActive = config.DefaultLastActive

// ErrBadLastActive is the failure when last_active is not a known window.
var ErrBadLastActive = config.ErrBadLastActive

// ParseLastActive turns a window into a lookback duration. An empty window takes
// the default of one day, and "unset" returns zero, which means no limit.
var ParseLastActive = config.ParseLastActive

// keep decides one session. A running session always stays. A stored or archived
// session stays only when its category is asked for and it was active since the
// cutoff. A zero cutoff, or a session with no last-active time, passes the cutoff.
func (in listIn) keep(item Session, cutoff time.Time) bool {
	if item.Archived {
		return in.Archived && activeSince(item, cutoff)
	}
	if !item.Live {
		return in.Stopped && activeSince(item, cutoff)
	}
	return true
}

func activeSince(item Session, cutoff time.Time) bool {
	if cutoff.IsZero() || item.LastActiveAt.IsZero() {
		return true
	}
	return !item.LastActiveAt.Before(cutoff)
}

// lastActiveCutoff turns a window into the earliest last-active time a session
// may have and still stay. A zero result means no limit.
func lastActiveCutoff(window string, now time.Time) (time.Time, error) {
	d, err := ParseLastActive(window)
	if err != nil {
		return time.Time{}, err
	}
	if d == 0 {
		return time.Time{}, nil
	}
	return now.Add(-d), nil
}

func (in listIn) filter(all []Session, now time.Time) ([]Session, error) {
	cutoff, err := lastActiveCutoff(in.LastActive, now)
	if err != nil {
		return nil, err
	}
	return selectSessions(all, func(item Session) bool {
		return in.keep(item, cutoff)
	}), nil
}

// listCategoryIn is the input to list_inactive_sessions and
// list_archived_sessions, which each name one category and take only the window.
type listCategoryIn struct {
	LastActive string `json:"last_active,omitempty" jsonschema:"how recently a session was last active to still return it: 1d, 1w, 1m, 1y, or unset for no limit; 1d by default"`
}

func (in listCategoryIn) filter(all []Session, now time.Time, inCategory func(Session) bool) ([]Session, error) {
	cutoff, err := lastActiveCutoff(in.LastActive, now)
	if err != nil {
		return nil, err
	}
	return selectSessions(all, func(item Session) bool {
		return inCategory(item) && activeSince(item, cutoff)
	}), nil
}

func selectSessions(all []Session, keep func(Session) bool) []Session {
	out := make([]Session, 0, len(all))
	for _, item := range all {
		if keep(item) {
			out = append(out, item)
		}
	}
	return out
}

func isInactive(item Session) bool { return !item.Live && !item.Archived }

func isArchived(item Session) bool { return item.Archived }

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

type stopWhenIdleIn struct {
	Stop    *bool `json:"stop,omitempty" jsonschema:"stop this session when it next goes idle; defaults to true; set stop false and archive false to disarm"`
	Archive bool  `json:"archive,omitempty" jsonschema:"archive this session after the stop; implies a stop"`
}

type createIn struct {
	Path    string `json:"path" jsonschema:"the directory the new session works in"`
	Name    string `json:"name,omitempty" jsonschema:"an optional name for the new session"`
	Model   string `json:"model,omitempty" jsonschema:"the model of the new session; the default model when it is empty"`
	Effort  string `json:"effort,omitempty" jsonschema:"the effort level of the new session; low, medium, high, xhigh, or max; the default when it is empty"`
	Profile string `json:"profile,omitempty" jsonschema:"the tool profile of the new session; minimal carries the session reads only, standard carries every open tool; the setting default when it is empty"`
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

type setAutoArchiveIn struct {
	Days int `json:"days" jsonschema:"the number of days a stopped session waits, idle, before the multiplexer archives it; must be one or more"`
}

type setAutoArchiveOut struct {
	OK      bool   `json:"ok"`
	Path    string `json:"path"`
	Days    int    `json:"days"`
	Message string `json:"message"`
}

type unsetAutoArchiveOut struct {
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

type listLocksIn struct {
	Session string `json:"session,omitempty" jsonschema:"the session to read the locks of; empty means this session"`
}

type lockIn struct {
	Lock string `json:"lock" jsonschema:"the label to take or release, such as 'repo:claude-multiplexer' or 'file:internal/manager/meta.go'"`
}

type setLocksIn struct {
	Locks []string `json:"locks" jsonschema:"the whole set of labels this session holds; an empty list releases every lock"`
}

type findLockedIn struct {
	Locks []string `json:"locks" jsonschema:"the labels to search for; a session matches only when it holds every one of them"`
	Live  bool     `json:"live,omitempty" jsonschema:"true to return only the sessions that run now"`
}

type lockOut struct {
	OK      bool     `json:"ok"`
	Locks   []string `json:"locks"`
	Changed bool     `json:"changed"`
	Holders []string `json:"holders,omitempty"`
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

type apiURLOut struct {
	URL string `json:"url"`
}

type peerURLOut struct {
	URL     string `json:"url"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Enabled bool   `json:"enabled"`
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
	Cron           string `json:"cron,omitempty" jsonschema:"a 5-field cron expression in local time, such as '*/5 * * * *' for every 5 minutes or '0 9 * * 1-5' for 09:00 on weekdays; leave it empty for a one-off run and set run_after instead"`
	RunAfter       string `json:"run_after,omitempty" jsonschema:"a one-off run time; a Go duration from now such as '30m' or '2h', or an RFC3339 timestamp; set it only when cron is empty"`
	Dir            string `json:"dir" jsonschema:"the directory the run works in; a relative path is resolved against the directory the multiplexer runs in"`
	Prompt         string `json:"prompt" jsonschema:"the prompt the schedule sends on each run"`
	Name           string `json:"name,omitempty" jsonschema:"an optional name for the schedule; the multiplexer derives one from the directory when it is empty"`
	Session        string `json:"session,omitempty" jsonschema:"leave empty to start a fresh session each run; give a session name to reuse one session, so it keeps its memory across runs"`
	Model          string `json:"model,omitempty" jsonschema:"the model of the session the run starts; the default model when it is empty"`
	PermissionMode string `json:"permission_mode,omitempty" jsonschema:"the permission mode of the session the run starts; the default when it is empty"`
	Effort         string `json:"effort,omitempty" jsonschema:"the effort level of the session the run starts"`
	Control        bool   `json:"control,omitempty" jsonschema:"true gives the session the control grant, so its own tools can drive other sessions; only a caller that holds the control grant may set it, and the multiplexer drops it otherwise"`
}

type updateScheduleIn struct {
	Name           string  `json:"name" jsonschema:"the name of the schedule to change"`
	Cron           *string `json:"cron,omitempty" jsonschema:"a new 5-field cron in local time; it switches the schedule to recurring and clears run_after; a field left out stays as it is"`
	RunAfter       *string `json:"run_after,omitempty" jsonschema:"a new one-off run time, a Go duration or an RFC3339 timestamp; it switches the schedule to one-off and clears cron; an empty string clears it; a field left out stays as it is"`
	Dir            *string `json:"dir,omitempty" jsonschema:"a new directory the run works in; a field left out stays as it is"`
	Prompt         *string `json:"prompt,omitempty" jsonschema:"a new prompt the schedule sends on each run; a field left out stays as it is"`
	Session        *string `json:"session,omitempty" jsonschema:"a session name to reuse one session; an empty string returns to a fresh session each run; a field left out stays as it is"`
	Model          *string `json:"model,omitempty" jsonschema:"a new model; an empty string returns to the default model; a field left out stays as it is"`
	PermissionMode *string `json:"permission_mode,omitempty" jsonschema:"a new permission mode; an empty string returns to the default; a field left out stays as it is"`
	Effort         *string `json:"effort,omitempty" jsonschema:"a new effort level; an empty string returns to the default; a field left out stays as it is"`
	Control        *bool   `json:"control,omitempty" jsonschema:"true gives the session the control grant; only a caller that holds the control grant may set it; a field left out stays as it is"`
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

// Package mcp serves the multiplexer's own tools to the Claude inside each
// session. See docs/mcp.md.
package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ServerName is the MCP server name, so a tool reaches Claude Code as
// mcp__mux__<tool>.
const ServerName = "mux"

const (
	ToolRename          = "rename_session"
	ToolList            = "list_sessions"
	ToolMessages        = "get_messages"
	ToolListJobs        = "list_jobs"
	ToolSetEditor       = "set_editor"
	ToolUnsetEditor     = "unset_editor"
	ToolSetWorkingDir   = "set_working_dir"
	ToolUnsetWorkingDir = "unset_working_dir"
	ToolSend            = "send_message"
	ToolStop            = "stop_session"
	ToolArchive         = "archive_session"
	ToolCreate          = "create_session"
	ToolStopJob         = "stop_job"
)

// OpenTools go to every session. ControlTools go only to a session that holds
// the control grant.
var (
	OpenTools = []string{ToolRename, ToolList, ToolMessages, ToolListJobs, ToolSetEditor, ToolUnsetEditor,
		ToolSetWorkingDir, ToolUnsetWorkingDir}
	ControlTools = []string{ToolSend, ToolStop, ToolArchive, ToolCreate, ToolStopJob}
)

var (
	ErrSelfSend = errors.New("mcp: a session cannot send a prompt to itself")
	ErrSelfStop = errors.New("mcp: a session cannot stop itself")
	ErrNoTarget = errors.New("mcp: this tool needs a session name")
	ErrNoPath   = errors.New("mcp: this tool needs a directory path")
	ErrNoJob    = errors.New("mcp: this tool needs a job id")
	ErrNoEditor = errors.New("mcp: this tool needs an editor, a terminal flag, or both")
	ErrNoDir    = errors.New("mcp: this tool needs a directory path")
)

// AllowedTools names the tools a session may call, in the form Claude Code
// takes on --allowedTools.
func AllowedTools(control bool) []string {
	names := append([]string{}, OpenTools...)
	if control {
		names = append(names, ControlTools...)
	}
	out := make([]string, len(names))
	for i, name := range names {
		out[i] = Qualify(name)
	}
	return out
}

func Qualify(tool string) string {
	return fmt.Sprintf("mcp__%s__%s", ServerName, tool)
}

// Session is one row of list_sessions. It repeats what the manager holds, so
// this package needs nothing from the manager package.
type Session struct {
	Name     string  `json:"name"`
	Title    string  `json:"title,omitempty"`
	Dir      string  `json:"dir"`
	State    string  `json:"state"`
	Model    string  `json:"model,omitempty"`
	Live     bool    `json:"live"`
	Archived bool    `json:"archived,omitempty"`
	Control  bool    `json:"control,omitempty"`
	Queued   int     `json:"queued,omitempty"`
	Turns    int     `json:"turns"`
	Cost     float64 `json:"cost_usd"`
}

// Message is one entry of get_messages. The transcript carries no timestamp for
// a message, so neither does this.
type Message struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

// Job is one row of list_jobs. It repeats what the session holds, so this
// package needs nothing from the session package. See docs/mcp.md.
type Job struct {
	ID          string `json:"id"`
	Description string `json:"description,omitempty"`
	TaskType    string `json:"task_type,omitempty"`
	Status      string `json:"status"`
	Running     bool   `json:"running"`
}

// Sessions is the slice of the manager this package uses. Every method that
// changes something takes the name of the calling session, so the interface can
// tell the human who did it.
type Sessions interface {
	SetTitle(name, title string) error
	SendFrom(target, from, text string) (int, error)
	Stop(ctx context.Context, name, by string) error
	Archive(name string, archived bool, by string) error
	Create(dir, name, by string) (string, error)
	List() []Session
	Messages(name string, limit int) ([]Message, error)
	Jobs(name string) ([]Job, error)
	SetEditor(editor string, terminal *bool, by string) (string, error)
	UnsetEditor(field, by string) (string, bool, error)
	SetWorkingDir(path, by string) (string, error)
	UnsetWorkingDir(by string) (bool, error)
	StopJob(target, jobID, by string) (int, error)
}

// DefaultMessageLimit is how many messages get_messages returns when the caller
// asks for no limit.
const DefaultMessageLimit = 20

const maxMessageLimit = 200

func clampLimit(limit int) int {
	if limit <= 0 {
		return DefaultMessageLimit
	}
	if limit > maxMessageLimit {
		return maxMessageLimit
	}
	return limit
}

func cleanTarget(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrNoTarget
	}
	return name, nil
}

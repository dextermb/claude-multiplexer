// Package mcp serves the multiplexer's own tools to the Claude inside each
// session. See docs/mcp.md.
package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/dextermb/claude-multiplexer/internal/usage"
	"github.com/dextermb/claude-multiplexer/internal/wire"
)

// ServerName is the MCP server name, so a tool reaches Claude Code as
// mcp__cmux__<tool>. It is the name install-as gives the binary. See
// docs/mcp.md.
const ServerName = "cmux"

const (
	ToolRename          = "rename_session"
	ToolList            = "list_sessions"
	ToolMessages        = "get_messages"
	ToolListJobs        = "list_jobs"
	ToolConfigPath      = "get_config_path"
	ToolTemplatePath    = "get_template_path"
	ToolSetConfig       = "set_config"
	ToolUnsetConfig     = "unset_config"
	ToolSetEditor       = "set_editor"
	ToolUnsetEditor     = "unset_editor"
	ToolSetBlockCap     = "set_block_cap"
	ToolUnsetBlockCap   = "unset_block_cap"
	ToolSetWorkingDir   = "set_working_dir"
	ToolUnsetWorkingDir = "unset_working_dir"
	ToolListProject     = "list_project"
	ToolAddProjectDir   = "add_project_dir"
	ToolRemoveProject   = "remove_project_dir"
	ToolSetProject      = "set_project"
	ToolClearProject    = "clear_project"
	ToolListLayouts     = "list_layouts"
	ToolSaveLayout      = "save_layout"
	ToolDeleteLayout    = "delete_layout"
	ToolSetLayout       = "set_layout"
	ToolUnsetLayout     = "unset_layout"
	ToolSend            = "send_message"
	ToolStop            = "stop_session"
	ToolArchive         = "archive_session"
	ToolCreate          = "create_session"
	ToolStopJob         = "stop_job"

	ToolCreateSchedule     = "create_schedule"
	ToolUpdateSchedule     = "update_schedule"
	ToolListSchedules      = "list_schedules"
	ToolDeleteSchedule     = "delete_schedule"
	ToolSetScheduleEnabled = "set_schedule_enabled"
	ToolRunSchedule        = "run_schedule"
	ToolSchedulePath       = "get_schedule_path"

	ToolAPIURL          = "get_api_url"
	ToolCreateAPIAdmin  = "create_api_admin"
	ToolRotateAPIAdmin  = "rotate_api_admin"
	ToolRevokeAPIAdmin  = "revoke_api_admin"
	ToolCreateAPIClient = "create_api_client"
	ToolUpdateAPIClient = "update_api_client"
	ToolRotateAPIClient = "rotate_api_client"
	ToolRevokeAPIClient = "revoke_api_client"
	ToolListAPIClients  = "list_api_clients"
	ToolCreateAPIKey    = "create_api_key"
	ToolRevokeAPIKey    = "revoke_api_key"
	ToolAPIEndpoint     = "get_api_endpoint"
	ToolAPIDocs         = "get_api_docs"

	ToolGetUsage  = "get_usage"
	ToolPeerUsage = "peer_usage"

	ToolListPeers      = "list_peers"
	ToolEnablePeering  = "enable_peering"
	ToolDisablePeering = "disable_peering"
	ToolAddPeer        = "add_peer"
	ToolUpdatePeer     = "update_peer"
	ToolRemovePeer     = "remove_peer"
	ToolSetReserve     = "set_reserve"
	ToolUnsetReserve   = "unset_reserve"
)

// OpenTools go to every session. ControlTools go only to a session that holds
// the control grant.
var (
	OpenTools = []string{ToolRename, ToolList, ToolMessages, ToolListJobs, ToolConfigPath, ToolTemplatePath,
		ToolSetConfig, ToolUnsetConfig,
		ToolSetEditor, ToolUnsetEditor, ToolSetBlockCap, ToolUnsetBlockCap, ToolSetWorkingDir, ToolUnsetWorkingDir,
		ToolListProject, ToolAddProjectDir, ToolRemoveProject, ToolSetProject, ToolClearProject,
		ToolListLayouts, ToolSaveLayout, ToolDeleteLayout, ToolSetLayout, ToolUnsetLayout,
		ToolCreateSchedule, ToolUpdateSchedule, ToolListSchedules, ToolDeleteSchedule, ToolSetScheduleEnabled, ToolRunSchedule,
		ToolSchedulePath, ToolAPIURL, ToolAPIDocs, ToolGetUsage, ToolPeerUsage}
	ControlTools = []string{ToolSend, ToolStop, ToolArchive, ToolCreate, ToolStopJob,
		ToolCreateAPIAdmin, ToolRotateAPIAdmin, ToolRevokeAPIAdmin,
		ToolCreateAPIClient, ToolUpdateAPIClient, ToolRotateAPIClient, ToolRevokeAPIClient,
		ToolListAPIClients, ToolCreateAPIKey, ToolRevokeAPIKey, ToolAPIEndpoint,
		ToolListPeers, ToolEnablePeering, ToolDisablePeering, ToolAddPeer, ToolUpdatePeer, ToolRemovePeer,
		ToolSetReserve, ToolUnsetReserve}
	// APITools go to an external client that reaches the session API. The set is
	// session-only, so no config, layout, or schedule tool is ever exposed. See
	// docs/mcp/api.md.
	APITools = []string{ToolRename, ToolList, ToolMessages, ToolListJobs,
		ToolSend, ToolStop, ToolArchive, ToolCreate, ToolStopJob}
)

var (
	ErrSelfSend     = errors.New("mcp: a session cannot send a prompt to itself")
	ErrSelfStop     = errors.New("mcp: a session cannot stop itself")
	ErrNotFound     = errors.New("mcp: no such session")
	ErrNoTarget     = errors.New("mcp: this tool needs a session name")
	ErrNoPath       = errors.New("mcp: this tool needs a directory path")
	ErrNoJob        = errors.New("mcp: this tool needs a job id")
	ErrNoClient     = errors.New("mcp: this tool needs a client id")
	ErrNoCredential = errors.New("mcp: this tool needs a credential value")
	ErrNoCron       = errors.New("mcp: this tool needs a cron expression")
	ErrNoPrompt     = errors.New("mcp: this tool needs a prompt")
	ErrNoSchedule   = errors.New("mcp: this tool needs a schedule name")
	ErrNoConfigPath = errors.New("mcp: this tool needs a settings path")
	ErrNoEditor     = errors.New("mcp: this tool needs an editor, a terminal flag, or both")
	ErrNoDir        = errors.New("mcp: this tool needs a directory path")
	ErrBadCap       = errors.New("mcp: the block cap must be zero or more rows")
	ErrCapBoth      = errors.New("mcp: give rows or unlimited, not both")
	ErrBadType      = errors.New("mcp: the block type must be prompt, message, tool, meta, bash, or error")
	ErrNoLayout     = errors.New("mcp: this tool needs a layout name")
	ErrBadScope     = errors.New("mcp: the scope must be session or all")
	ErrBadDim       = errors.New("mcp: a layout dimension must be one or more")

	ErrBadPosition = errors.New("mcp: the diff position must be left, right, top, or bottom")

	ErrNoPeer = errors.New("mcp: this tool needs a peer name and url")
)

// The scopes a layout tool takes. ScopeSession sets the calling session; ScopeAll
// sets the global default. See docs/mcp/tools.md.
const (
	ScopeSession = "session"
	ScopeAll     = "all"
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
	Owner    string  `json:"owner,omitempty"`
	Queued   int     `json:"queued,omitempty"`
	Turns    int     `json:"turns"`
	Cost     float64 `json:"cost_usd"`
	// Host names the peer a streamed session runs on, and is empty for a session
	// this host runs. Hosted marks a session this host runs on behalf of a peer.
	// The sidebar sorts a session into a section from these two. See docs/peers.md.
	Host   string `json:"host,omitempty"`
	Hosted bool   `json:"hosted,omitempty"`
	// Lender names the peer whose Claude credential a hoisted session runs with.
	// A hoisted session runs locally, so Host is empty and Hosted is false. See
	// docs/peers/hoisted.md.
	Lender string `json:"lender,omitempty"`
}

// Message is one entry of get_messages. The transcript carries no timestamp for
// a message, so neither does this.
type Message struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

// Job is one row of list_jobs. It repeats what the session holds, so this
// package needs nothing from the session package. See docs/mcp/tools.md.
type Job struct {
	ID          string `json:"id"`
	Description string `json:"description,omitempty"`
	TaskType    string `json:"task_type,omitempty"`
	Status      string `json:"status"`
	Running     bool   `json:"running"`
}

// Schedule is one row of list_schedules, and the record the schedule tools
// return. It repeats what the manager holds, so this package needs nothing from
// the manager package. See docs/scheduler.md.
type Schedule struct {
	Name           string `json:"name"`
	Cron           string `json:"cron"`
	Dir            string `json:"dir"`
	Prompt         string `json:"prompt"`
	Session        string `json:"session,omitempty"`
	Model          string `json:"model,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	Effort         string `json:"effort,omitempty"`
	Control        bool   `json:"control,omitempty"`
	Enabled        bool   `json:"enabled"`
	LastRun        string `json:"last_run,omitempty"`
	LastSession    string `json:"last_session,omitempty"`
}

// ScheduleInput is the input to CreateSchedule, so the manager package fills the
// rest of the record.
type ScheduleInput struct {
	Name           string
	Cron           string
	Dir            string
	Prompt         string
	Session        string
	Model          string
	PermissionMode string
	Effort         string
	Control        bool
}

// ScheduleEdit is the input to UpdateSchedule. A nil field stays as it is, so the
// caller changes only the fields it sends.
type ScheduleEdit struct {
	Cron           *string
	Dir            *string
	Prompt         *string
	Session        *string
	Model          *string
	PermissionMode *string
	Effort         *string
	Control        *bool
}

// ConfigPath names the settings files, in the order they are read. See
// docs/mcp/tools.md.
type ConfigPath struct {
	Paths  []string `json:"paths"`
	Active string   `json:"active,omitempty"`
	Target string   `json:"target"`
}

// LayoutDims are the interface dimensions a layout sets. A nil field takes the
// built-in default, so a layout may set only some of them. See docs/mcp/tools.md.
type LayoutDims struct {
	PromptMin    *int    `json:"promptMin,omitempty"`
	PromptMax    *int    `json:"promptMax,omitempty"`
	SidebarSize  *int    `json:"sidebarSize,omitempty"`
	TaskSize     *int    `json:"taskSize,omitempty"`
	DiffSize     *int    `json:"diffSize,omitempty"`
	DiffPosition *string `json:"diffPosition,omitempty"`
}

// LayoutInfo is one row of list_layouts.
type LayoutInfo struct {
	Name string `json:"name"`
	LayoutDims
}

// LayoutList is the output of list_layouts: the named layouts, the global active
// layout, and the layout of the calling session. See docs/mcp/tools.md.
type LayoutList struct {
	Session       string       `json:"session"`
	ActiveGlobal  string       `json:"active_global,omitempty"`
	ActiveSession string       `json:"active_session,omitempty"`
	Layouts       []LayoutInfo `json:"layouts"`
}

// SchedulePath names the directory the multiplexer writes schedule records to.
// See docs/scheduler.md.
type SchedulePath struct {
	Dir string `json:"dir"`
}

// TemplatePath names the directories one session reads a preset prompt from,
// in the order they are read. See docs/mcp/tools.md.
type TemplatePath struct {
	Session string   `json:"session"`
	Root    string   `json:"root"`
	Dir     string   `json:"dir"`
	Dirs    []string `json:"dirs"`
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
	ConfigPath() ConfigPath
	TemplatePath(name string) (TemplatePath, error)
	SetConfig(path string, value json.RawMessage, by string) (string, error)
	UnsetConfig(path, by string) (string, bool, error)
	SetEditor(editor string, terminal *bool, by string) (string, error)
	UnsetEditor(field, by string) (string, bool, error)
	SetBlockCap(bucket string, rows *int, by string) (string, error)
	UnsetBlockCap(bucket, by string) (string, bool, error)
	SetWorkingDir(path, by string) (string, error)
	UnsetWorkingDir(by string) (bool, error)
	Project(session string) ([]string, error)
	SetProject(paths []string, by string) ([]string, error)
	AddProjectDir(path, by string) ([]string, error)
	RemoveProjectDir(path, by string) ([]string, error)
	ClearProject(by string) (bool, error)
	Layouts(session string) (LayoutList, error)
	SaveLayout(name string, dims LayoutDims, by string) (string, error)
	DeleteLayout(name, by string) (string, bool, error)
	SetLayout(name, scope, by string) (string, error)
	UnsetLayout(scope, by string) (string, bool, error)
	StopJob(target, jobID, by string) (int, error)
	CreateSchedule(in ScheduleInput, by string) (Schedule, error)
	UpdateSchedule(name string, up ScheduleEdit, by string) (Schedule, error)
	ListSchedules() []Schedule
	DeleteSchedule(name, by string) (bool, error)
	SetScheduleEnabled(name string, on bool, by string) (Schedule, error)
	RunSchedule(name, by string) (string, error)
	SchedulePath() SchedulePath
	CreateAPIAdmin() (string, error)
	RotateAPIAdmin() (string, error)
	RevokeAPIAdmin() error
	CreateAPIClient(name string) (APIClient, string, error)
	UpdateAPIClient(id string, name *string, disabled *bool) (APIClient, error)
	RotateAPIClient(id string) (string, error)
	RevokeAPIClient(id string) error
	ListAPIClients() []APIClient
	CreateAPIKey(client, credentialType, value string) (APIClient, error)
	RevokeAPIKey(client string) (bool, error)
	APIEndpoint() APIEndpoint
	Usage() usage.Usage
	PeerUsage(ctx context.Context) []PeerReport
	Peers() PeersView
	EnablePeering(port int) (string, error)
	DisablePeering() (string, bool, error)
	AddPeer(in PeerHostInput) (string, error)
	UpdatePeer(in PeerHostUpdate) (string, error)
	RemovePeer(name string) (string, bool, error)
	SetReserve(window string, minPercent int) (string, error)
	UnsetReserve() (string, bool, error)
	// HostingPaused reports whether the reserve gate is tripped, so the peer
	// listener refuses a new hosted session. See docs/peers.md.
	HostingPaused() bool
}

// PeersView is the output of list_peers: the listen address, the reserve, and
// the peer hosts. It never holds a secret. See docs/peers.md.
type PeersView struct {
	Enabled bool           `json:"enabled"`
	Port    int            `json:"port,omitempty"`
	Reserve *ReserveView   `json:"reserve,omitempty"`
	Hosts   []PeerHostView `json:"hosts"`
}

// ReserveView is the usage floor in a PeersView.
type ReserveView struct {
	Window     string `json:"window"`
	MinPercent int    `json:"min_percent"`
}

// PeerHostView is one peer host in a PeersView, without its secret. Credential
// marks a peer that holds a lent Claude credential, with no value. See
// docs/peers/hoisted.md.
type PeerHostView struct {
	Name       string              `json:"name"`
	URL        string              `json:"url,omitempty"`
	ClientID   string              `json:"client_id,omitempty"`
	Credential *PeerCredentialView `json:"credential,omitempty"`
}

// PeerCredentialView is the lent credential in a PeerHostView: its type and the
// last four characters, never the value. See docs/peers/hoisted.md.
type PeerCredentialView struct {
	Type  string `json:"type"`
	Last4 string `json:"last4"`
}

// PeerHostInput is the input to AddPeer: a peer host with its secret, and an
// optional lent Claude credential that turns hoisting on. See docs/peers.md and
// docs/peers/hoisted.md.
type PeerHostInput struct {
	Name           string
	URL            string
	ClientID       string
	ClientSecret   string
	CredentialType string
	Credential     string
}

// PeerHostUpdate is the input to UpdatePeer. Name finds the peer; each other
// field changes it only when non-empty, so a regenerated secret or a new url
// updates without re-supplying the rest. ClearCredential removes the lent
// credential. See docs/peers.md and docs/peers/hoisted.md.
type PeerHostUpdate struct {
	Name            string
	URL             string
	ClientID        string
	ClientSecret    string
	CredentialType  string
	Credential      string
	ClearCredential bool
}

// PeerReport is one peer's usage, or the error that stopped the read. See
// docs/peers.md.
type PeerReport struct {
	Name      string      `json:"name"`
	URL       string      `json:"url"`
	Reachable bool        `json:"reachable"`
	Usage     usage.Usage `json:"usage,omitempty"`
	Error     string      `json:"error,omitempty"`
}

// APISessions is the session-only slice of the manager an external API client
// reaches. Each client sees an owner-scoped view, so a method works only on the
// sessions the client owns. See docs/mcp/api.md.
type APISessions interface {
	SetTitle(name, title string) error
	SendFrom(target, from, text string) (int, error)
	Stop(ctx context.Context, name, by string) error
	Interrupt(ctx context.Context, name, by string) error
	Archive(name string, archived bool, by string) error
	Create(in CreateInput, by string) (string, error)
	List() []Session
	Messages(name string, limit int) ([]Message, error)
	Jobs(name string) ([]Job, error)
	StopJob(target, jobID, by string) (int, error)
	// Stream replays the session's current lines, then tails its live events,
	// until the context is done. It is the source of a remote session's stream.
	// See docs/peers.md.
	Stream(ctx context.Context, name string) (<-chan wire.Event, error)
}

// CreateInput is the input to Create over the API. It carries the fields a peer
// picks for a session it starts on a host: the directory, the name, and the
// model, permission mode, and effort. See docs/peers.md.
type CreateInput struct {
	Dir            string
	Name           string
	Model          string
	PermissionMode string
	Effort         string
	// Hosted marks a session the peer listener creates on behalf of a peer, so
	// the host sorts it under the hosted section. The loopback API never sets it.
	// See docs/peers.md.
	Hosted bool
	// TempDir asks the host to run the session in a fresh temporary directory it
	// makes, so a peer starts a session without naming a path on the host. See
	// docs/peers.md.
	TempDir bool
}

// APIClient is one row of list_api_clients, and the record the client tools
// return. It never holds the secret. LentKey marks a client that holds a lent
// Claude credential, with no value. See docs/mcp/api.md and docs/peers/hoisted.md.
type APIClient struct {
	ClientID  string      `json:"client_id"`
	Name      string      `json:"name"`
	Disabled  bool        `json:"disabled,omitempty"`
	CreatedAt string      `json:"created_at,omitempty"`
	RotatedAt string      `json:"rotated_at,omitempty"`
	LentKey   *APILentKey `json:"lent_key,omitempty"`
}

// APILentKey is the metadata of a lent Claude credential in an APIClient. It
// holds the type and the last four characters, never the value. See
// docs/peers/hoisted.md.
type APILentKey struct {
	Type     string `json:"type"`
	Last4    string `json:"last4"`
	IssuedAt string `json:"issued_at,omitempty"`
}

// APIEndpoint names the base URL of the API, and the port range it binds inside.
// See docs/mcp/api.md.
type APIEndpoint struct {
	URL       string `json:"url"`
	PortStart int    `json:"port_start"`
	PortEnd   int    `json:"port_end"`
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

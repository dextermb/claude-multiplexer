// Package mcp serves the multiplexer's own tools to the Claude inside each
// session. See docs/mcp.md.
package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/usage"
	"github.com/dextermb/claude-multiplexer/internal/wire"
)

// ServerName is the MCP server name, so a tool reaches Claude Code as
// mcp__cmux__<tool>. It is the name install-as gives the binary. See
// docs/mcp.md.
const ServerName = "cmux"

const (
	ToolRename        = "rename_session"
	ToolList          = "list_sessions"
	ToolListInactive  = "list_inactive_sessions"
	ToolListArchived  = "list_archived_sessions"
	ToolMessages      = "get_messages"
	ToolListJobs      = "list_jobs"
	ToolConfigPath    = "get_config_path"
	ToolConfigKeys    = "list_config_keys"
	ToolTemplatePath  = "get_template_path"
	ToolSetConfig     = "set_config"
	ToolUnsetConfig   = "unset_config"
	ToolSetEditor     = "set_editor"
	ToolUnsetEditor   = "unset_editor"
	ToolSetBlockCap   = "set_block_cap"
	ToolUnsetBlockCap = "unset_block_cap"

	ToolSetAutoArchive   = "set_auto_archive"
	ToolUnsetAutoArchive = "unset_auto_archive"

	ToolSetWorkingDir   = "set_working_dir"
	ToolUnsetWorkingDir = "unset_working_dir"
	ToolListProject     = "list_project"
	ToolAddProjectDir   = "add_project_dir"
	ToolRemoveProject   = "remove_project_dir"
	ToolSetProject      = "set_project"
	ToolClearProject    = "clear_project"

	ToolListLocks  = "list_locks"
	ToolAddLock    = "add_lock"
	ToolRemoveLock = "remove_lock"
	ToolSetLocks   = "set_locks"
	ToolClearLocks = "clear_locks"
	ToolFindLocked = "find_locked_sessions"

	ToolConfigureJira   = "configure_jira"
	ToolConfigureLinear = "configure_linear"

	ToolGetWorkItem       = "get_workitem"
	ToolSetWorkItem       = "set_workitem"
	ToolUnsetWorkItem     = "unset_workitem"
	ToolWorkItemStatuses  = "list_workitem_statuses"
	ToolSetWorkItemStatus = "set_workitem_status"

	ToolListLayouts  = "list_layouts"
	ToolSaveLayout   = "save_layout"
	ToolDeleteLayout = "delete_layout"
	ToolSetLayout    = "set_layout"
	ToolUnsetLayout  = "unset_layout"
	ToolSend         = "send_message"
	ToolStop         = "stop_session"
	ToolArchive      = "archive_session"
	ToolStopWhenIdle = "stop_when_idle"
	ToolCreate       = "create_session"
	ToolStopJob      = "stop_job"

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

	ToolPeerURL        = "get_peer_url"
	ToolListPeers      = "list_peers"
	ToolEnablePeering  = "enable_peering"
	ToolDisablePeering = "disable_peering"
	ToolAddPeer        = "add_peer"
	ToolUpdatePeer     = "update_peer"
	ToolRemovePeer     = "remove_peer"
	ToolSetReserve     = "set_reserve"
	ToolUnsetReserve   = "unset_reserve"

	ToolShareSession = "share_session"
	ToolListShares   = "list_shares"
	ToolRevokeShare  = "revoke_share"
	ToolWatchShare   = "watch_share"
)

// A Profile names the open tools a session carries. It is a cost control, and
// not a permission: the control grant adds the control tools whatever the
// profile says. See docs/mcp/profiles.md.
type Profile string

const (
	ProfileMinimal  Profile = "minimal"
	ProfileStandard Profile = "standard"
)

// DefaultProfile is the profile of a session that names none.
const DefaultProfile = ProfileStandard

// ErrBadProfile is the failure when a name is not a profile.
var ErrBadProfile = errors.New("mcp: the profile must be minimal or standard")

// ParseProfile reads a profile name. An empty name takes the default.
func ParseProfile(name string) (Profile, error) {
	switch Profile(strings.TrimSpace(name)) {
	case "":
		return DefaultProfile, nil
	case ProfileMinimal:
		return ProfileMinimal, nil
	case ProfileStandard:
		return ProfileStandard, nil
	}
	return "", ErrBadProfile
}

// MinimalTools are the open tools of the minimal profile: the session reads, and
// the description of the REST API.
var MinimalTools = []string{ToolRename, ToolList, ToolListInactive, ToolListArchived, ToolMessages, ToolListJobs,
	ToolConfigPath, ToolConfigKeys, ToolTemplatePath, ToolAPIDocs}

// OpenToolsFor names the open tools of a profile.
func OpenToolsFor(profile Profile) []string {
	if profile == ProfileMinimal {
		return MinimalTools
	}
	return OpenTools
}

// OpenTools go to every session on the standard profile. ControlTools go only to
// a session that holds the control grant.
var (
	OpenTools = []string{ToolRename, ToolList, ToolListInactive, ToolListArchived, ToolMessages, ToolListJobs, ToolConfigPath, ToolConfigKeys, ToolTemplatePath,
		ToolSetConfig, ToolUnsetConfig,
		ToolSetEditor, ToolUnsetEditor, ToolSetBlockCap, ToolUnsetBlockCap,
		ToolSetAutoArchive, ToolUnsetAutoArchive, ToolSetWorkingDir, ToolUnsetWorkingDir,
		ToolListProject, ToolAddProjectDir, ToolRemoveProject, ToolSetProject, ToolClearProject,
		ToolListLocks, ToolAddLock, ToolRemoveLock, ToolSetLocks, ToolClearLocks, ToolFindLocked,
		ToolListLayouts, ToolSaveLayout, ToolDeleteLayout, ToolSetLayout, ToolUnsetLayout,
		ToolCreateSchedule, ToolUpdateSchedule, ToolListSchedules, ToolDeleteSchedule, ToolSetScheduleEnabled, ToolRunSchedule,
		ToolSchedulePath, ToolAPIURL, ToolAPIDocs, ToolGetUsage, ToolPeerUsage, ToolPeerURL,
		ToolShareSession,
		ToolConfigureJira, ToolConfigureLinear}
	// WorkItemTools link a session to a Jira or Linear work item and change its
	// status. They are open, but a session carries them only when a provider is
	// configured, so the gate is the settings file. The configure_* tools that
	// set the token are always open, in OpenTools above, so a session can turn
	// the feature on. See docs/work-items.md.
	WorkItemTools = []string{ToolGetWorkItem, ToolSetWorkItem, ToolUnsetWorkItem, ToolWorkItemStatuses, ToolSetWorkItemStatus}
	ControlTools  = []string{ToolSend, ToolStop, ToolArchive, ToolCreate, ToolStopJob,
		ToolCreateAPIAdmin, ToolRotateAPIAdmin, ToolRevokeAPIAdmin,
		ToolCreateAPIClient, ToolUpdateAPIClient, ToolRotateAPIClient, ToolRevokeAPIClient,
		ToolListAPIClients, ToolCreateAPIKey, ToolRevokeAPIKey, ToolAPIEndpoint,
		ToolListPeers, ToolEnablePeering, ToolDisablePeering, ToolAddPeer, ToolUpdatePeer, ToolRemovePeer,
		ToolSetReserve, ToolUnsetReserve,
		ToolListShares, ToolRevokeShare, ToolWatchShare}
	// APITools go to an external client that reaches the session API. The set is
	// session-only, so no config, layout, or schedule tool is ever exposed. See
	// docs/mcp/api.md.
	APITools = []string{ToolRename, ToolList, ToolListInactive, ToolListArchived, ToolMessages, ToolListJobs,
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
	ErrNoLock       = errors.New("mcp: this tool needs a lock label")

	ErrNoWorkItemKey    = errors.New("mcp: this tool needs a work-item key")
	ErrNoWorkItemStatus = errors.New("mcp: this tool needs a target status")
	ErrNoWorkItemToken  = errors.New("mcp: this tool needs a provider token")
	ErrBadCap           = errors.New("mcp: the block cap must be zero or more rows")
	ErrBadDays          = errors.New("mcp: the auto-archive days must be one or more")
	ErrCapBoth          = errors.New("mcp: give rows or unlimited, not both")
	ErrBadType          = errors.New("mcp: the block type must be prompt, message, tool, meta, bash, or error")
	ErrNoLayout         = errors.New("mcp: this tool needs a layout name")
	ErrBadScope         = errors.New("mcp: the scope must be session or all")
	ErrBadDim           = errors.New("mcp: a layout dimension must be one or more")

	ErrBadPosition = errors.New("mcp: the diff position must be left, right, top, or bottom")

	ErrNoPeer = errors.New("mcp: this tool needs a peer name and url")

	ErrReadOnly = errors.New("mcp: a share is read-only")
	ErrNoShare  = errors.New("mcp: this tool needs a share id")
	ErrNotSelf  = errors.New("mcp: a session may share only itself; a control session may share any session")
)

// The scopes a layout tool takes. ScopeSession sets the calling session; ScopeAll
// sets the global default. See docs/mcp/tools/layouts.md.
const (
	ScopeSession = "session"
	ScopeAll     = "all"
)

// AllowedTools names the tools a session may call, in the form Claude Code
// takes on --allowedTools.
func AllowedTools(profile Profile, control bool) []string {
	names := append([]string{}, OpenToolsFor(profile)...)
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
	// LastActiveAt is when the session last took a turn. It is zero for a session
	// that never took one, and for a streamed peer session. list_sessions filters
	// the stored and archived rows by it. See docs/mcp/tools/sessions.md.
	LastActiveAt time.Time `json:"last_active_at,omitempty"`
	// Host names the peer a streamed session runs on, and is empty for a session
	// this host runs. Hosted marks a session this host runs on behalf of a peer.
	// The sidebar sorts a session into a section from these two. See docs/peers.md.
	Host   string `json:"host,omitempty"`
	Hosted bool   `json:"hosted,omitempty"`
	// ReadOnly marks a spectator session: it streams a peer's session read-only
	// through a share, so the interface disables input for it. See docs/peers.md.
	ReadOnly bool `json:"read_only,omitempty"`
	// Watched marks a session a spectator watches now, through a share this host
	// minted, so the host sees it is shared. See docs/peers.md.
	Watched bool `json:"watched,omitempty"`
	// Lender names the peer whose Claude credential a hoisted session runs with.
	// A hoisted session runs locally, so Host is empty and Hosted is false. See
	// docs/peers/hoisted.md.
	Lender string `json:"lender,omitempty"`
	// Locks are the labels a session holds, to say it works on something and
	// another session must keep away. They are advisory. See
	// docs/mcp/tools/locks.md.
	Locks []string `json:"locks,omitempty"`
}

// WorkItem is the work item a session links to: the provider, the key, and the
// mirror of the item title, url, and status. Linked is false for a session with
// no link. See docs/work-items.md.
type WorkItem struct {
	Provider string `json:"provider,omitempty"`
	Key      string `json:"key,omitempty"`
	Title    string `json:"title,omitempty"`
	URL      string `json:"url,omitempty"`
	Status   string `json:"status,omitempty"`
	StatusID string `json:"status_id,omitempty"`
	Linked   bool   `json:"linked"`
}

// WorkItemStatus is one status a work item may move to: the display name the
// platform offers, and the raw provider id. See docs/work-items.md.
type WorkItemStatus struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
}

// Message is one entry of get_messages. The transcript carries no timestamp for
// a message, so neither does this.
type Message struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

// Job is one row of list_jobs. It repeats what the session holds, so this
// package needs nothing from the session package. See docs/mcp/tools/sessions.md.
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
	RunAfter       string `json:"run_after,omitempty"`
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
	RunAfter       string
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
	RunAfter       *string
	Dir            *string
	Prompt         *string
	Session        *string
	Model          *string
	PermissionMode *string
	Effort         *string
	Control        *bool
}

// ConfigKeys is the output of list_config_keys: the settings key paths, in the
// dot notation set_config and unset_config take, each with its JSON type. A map
// key is a "<key>" placeholder. See docs/mcp/tools/settings.md.
type ConfigKeys struct {
	Keys []config.KeyPath `json:"keys"`
}

// ConfigPath names the settings files, in the order they are read. See
// docs/mcp/tools/settings.md.
type ConfigPath struct {
	Paths  []string `json:"paths"`
	Active string   `json:"active,omitempty"`
	Target string   `json:"target"`
}

// LayoutDims are the interface dimensions a layout sets. A nil field takes the
// built-in default, so a layout may set only some of them. See
// docs/mcp/tools/layouts.md.
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
// layout, and the layout of the calling session. See docs/mcp/tools/layouts.md.
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
// in the order they are read. See docs/mcp/tools/settings.md.
type TemplatePath struct {
	Session string   `json:"session"`
	Root    string   `json:"root"`
	Dir     string   `json:"dir"`
	Dirs    []string `json:"dirs"`
}

// ShareView is one row of list_shares. It never holds the secret. See
// docs/peers.md.
type ShareView struct {
	ID      string    `json:"id"`
	Session string    `json:"session"`
	Scope   string    `json:"scope"`
	Created time.Time `json:"created_at"`
	Expires time.Time `json:"expires_at,omitempty"`
}

// ShareCreated is the result of share_session: the share and its link. The link
// is shown only here, the same as a client secret. See docs/peers.md.
type ShareCreated struct {
	ID      string    `json:"id"`
	Session string    `json:"session"`
	Expires time.Time `json:"expires_at,omitempty"`
	Link    string    `json:"link"`
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
	Unqueue(name string) (bool, error)
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
	// Profile names the open tools the session carries. An empty profile takes
	// the defaultToolProfile setting. See docs/mcp/profiles.md.
	Profile string
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

// PeerEndpoint names the address a peer on the network dials to reach this host:
// the base URL, and the host and port inside it. The host is a routable LAN
// address, not the `0.0.0.0` the listener binds. It is empty when peering is
// off. See docs/peers.md.
type PeerEndpoint struct {
	URL     string `json:"url"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Enabled bool   `json:"enabled"`
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

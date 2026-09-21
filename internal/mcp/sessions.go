package mcp

import (
	"context"
	"encoding/json"

	"github.com/dextermb/claude-multiplexer/internal/keys"
	"github.com/dextermb/claude-multiplexer/internal/usage"
)

// Sessions is the slice of the manager this package uses. It is a composite of
// one port per concept, so a tool group and its test depend on the port they
// need, not the whole surface. Every method that changes something takes the
// name of the calling session, so the interface can tell the human who did it.
// See docs/mcp.md.
type Sessions interface {
	SessionReader
	ControlPort
	ConfigPort
	LockPort
	LayoutPort
	SchedulePort
	UsagePort
	APIPort
	PeerPort
	SharePort
	WorkItemPort
	PullRequestPort
}

// SessionReader reads the shape of a session.
type SessionReader interface {
	List() []Session
	Messages(name string, limit int) ([]Message, error)
	Jobs(name string) ([]Job, error)
}

// ControlPort drives a session. A control grant gates every method.
type ControlPort interface {
	SetTitle(name, title string) error
	SendFrom(target, from, text string) (int, error)
	Stop(ctx context.Context, name, by string) error
	StopJob(target, jobID, by string) (int, error)
	Archive(name string, archived bool, by string) error
	StopWhenIdle(name string, stop, archive bool) error
	Create(in CreateInput, by string) (string, error)
}

// ConfigPort reads and changes the settings and the directories of a session.
type ConfigPort interface {
	ConfigPath() ConfigPath
	TemplatePath(name string) (TemplatePath, error)
	SetConfig(path string, value json.RawMessage, by string) (string, error)
	UnsetConfig(path, by string) (string, bool, error)
	SetKeybinding(action string, keys []string, by string) (string, string, error)
	ResetKeybinding(action, by string) (string, bool, error)
	Keybindings() []keys.Entry
	SetEditor(editor string, terminal *bool, by string) (string, error)
	UnsetEditor(field, by string) (string, bool, error)
	SetBlockCap(bucket string, rows *int, by string) (string, error)
	UnsetBlockCap(bucket, by string) (string, bool, error)
	SetAutoArchive(days int, by string) (string, error)
	UnsetAutoArchive(by string) (string, bool, error)
	SetWorkingDir(path, by string) (string, error)
	UnsetWorkingDir(by string) (bool, error)
	Project(session string) ([]string, error)
	SetProject(paths []string, by string) ([]string, error)
	AddProjectDir(path, by string) ([]string, error)
	RemoveProjectDir(path, by string) ([]string, error)
	ClearProject(by string) (bool, error)
}

// LockPort reads and changes the advisory locks a session holds. See
// docs/mcp/tools/locks.md.
type LockPort interface {
	Locks(session string) ([]string, error)
	SetLocks(labels []string, by string) ([]string, error)
	AddLock(label, by string) ([]string, error)
	RemoveLock(label, by string) ([]string, error)
	ClearLocks(by string) (bool, error)
	FindLocked(labels []string, live bool) ([]Session, error)
}

// WorkItemPort reads and changes the work item a session links to. A read takes
// a session name; a change acts on the calling session. See docs/work-items.md.
type WorkItemPort interface {
	WorkItemsEnabled() bool
	WorkItemProviders() []string
	ConfigureWorkItem(provider, token, email, url, by string) (string, error)
	WorkItem(session string) (WorkItem, error)
	SetWorkItem(ctx context.Context, provider, key, by string) (WorkItem, error)
	UnsetWorkItem(by string) (bool, error)
	WorkItemStatuses(ctx context.Context, by string) ([]WorkItemStatus, error)
	SetWorkItemStatus(ctx context.Context, target, by string) (WorkItem, error)
}

// PullRequestPort reads the pull requests of a session's code bases, and
// configures the providers. A read takes a session name. See
// docs/pull-requests.md.
type PullRequestPort interface {
	PullRequestsEnabled() bool
	ConfigurePullRequest(provider, token, mode, url, by string) (string, error)
	PullRequestsFor(ctx context.Context, session string) ([]PullRequest, error)
}

// LayoutPort reads and changes the saved screen layouts. See
// docs/tui/layouts.md.
type LayoutPort interface {
	Layouts(session string) (LayoutList, error)
	SaveLayout(name string, dims LayoutDims, by string) (string, error)
	DeleteLayout(name, by string) (string, bool, error)
	SetLayout(name, scope, by string) (string, error)
	UnsetLayout(scope, by string) (string, bool, error)
}

// SchedulePort reads and changes the scheduled runs. See docs/scheduler.md.
type SchedulePort interface {
	CreateSchedule(in ScheduleInput, by string) (Schedule, error)
	UpdateSchedule(name string, up ScheduleEdit, by string) (Schedule, error)
	ListSchedules() []Schedule
	DeleteSchedule(name, by string) (bool, error)
	SetScheduleEnabled(name string, on bool, by string) (Schedule, error)
	RunSchedule(name, by string) (string, error)
	SchedulePath() SchedulePath
}

// UsagePort reads the token and cost usage, local and on the peers. See
// docs/cost.md.
type UsagePort interface {
	Usage() usage.Usage
	PeerUsage(ctx context.Context) []PeerReport
}

// APIPort mints and revokes the admin and client credentials of the session
// API. See docs/mcp/api.md.
type APIPort interface {
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
}

// PeerPort reads and changes the peers and the reserve gate. See docs/peers.md.
type PeerPort interface {
	Peers() PeersView
	PeerEndpoint() PeerEndpoint
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

// SharePort mints, lists, revokes, and watches the read-only shares. See
// docs/peers.md.
type SharePort interface {
	// ShareSession mints a read-only share for one session, and returns the share
	// and its link. A nil expiresHours takes the default; a value of zero or less
	// makes a share with no expiry. See docs/peers.md.
	ShareSession(session string, expiresHours *float64) (ShareCreated, error)
	// ListShares lists the active shares. It never returns a secret.
	ListShares() []ShareView
	// RevokeShare ends a share by id, and reports whether the share was there.
	RevokeShare(id string) (bool, error)
	// WatchShare attaches a read-only spectator session from a spectate link, and
	// returns the local name. See docs/peers.md.
	WatchShare(link string) (string, error)
}

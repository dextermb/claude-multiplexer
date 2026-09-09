package manager

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/peer"
	"github.com/dextermb/claude-multiplexer/internal/protocol"
	"github.com/dextermb/claude-multiplexer/internal/session"
	"github.com/dextermb/claude-multiplexer/internal/wire"
)

// remoteRetry is the pause before the pump re-opens a stream that dropped, so a
// peer that restarts does not draw a tight reconnect loop.
const remoteRetry = 2 * time.Second

// errUnknownPeer is the failure when a name does not match a configured peer.
var errUnknownPeer = errors.New("manager: unknown peer")

// remoteEntry is a streamed session: it runs on a peer, and the pump
// republishes the peer's stream to the local bus under localName, so the pane
// treats it as local. See docs/peers.md.
type remoteEntry struct {
	localName  string
	peer       string // the peer host name, shown as the Host of the session
	remoteName string // the name the session has on the peer
	client     *peer.Client

	lines  *lineBuffer
	cancel context.CancelFunc

	mu    sync.Mutex
	snap  session.Snapshot
	todos []protocol.Todo
}

func (r *remoteEntry) snapshot() session.Snapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.snap
}

func (r *remoteEntry) todoList() []protocol.Todo {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.todos
}

func (r *remoteEntry) set(snap session.Snapshot, todos []protocol.Todo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.snap = snap
	if todos != nil {
		r.todos = todos
	}
}

// PeerNames lists the configured peer hosts by name, in file order, for the
// new-session form. It reads no secret. See docs/peers.md.
func (m *Manager) PeerNames() []string {
	hosts := m.peerHosts()
	names := make([]string, 0, len(hosts))
	for _, host := range hosts {
		names = append(names, host.Name)
	}
	return names
}

// AttachRemoteByName starts a session on the named peer, so the TUI attaches a
// remote session without handling a credential. It fails when no peer has the
// name. See docs/peers.md.
func (m *Manager) AttachRemoteByName(name string, spec peer.CreateSpec) (string, error) {
	for _, host := range m.peerHosts() {
		if host.Name == name {
			return m.AttachRemote(host, spec)
		}
	}
	return "", fmt.Errorf("%w: %s", errUnknownPeer, name)
}

// AttachRemote starts a session on a peer, then streams it into this host under
// a local name. It returns the local name. The session appears as a streamed
// session, marked with the peer. See docs/peers.md.
func (m *Manager) AttachRemote(host config.PeerHost, spec peer.CreateSpec) (string, error) {
	client := peer.New(host)
	remoteName, err := client.CreateSession(context.Background(), spec)
	if err != nil {
		return "", err
	}
	local := m.attach(host.Name, client, remoteName, "")
	m.saveRemotes()
	return local, nil
}

// attach registers a streamed session and starts its pump. A non-empty
// localName is taken as given (a re-attach on start); an empty one takes a
// unique name. It makes no network call, so an unreachable peer still registers
// and the pump retries.
func (m *Manager) attach(peerName string, client *peer.Client, remoteName, localName string) string {
	m.mu.Lock()
	if localName == "" {
		localName = m.uniqueName(remoteName, "", false)
	}
	ctx, cancel := context.WithCancel(context.Background())
	re := &remoteEntry{
		localName:  localName,
		peer:       peerName,
		remoteName: remoteName,
		client:     client,
		lines:      newLineBuffer(m.opts.MaxLines),
		cancel:     cancel,
		snap:       session.Snapshot{Name: localName, State: session.StateStarting},
	}
	m.remotes[localName] = re
	m.remoteOrder = append(m.remoteOrder, localName)
	m.mu.Unlock()

	m.pumps.Add(1)
	go m.remotePump(ctx, re)
	return localName
}

// remotePump reads the peer's stream and republishes each event to the local
// bus. It reconnects when the stream drops, and stops when the session closes
// on the peer or the context is done. See docs/peers.md.
func (m *Manager) remotePump(ctx context.Context, re *remoteEntry) {
	defer m.pumps.Done()
	for {
		if ctx.Err() != nil {
			return
		}
		ch, err := re.client.Stream(ctx, re.remoteName)
		if err != nil {
			if !sleepCtx(ctx, remoteRetry) {
				return
			}
			continue
		}
		first := true
		for ev := range ch {
			m.applyRemote(re, ev, first)
			first = false
			if ev.Closed {
				m.detachRemote(re.localName)
				return
			}
		}
		if !sleepCtx(ctx, remoteRetry) {
			return
		}
	}
}

// applyRemote records a streamed event and republishes it to the local bus. The
// first event of each connection carries the whole line buffer, so it replaces
// the local buffer; a later event appends. See docs/peers.md.
func (m *Manager) applyRemote(re *remoteEntry, ev wire.Event, replace bool) {
	snap := fromWireSnapshot(ev.Snapshot, re.localName)
	re.set(snap, ev.Todos)
	local := Event{
		Session:    re.localName,
		Kind:       session.EventKind(ev.Kind),
		Lines:      ev.Lines,
		Partial:    ev.Partial,
		Snapshot:   snap,
		Closed:     ev.Closed,
		Questions:  ev.Questions,
		QuestionID: ev.QuestionID,
		Todos:      ev.Todos,
		Replace:    replace,
	}
	if replace {
		m.bus.publishReset(re.lines, ev.Lines, local)
	} else {
		m.bus.publishLines(re.lines, ev.Lines, local)
	}
}

// detachRemote stops a streamed session's pump and forgets it, so it leaves the
// sidebar when the session closes on the peer.
func (m *Manager) detachRemote(localName string) {
	m.mu.Lock()
	re, ok := m.remotes[localName]
	if ok {
		delete(m.remotes, localName)
		m.remoteOrder = removeName(m.remoteOrder, localName)
	}
	m.mu.Unlock()
	if !ok {
		return
	}
	re.cancel()
	m.saveRemotes()
}

// remote returns the streamed session of a name, or nil when the name is not a
// streamed session.
func (m *Manager) remote(name string) *remoteEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.remotes[name]
}

// RemoteSnapshots gives the snapshot of every streamed session, in order, so the
// interface draws a streamed session like a local one. See docs/peers.md.
func (m *Manager) RemoteSnapshots() []session.Snapshot {
	m.mu.Lock()
	res := make([]*remoteEntry, 0, len(m.remoteOrder))
	for _, name := range m.remoteOrder {
		if re, ok := m.remotes[name]; ok {
			res = append(res, re)
		}
	}
	m.mu.Unlock()
	out := make([]session.Snapshot, 0, len(res))
	for _, re := range res {
		out = append(out, re.snapshot())
	}
	return out
}

// Hosts reports the peer each streamed session runs on, keyed by the local name,
// so the sidebar sorts a streamed session under its peer. See docs/peers.md.
func (m *Manager) Hosts() map[string]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]string, len(m.remotes))
	for name, re := range m.remotes {
		out[name] = re.peer
	}
	return out
}

// Hosted reports which live sessions this host runs on behalf of a peer, so the
// sidebar sorts them under the hosted section. See docs/peers.md.
func (m *Manager) Hosted() map[string]bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]bool, len(m.entries))
	for name, item := range m.entries {
		if item.metaCopy().Hosted {
			out[name] = true
		}
	}
	return out
}

// Owners reports the client that owns each live hosted session, keyed by the
// local name, so the sidebar groups a hosted session under its client. See
// docs/peers.md.
func (m *Manager) Owners() map[string]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]string, len(m.entries))
	for name, item := range m.entries {
		meta := item.metaCopy()
		if meta.Hosted && meta.Owner != "" {
			out[name] = meta.Owner
		}
	}
	return out
}

// fromWireSnapshot converts a streamed snapshot back to a session snapshot,
// under the local name, so the pane reads it like a local one.
func fromWireSnapshot(s wire.Snapshot, localName string) session.Snapshot {
	out := session.Snapshot{
		Name:            localName,
		Title:           s.Title,
		Dir:             s.Dir,
		Model:           s.Model,
		PermissionMode:  s.PermissionMode,
		Effort:          s.Effort,
		State:           session.ParseState(s.State),
		ClaudeSessionID: s.ClaudeSessionID,
		Cost:            s.Cost,
		Turns:           s.Turns,
		Queued:          s.Queued,
		LastDuration:    time.Duration(s.LastDurationMS) * time.Millisecond,
		InputTokens:     s.InputTokens,
		OutputTokens:    s.OutputTokens,
		ContextTokens:   s.ContextTokens,
	}
	if s.Err != "" {
		out.Err = errors.New(s.Err)
	}
	for _, j := range s.Jobs {
		out.Jobs = append(out.Jobs, session.Job{
			ID:          j.ID,
			Description: j.Description,
			TaskType:    j.TaskType,
			Status:      session.ParseJobStatus(j.Status),
		})
	}
	return out
}

// sleepCtx waits for d, or returns false at once when the context is done.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

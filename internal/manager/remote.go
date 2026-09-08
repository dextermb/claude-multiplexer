package manager

import (
	"context"
	"errors"
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
	if replace {
		re.lines.reset(ev.Lines)
	} else {
		re.lines.append(ev.Lines)
	}
	snap := fromWireSnapshot(ev.Snapshot, re.localName)
	re.set(snap, ev.Todos)
	m.bus.Publish(Event{
		Session:    re.localName,
		Kind:       session.EventKind(ev.Kind),
		Lines:      ev.Lines,
		Partial:    ev.Partial,
		Snapshot:   snap,
		Closed:     ev.Closed,
		Questions:  ev.Questions,
		QuestionID: ev.QuestionID,
		Todos:      ev.Todos,
	})
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

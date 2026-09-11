package manager

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/peer"
)

// remoteLink is the record of one streamed or spectator session, so a restart
// re-attaches it. The peer keeps running the session, so this host re-opens the
// stream under the same local name. A spectator session records its share link
// in place of a peer, and re-attaches from it. See docs/peers.md.
type remoteLink struct {
	Peer       string `json:"peer"`
	RemoteName string `json:"remote_name"`
	LocalName  string `json:"local_name"`
	ReadOnly   bool   `json:"read_only,omitempty"`
	ShareLink  string `json:"share_link,omitempty"`
}

func remotesPath(root string) string {
	return filepath.Join(root, "remotes.json")
}

// saveRemotes writes the current streamed sessions to disk, in order, so a
// restart re-attaches them.
func (m *Manager) saveRemotes() {
	m.mu.Lock()
	links := make([]remoteLink, 0, len(m.remoteOrder))
	for _, name := range m.remoteOrder {
		if re, ok := m.remotes[name]; ok {
			links = append(links, remoteLink{
				Peer:       re.peer,
				RemoteName: re.remoteName,
				LocalName:  re.localName,
				ReadOnly:   re.readOnly,
				ShareLink:  re.shareLink,
			})
		}
	}
	m.mu.Unlock()

	path := remotesPath(m.opts.Root)
	if len(links) == 0 {
		_ = os.Remove(path)
		return
	}
	data, err := json.MarshalIndent(links, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, append(data, '\n'), 0o644)
}

// reattachRemotes re-opens the stream of every streamed session recorded before
// a restart. It matches each link to a peer host in the config, so it reaches
// the peer with the same credentials. A link whose peer is gone from the config
// is dropped. See docs/peers.md.
func (m *Manager) reattachRemotes() {
	data, err := os.ReadFile(remotesPath(m.opts.Root))
	if err != nil {
		return
	}
	var links []remoteLink
	if err := json.Unmarshal(data, &links); err != nil {
		return
	}

	hosts := map[string]config.PeerHost{}
	if cfg, err := config.Load(m.opts.ConfigPaths...); err == nil && cfg.Peers != nil {
		for _, host := range cfg.Peers.Hosts {
			hosts[host.Name] = host
		}
	}

	changed := false
	for _, link := range links {
		if link.ShareLink != "" {
			payload, id, err := parseSpectateLink(link.ShareLink)
			if err != nil {
				changed = true
				continue
			}
			m.attach(attachSpec{
				peer:       shareHost(payload.U),
				client:     peer.NewShare(payload.U, payload.T),
				remoteName: id,
				base:       spectatorBase,
				localName:  link.LocalName,
				readOnly:   true,
				shareLink:  link.ShareLink,
			})
			continue
		}
		host, ok := hosts[link.Peer]
		if !ok {
			changed = true
			continue
		}
		m.attach(attachSpec{peer: link.Peer, client: peer.New(host), remoteName: link.RemoteName, base: link.RemoteName, localName: link.LocalName})
	}
	if changed {
		m.saveRemotes()
	}
}

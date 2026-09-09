package manager

import (
	"context"
	"errors"
	"sync"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/mcp"
	"github.com/dextermb/claude-multiplexer/internal/peer"
	"github.com/dextermb/claude-multiplexer/internal/usage"
)

var (
	errBadWindow  = errors.New("manager: the reserve window must be 5h or 7d")
	errBadPercent = errors.New("manager: the reserve percent must be 0 to 100")
	errBadPort    = errors.New("manager: the peer port must be 1 to 65535")
)

// startUsage starts the usage poll when a fetch is wired, and runs it until the
// manager stops. A nil fetch keeps the poll off, so usage reads as unknown.
func (m *Manager) startUsage() {
	if m.opts.UsageFetch == nil {
		return
	}
	m.usagePoll = usage.NewPoller(m.opts.UsageFetch, 0)
	m.usagePoll.OnUpdate(m.evaluateReserve)
	ctx, cancel := context.WithCancel(context.Background())
	m.usageStop = cancel
	go m.usagePoll.Run(ctx)
}

// Usage returns this host's Claude usage from the poll cache. It reads as
// unknown when the poll is not wired. See docs/peers.md.
func (m *Manager) Usage() usage.Usage {
	if m.usagePoll == nil {
		return usage.Usage{Error: "usage poll not configured"}
	}
	return m.usagePoll.Usage()
}

// PeerUsage reads the usage of every configured peer, concurrently. A peer that
// is off or unreachable reports reachable=false with the error, and never
// blocks the others. See docs/peers.md.
func (m *Manager) PeerUsage(ctx context.Context) []mcp.PeerReport {
	hosts := m.peerHosts()
	reports := make([]mcp.PeerReport, len(hosts))
	var wg sync.WaitGroup
	for i, host := range hosts {
		wg.Add(1)
		go func(i int, host config.PeerHost) {
			defer wg.Done()
			report := mcp.PeerReport{Name: host.Name, URL: host.URL}
			got, err := peer.New(host).Usage(ctx)
			if err != nil {
				report.Error = err.Error()
			} else {
				report.Reachable = true
				report.Usage = got
			}
			reports[i] = report
		}(i, host)
	}
	wg.Wait()
	return reports
}

// peerHosts reads the configured peer hosts, or none when peering is off.
func (m *Manager) peerHosts() []config.PeerHost {
	cfg, err := config.Load(m.opts.ConfigPaths...)
	if err != nil || cfg.Peers == nil {
		return nil
	}
	return cfg.Peers.Hosts
}

// Peers reads the peer settings as a view with no secret. See docs/peers.md.
func (m *Manager) Peers() mcp.PeersView {
	cfg, err := config.Load(m.opts.ConfigPaths...)
	if err != nil {
		return mcp.PeersView{Hosts: []mcp.PeerHostView{}}
	}
	return peersView(cfg.Peers)
}

// EnablePeering turns the peer listener on. It binds 0.0.0.0 on the next
// restart. A port over zero sets the port; zero keeps the default.
func (m *Manager) EnablePeering(port int) (string, error) {
	if port < 0 || port > 65535 {
		return "", errBadPort
	}
	return m.mutatePeers(func(p *config.Peers) error {
		p.Enabled = true
		if port > 0 {
			p.Port = port
		}
		return nil
	})
}

// DisablePeering turns the peer listener off, and reports whether it was on.
func (m *Manager) DisablePeering() (string, bool, error) {
	return m.mutatePeersBool(func(p *config.Peers) bool {
		if !p.Enabled {
			return false
		}
		p.Enabled = false
		p.Port = 0
		return true
	})
}

// AddPeer adds a peer host, or replaces one with the same name.
func (m *Manager) AddPeer(in mcp.PeerHostInput) (string, error) {
	return m.mutatePeers(func(p *config.Peers) error {
		host := config.PeerHost{Name: in.Name, URL: in.URL, ClientID: in.ClientID, ClientSecret: in.ClientSecret}
		for i := range p.Hosts {
			if p.Hosts[i].Name == in.Name {
				p.Hosts[i] = host
				return nil
			}
		}
		p.Hosts = append(p.Hosts, host)
		return nil
	})
}

// RemovePeer removes a peer host by name, and reports whether it was there.
func (m *Manager) RemovePeer(name string) (string, bool, error) {
	return m.mutatePeersBool(func(p *config.Peers) bool {
		for i := range p.Hosts {
			if p.Hosts[i].Name == name {
				p.Hosts = append(p.Hosts[:i], p.Hosts[i+1:]...)
				return true
			}
		}
		return false
	})
}

// SetReserve sets the usage reserve: the window and the percent-remaining floor.
func (m *Manager) SetReserve(window string, minPercent int) (string, error) {
	if !config.ValidWindow(window) {
		return "", errBadWindow
	}
	if minPercent < 0 || minPercent > 100 {
		return "", errBadPercent
	}
	return m.mutatePeers(func(p *config.Peers) error {
		p.Reserve = &config.Reserve{Window: window, MinPercent: minPercent}
		return nil
	})
}

// UnsetReserve clears the usage reserve, and reports whether one was set.
func (m *Manager) UnsetReserve() (string, bool, error) {
	return m.mutatePeersBool(func(p *config.Peers) bool {
		if p.Reserve == nil {
			return false
		}
		p.Reserve = nil
		return true
	})
}

// mutatePeers loads the settings, changes the peers block, and writes it back,
// in the pattern of the layout tools. See docs/manager.md.
func (m *Manager) mutatePeers(fn func(*config.Peers) error) (string, error) {
	path := config.Target(m.opts.ConfigPaths...)
	if path == "" {
		return "", errNoSettings
	}
	current, err := config.Load(path)
	if err != nil {
		return "", err
	}
	if current.Peers == nil {
		current.Peers = &config.Peers{}
	}
	if err := fn(current.Peers); err != nil {
		return "", err
	}
	current.Peers = normalizePeers(current.Peers)
	if err := config.Write(path, current); err != nil {
		return "", err
	}
	return path, nil
}

// mutatePeersBool is mutatePeers for a change that may find nothing to do, so it
// reports whether it changed the settings.
func (m *Manager) mutatePeersBool(fn func(*config.Peers) bool) (string, bool, error) {
	path := config.Target(m.opts.ConfigPaths...)
	if path == "" {
		return "", false, errNoSettings
	}
	current, err := config.Load(path)
	if err != nil {
		return "", false, err
	}
	if current.Peers == nil || !fn(current.Peers) {
		return path, false, nil
	}
	current.Peers = normalizePeers(current.Peers)
	if err := config.Write(path, current); err != nil {
		return "", false, err
	}
	return path, true, nil
}

// normalizePeers drops an empty peers block to nil, so the file does not keep an
// empty object.
func normalizePeers(p *config.Peers) *config.Peers {
	if p == nil || (!p.Enabled && p.Reserve == nil && len(p.Hosts) == 0) {
		return nil
	}
	return p
}

func peersView(p *config.Peers) mcp.PeersView {
	if p == nil {
		return mcp.PeersView{Hosts: []mcp.PeerHostView{}}
	}
	view := mcp.PeersView{Enabled: p.Enabled, Port: p.Port, Hosts: make([]mcp.PeerHostView, 0, len(p.Hosts))}
	if p.Reserve != nil {
		view.Reserve = &mcp.ReserveView{Window: p.Reserve.Window, MinPercent: p.Reserve.MinPercent}
	}
	for _, h := range p.Hosts {
		view.Hosts = append(view.Hosts, mcp.PeerHostView{Name: h.Name, URL: h.URL, ClientID: h.ClientID})
	}
	return view
}

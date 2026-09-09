package manager

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
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

// PeerEndpoint names the address a peer on the network dials to reach this
// host. It reads the bound peer listener, then swaps the unspecified bind host
// (0.0.0.0) for a routable LAN address, so the URL is one a peer can use. It is
// empty when the peer listener is off. See docs/peers.md.
func (m *Manager) PeerEndpoint() mcp.PeerEndpoint {
	base := ""
	if m.mcp != nil {
		base = m.mcp.PeerBaseURL()
	}
	if base == "" {
		return mcp.PeerEndpoint{}
	}
	u, err := url.Parse(base)
	if err != nil {
		return mcp.PeerEndpoint{}
	}
	host := u.Hostname()
	if ip := net.ParseIP(host); ip != nil && ip.IsUnspecified() {
		if lan := localIPv4(); lan != "" {
			host = lan
		}
	}
	port, _ := strconv.Atoi(u.Port())
	return mcp.PeerEndpoint{
		URL:     "http://" + net.JoinHostPort(host, u.Port()),
		Host:    host,
		Port:    port,
		Enabled: true,
	}
}

// localIPv4 is the first routable IPv4 address of an interface that is up. It
// skips loopback and link-local, so the address is one a peer on the same
// network reaches. It is empty when no such interface exists.
func localIPv4() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipnet.IP.To4()
			if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}
			return ip.String()
		}
	}
	return ""
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
	cred, err := credentialFrom(in.CredentialType, in.Credential)
	if err != nil {
		return "", err
	}
	return m.mutatePeers(func(p *config.Peers) error {
		host := config.PeerHost{Name: in.Name, URL: in.URL, ClientID: in.ClientID, ClientSecret: in.ClientSecret, Credential: cred}
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

// UpdatePeer changes a peer host found by name. Each field changes only when
// non-empty, so a regenerated secret or a new url updates without re-supplying
// the rest. ClearCredential removes the lent credential. It fails when no peer
// has the name.
func (m *Manager) UpdatePeer(in mcp.PeerHostUpdate) (string, error) {
	cred, err := credentialFrom(in.CredentialType, in.Credential)
	if err != nil {
		return "", err
	}
	return m.mutatePeers(func(p *config.Peers) error {
		for i := range p.Hosts {
			if p.Hosts[i].Name != in.Name {
				continue
			}
			if in.URL != "" {
				p.Hosts[i].URL = in.URL
			}
			if in.ClientID != "" {
				p.Hosts[i].ClientID = in.ClientID
			}
			if in.ClientSecret != "" {
				p.Hosts[i].ClientSecret = in.ClientSecret
			}
			if in.ClearCredential {
				p.Hosts[i].Credential = nil
			} else if cred != nil {
				p.Hosts[i].Credential = cred
			}
			return nil
		}
		return fmt.Errorf("%w: %s", errUnknownPeer, in.Name)
	})
}

// credentialFrom builds a lent credential from a type and a value. An empty
// value gives no credential. A value with no valid type is an error, because a
// lent credential must say whether it is a token or a key. See
// docs/peers/hoisted.md.
func credentialFrom(ctype, value string) (*config.PeerCredential, error) {
	ctype = strings.TrimSpace(ctype)
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	if !config.ValidCredentialType(ctype) {
		return nil, errBadCredentialType
	}
	return &config.PeerCredential{Type: ctype, Value: value}, nil
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
		hv := mcp.PeerHostView{Name: h.Name, URL: h.URL, ClientID: h.ClientID}
		if h.Credential != nil {
			hv.Credential = &mcp.PeerCredentialView{Type: h.Credential.Type, Last4: credLast4(h.Credential.Value)}
		}
		view.Hosts = append(view.Hosts, hv)
	}
	return view
}

// credLast4 gives the last four characters of a credential value, so a view
// shows which credential a peer holds without the value.
func credLast4(value string) string {
	r := []rune(value)
	if len(r) <= 4 {
		return string(r)
	}
	return string(r[len(r)-4:])
}

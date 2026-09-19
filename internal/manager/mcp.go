package manager

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/dextermb/claude-multiplexer/internal/api"
	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/mcp"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

// StartMCP starts the server that serves the multiplexer's own tools to each
// session. Call it before the first Spawn, because a session that starts
// earlier never learns the address. See docs/mcp/transport.md.
func (m *Manager) StartMCP() error {
	if m.mcp != nil {
		return nil
	}
	store, err := api.Open(m.opts.Root)
	if err != nil {
		return err
	}
	m.apiStore = store
	server := mcp.NewServer(&bridge{m: m})
	server.EnableAPI(store, m.apiSessions)
	server.EnableShares(m.shareSessions)
	start, end := m.apiPortRange()
	if err := server.Start(start, end); err != nil {
		return err
	}
	m.mcp = server
	m.writeEndpoint()
	m.startUsage()
	cfg, err := config.Load(m.opts.ConfigPaths...)
	if err == nil && cfg.Peers != nil {
		if err := server.StartPeer(cfg.Peers.ListenAddr()); err != nil {
			return err
		}
	}
	m.reattachRemotes()
	return nil
}

// writeEndpoint records the base URL of the API, so a client reads the exact
// address after a restart moves the port. See docs/mcp/api.md.
func (m *Manager) writeEndpoint() {
	if m.mcp == nil {
		return
	}
	dir := filepath.Join(m.opts.Root, "api")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	data, err := json.Marshal(map[string]string{"url": m.mcp.BaseURL()})
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(dir, "endpoint.json"), append(data, '\n'), 0o600)
}

func (m *Manager) MCPURL() string {
	if m.mcp == nil {
		return ""
	}
	return m.mcp.URL()
}

// resolveProfile reads the profile of a new session: the one the spec names, or
// the defaultToolProfile setting, or the default. A name that does not parse
// takes the default, so a mistyped setting never stops a session starting. See
// docs/mcp/profiles.md.
func (m *Manager) resolveProfile(name string) mcp.Profile {
	if name == "" {
		if cfg, err := config.Load(m.opts.ConfigPaths...); err == nil {
			name = cfg.DefaultToolProfile
		}
	}
	profile, err := mcp.ParseProfile(name)
	if err != nil {
		return mcp.DefaultProfile
	}
	return profile
}

// equipTools gives one session its token, its configuration file, and the tool
// names it may call.
func (m *Manager) equipTools(cfg *session.Config, name string, profile mcp.Profile, control bool) (string, error) {
	if m.mcp == nil {
		return "", nil
	}
	token, err := m.mcp.Register(name, profile, control)
	if err != nil {
		return "", err
	}
	path := mcpConfigPath(m.opts.Root, name)
	data, err := m.mcp.Config(token)
	if err != nil {
		m.mcp.Unregister(token)
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		m.mcp.Unregister(token)
		return "", err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		m.mcp.Unregister(token)
		return "", err
	}
	cfg.ExtraArgs = append(cfg.ExtraArgs, "--mcp-config", path)
	cfg.AllowedTools = append(append([]string{}, cfg.AllowedTools...), mcp.AllowedTools(profile, control)...)
	if profile != mcp.ProfileMinimal && m.WorkItemsEnabled() {
		for _, tool := range mcp.WorkItemTools {
			cfg.AllowedTools = append(cfg.AllowedTools, mcp.Qualify(tool))
		}
	}
	if profile != mcp.ProfileMinimal && m.PullRequestsEnabled() {
		for _, tool := range mcp.PullRequestTools {
			cfg.AllowedTools = append(cfg.AllowedTools, mcp.Qualify(tool))
		}
	}
	return token, nil
}

func (m *Manager) releaseTools(token string) {
	if m.mcp == nil || token == "" {
		return
	}
	m.mcp.Unregister(token)
}

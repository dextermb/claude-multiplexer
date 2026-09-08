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
	start, end := m.apiPortRange()
	if err := server.Start(start, end); err != nil {
		return err
	}
	m.mcp = server
	m.writeEndpoint()
	m.startUsage()
	cfg, err := config.Load(m.opts.ConfigPaths...)
	if err == nil && cfg.Peers != nil {
		if err := server.StartPeer(cfg.Peers.Listen); err != nil {
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

// equipTools gives one session its token, its configuration file, and the tool
// names it may call.
func (m *Manager) equipTools(cfg *session.Config, name string, control bool) (string, error) {
	if m.mcp == nil {
		return "", nil
	}
	token, err := m.mcp.Register(name, control)
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
	cfg.AllowedTools = append(append([]string{}, cfg.AllowedTools...), mcp.AllowedTools(control)...)
	return token, nil
}

func (m *Manager) releaseTools(token string) {
	if m.mcp == nil || token == "" {
		return
	}
	m.mcp.Unregister(token)
}

package mcp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/api"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	Path    = "/mcp"
	version = "0.1.0"

	// The default loopback port range the API binds inside. See docs/mcp/api.md.
	DefaultAPIPortStart = 51890
	DefaultAPIPortEnd   = 51899
)

// Server serves one MCP endpoint for every session, and the session API for a
// program outside the multiplexer. The bearer token on a request names the
// caller, so a tool knows who asked. See docs/mcp/api.md.
type Server struct {
	sessions Sessions

	store   *api.Store
	resolve func(clientID, clientName string) APISessions

	mu        sync.Mutex
	servers   map[string]*sdk.Server
	apiTokens map[string]apiGrant

	ln   net.Listener
	http *http.Server
}

func NewServer(sessions Sessions) *Server {
	return &Server{
		sessions:  sessions,
		servers:   make(map[string]*sdk.Server),
		apiTokens: make(map[string]apiGrant),
	}
}

// EnableAPI gives the server the credential store and the owner-scoped session
// resolver, so it can serve the token, admin, and REST surfaces. Call it before
// Start.
func (s *Server) EnableAPI(store *api.Store, resolve func(clientID, clientName string) APISessions) {
	s.store = store
	s.resolve = resolve
}

// Start binds the first free port in the range, on loopback. It fails only when
// the whole range is taken. Call it before the first session starts, because a
// session needs the address for its configuration file. See docs/mcp/api.md.
func (s *Server) Start(portStart, portEnd int) error {
	ln, err := listenRange(portStart, portEnd)
	if err != nil {
		return err
	}
	s.ln = ln

	mux := http.NewServeMux()
	mux.Handle(Path, s.handler())
	if s.store != nil {
		s.mountAPI(mux)
	}
	s.http = &http.Server{Handler: mux}
	go func() { _ = s.http.Serve(ln) }()
	return nil
}

func listenRange(portStart, portEnd int) (net.Listener, error) {
	if portStart <= 0 {
		return net.Listen("tcp", "127.0.0.1:0")
	}
	if portEnd < portStart {
		portEnd = portStart
	}
	var last error
	for port := portStart; port <= portEnd; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			return ln, nil
		}
		last = err
	}
	return nil, fmt.Errorf("mcp: no free port in %d-%d: %w", portStart, portEnd, last)
}

func (s *Server) Addr() string {
	if s.ln == nil {
		return ""
	}
	return s.ln.Addr().String()
}

func (s *Server) URL() string {
	if s.ln == nil {
		return ""
	}
	return "http://" + s.Addr() + Path
}

// BaseURL is the address of the API, without the MCP path. A client reads it to
// reach the token, admin, and REST surfaces.
func (s *Server) BaseURL() string {
	if s.ln == nil {
		return ""
	}
	return "http://" + s.Addr()
}

func (s *Server) Close(ctx context.Context) error {
	if s.http == nil {
		return nil
	}
	return s.http.Shutdown(ctx)
}

// Register gives a session its own token and its own tool set. A session
// without the control grant is never offered the tools it may not call.
func (s *Server) Register(name string, control bool) (string, error) {
	token, err := newToken()
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	s.servers[token] = s.build(name, control)
	s.mu.Unlock()
	return token, nil
}

func (s *Server) Unregister(token string) {
	if token == "" {
		return
	}
	s.mu.Lock()
	delete(s.servers, token)
	s.mu.Unlock()
}

// Config is the --mcp-config document for one session.
func (s *Server) Config(token string) ([]byte, error) {
	doc := map[string]any{
		"mcpServers": map[string]any{
			ServerName: map[string]any{
				"type":    "http",
				"url":     s.URL(),
				"headers": map[string]string{"Authorization": "Bearer " + token},
			},
		},
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func (s *Server) lookup(token string) *sdk.Server {
	if token == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if grant, ok := s.apiTokens[token]; ok && time.Now().After(grant.expiry) {
		delete(s.apiTokens, token)
		delete(s.servers, token)
		return nil
	}
	return s.servers[token]
}

func (s *Server) handler() http.Handler {
	inner := sdk.NewStreamableHTTPHandler(func(r *http.Request) *sdk.Server {
		return s.lookup(bearer(r))
	}, nil)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.lookup(bearer(r)) == nil {
			http.Error(w, "unknown session token", http.StatusUnauthorized)
			return
		}
		inner.ServeHTTP(w, r)
	})
}

func bearer(r *http.Request) string {
	value := r.Header.Get("Authorization")
	if value == "" {
		return ""
	}
	if token, ok := strings.CutPrefix(value, "Bearer "); ok {
		return strings.TrimSpace(token)
	}
	return ""
}

func newToken() (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("mcp: token: %w", err)
	}
	return hex.EncodeToString(raw), nil
}

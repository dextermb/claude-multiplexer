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

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	Path    = "/mcp"
	version = "0.1.0"
)

// Server serves one MCP endpoint for every session. The bearer token on a
// request names the calling session, so a tool knows who asked.
type Server struct {
	sessions Sessions

	mu      sync.Mutex
	servers map[string]*sdk.Server

	ln   net.Listener
	http *http.Server
}

func NewServer(sessions Sessions) *Server {
	return &Server{sessions: sessions, servers: make(map[string]*sdk.Server)}
}

// Start listens on a free loopback port. Call it before the first session
// starts, because a session needs the address for its configuration file.
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("mcp: listen: %w", err)
	}
	s.ln = ln

	mux := http.NewServeMux()
	mux.Handle(Path, s.handler())
	s.http = &http.Server{Handler: mux}
	go func() { _ = s.http.Serve(ln) }()
	return nil
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

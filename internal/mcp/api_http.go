package mcp

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

// apiTokenTTL is how long an access token lives. A client runs the grant again
// when it expires. See docs/mcp/api.md.
const apiTokenTTL = time.Hour

// apiGrant is one live access token: the client it belongs to, its owner-scoped
// session view, and when it expires.
type apiGrant struct {
	clientID   string
	clientName string
	sess       APISessions
	expiry     time.Time
}

// mountAPI adds the token, admin, and REST surfaces to the mux, next to the MCP
// endpoint. See docs/mcp/api.md.
func (s *Server) mountAPI(mux *http.ServeMux) {
	mux.HandleFunc("/token", s.handleToken)
	mux.HandleFunc("/admin/clients", s.handleClients)
	mux.HandleFunc("/admin/clients/", s.handleClient)
	mux.HandleFunc("/api/usage", s.handleUsage)
	mux.HandleFunc("/api/", s.handleAPI)
}

// RevokeClientTokens drops every access token of a client, so a rotate or a
// revoke takes effect at once. See docs/mcp/api.md.
func (s *Server) RevokeClientTokens(clientID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for token, grant := range s.apiTokens {
		if grant.clientID == clientID {
			delete(s.apiTokens, token)
			delete(s.servers, token)
		}
	}
}

func (s *Server) apiGrantFor(token string) (apiGrant, bool) {
	if token == "" {
		return apiGrant{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	grant, ok := s.apiTokens[token]
	if !ok {
		return apiGrant{}, false
	}
	if time.Now().After(grant.expiry) {
		delete(s.apiTokens, token)
		delete(s.servers, token)
		return apiGrant{}, false
	}
	return grant, true
}

func writeJSON(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	_, _ = w.Write(append(data, '\n'))
}

func writeError(w http.ResponseWriter, code int, message string) {
	writeJSON(w, code, map[string]string{"error": message})
}

func writeAPIError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "no such session")
		return
	}
	writeError(w, http.StatusBadRequest, err.Error())
}

func queryInt(r *http.Request, key string) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return 0
	}
	n := 0
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return 0
		}
		n = n*10 + int(ch-'0')
	}
	return n
}

package mcp

import "net/http"

// handleUsage serves this host's Claude usage to a peer. It needs a valid access
// token, the same grant the session REST uses. See docs/peers.md.
func (s *Server) handleUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "use GET")
		return
	}
	if _, ok := s.apiGrantFor(bearer(r)); !ok {
		writeError(w, http.StatusUnauthorized, "access token required")
		return
	}
	writeJSON(w, http.StatusOK, s.sessions.Usage())
}

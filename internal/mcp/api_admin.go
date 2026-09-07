package mcp

import (
	"encoding/json"
	"net/http"
	"strings"
)

// handleClients serves the collection route: list the clients, or create one.
// The admin secret gates it. See docs/mcp/api.md.
func (s *Server) handleClients(w http.ResponseWriter, r *http.Request) {
	if !s.adminOK(r) {
		writeError(w, http.StatusUnauthorized, "admin secret required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"clients": s.store.ListClients()})
	case http.MethodPost:
		var body struct {
			Name string `json:"name"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		client, secret, err := s.store.CreateClient(body.Name)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"client_id":     client.ClientID,
			"name":          client.Name,
			"client_secret": secret,
		})
	default:
		writeError(w, http.StatusMethodNotAllowed, "use GET or POST")
	}
}

// handleClient serves one client: change it, rotate its secret, or remove it.
func (s *Server) handleClient(w http.ResponseWriter, r *http.Request) {
	if !s.adminOK(r) {
		writeError(w, http.StatusUnauthorized, "admin secret required")
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/admin/clients/")
	id, action, _ := strings.Cut(rest, "/")
	if id == "" {
		writeError(w, http.StatusNotFound, "no client id")
		return
	}
	switch {
	case action == "rotate" && r.Method == http.MethodPost:
		secret, err := s.store.RotateClient(id)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		s.RevokeClientTokens(id)
		writeJSON(w, http.StatusOK, map[string]string{"client_id": id, "client_secret": secret})
	case action == "" && r.Method == http.MethodPatch:
		var body struct {
			Name     *string `json:"name"`
			Disabled *bool   `json:"disabled"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		client, err := s.store.UpdateClient(id, body.Name, body.Disabled)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if body.Disabled != nil && *body.Disabled {
			s.RevokeClientTokens(id)
		}
		writeJSON(w, http.StatusOK, client)
	case action == "" && r.Method == http.MethodDelete:
		if err := s.store.RevokeClient(id); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		s.RevokeClientTokens(id)
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	default:
		writeError(w, http.StatusMethodNotAllowed, "unsupported method")
	}
}

func (s *Server) adminOK(r *http.Request) bool {
	return s.store.VerifyAdmin(bearer(r))
}

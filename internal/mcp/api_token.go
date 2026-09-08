package mcp

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// handleToken runs the client-credentials grant. It hashes the secret, mints a
// random access token, and builds the owner-scoped tool set. See docs/mcp/api.md.
func (s *Server) handleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	clientID, secret, grantType := readGrant(r)
	if grantType != "client_credentials" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported_grant_type"})
		return
	}
	client, err := s.store.VerifyClient(clientID, secret)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_client"})
		return
	}
	token, err := newToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token")
		return
	}
	sess := s.resolve(client.ClientID, client.Name)
	s.mu.Lock()
	s.servers[token] = buildAPI(client.Name, sess)
	s.apiTokens[token] = apiGrant{
		clientID:   client.ClientID,
		clientName: client.Name,
		sess:       sess,
		expiry:     time.Now().Add(apiTokenTTL),
	}
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, tokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int(apiTokenTTL.Seconds()),
	})
}

func readGrant(r *http.Request) (id, secret, grantType string) {
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		var body struct {
			GrantType    string `json:"grant_type"`
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
		}
		if json.NewDecoder(r.Body).Decode(&body) == nil {
			return body.ClientID, body.ClientSecret, body.GrantType
		}
		return "", "", ""
	}
	_ = r.ParseForm()
	return r.PostForm.Get("client_id"), r.PostForm.Get("client_secret"), r.PostForm.Get("grant_type")
}

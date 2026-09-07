package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
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

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

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

// handleClients serves the collection route: list the clients, or create one.
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

// handleAPI serves the REST surface. A client reaches only the sessions it owns,
// so a route on any other session answers 404. See docs/mcp/api.md.
func (s *Server) handleAPI(w http.ResponseWriter, r *http.Request) {
	grant, ok := s.apiGrantFor(bearer(r))
	if !ok {
		writeError(w, http.StatusUnauthorized, "access token required")
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/api/")
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) == 0 || parts[0] != "sessions" {
		writeError(w, http.StatusNotFound, "unknown route")
		return
	}
	sess := grant.sess
	switch {
	case len(parts) == 1 && r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, listOut{Sessions: sess.List()})
	case len(parts) == 1 && r.Method == http.MethodPost:
		s.restCreate(w, r, sess, grant.clientName)
	case len(parts) == 2 && r.Method == http.MethodPatch:
		s.restRename(w, r, sess, parts[1])
	case len(parts) == 3 && parts[2] == "messages" && r.Method == http.MethodGet:
		s.restMessages(w, r, sess, parts[1])
	case len(parts) == 3 && parts[2] == "jobs" && r.Method == http.MethodGet:
		s.restJobs(w, sess, parts[1])
	case len(parts) == 3 && parts[2] == "message" && r.Method == http.MethodPost:
		s.restMessage(w, r, sess, parts[1], grant.clientName)
	case len(parts) == 3 && parts[2] == "stop" && r.Method == http.MethodPost:
		s.restStop(w, r, sess, parts[1], grant.clientName)
	case len(parts) == 3 && parts[2] == "archive" && r.Method == http.MethodPost:
		s.restArchive(w, r, sess, parts[1], grant.clientName)
	case len(parts) == 5 && parts[2] == "jobs" && parts[4] == "stop" && r.Method == http.MethodPost:
		s.restStopJob(w, sess, parts[1], parts[3], grant.clientName)
	default:
		writeError(w, http.StatusNotFound, "unknown route")
	}
}

func (s *Server) restCreate(w http.ResponseWriter, r *http.Request, sess APISessions, by string) {
	var body struct {
		Dir  string `json:"dir"`
		Name string `json:"name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if strings.TrimSpace(body.Dir) == "" {
		writeError(w, http.StatusBadRequest, "dir required")
		return
	}
	name, err := sess.Create(body.Dir, strings.TrimSpace(body.Name), by)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"name": name})
}

func (s *Server) restRename(w http.ResponseWriter, r *http.Request, sess APISessions, name string) {
	var body struct {
		Title string `json:"title"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if err := sess.SetTitle(name, body.Title); err != nil {
		writeAPIError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) restMessages(w http.ResponseWriter, r *http.Request, sess APISessions, name string) {
	limit := clampLimit(queryInt(r, "limit"))
	messages, err := sess.Messages(name, limit)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, messagesOut{Session: name, Messages: messages})
}

func (s *Server) restJobs(w http.ResponseWriter, sess APISessions, name string) {
	jobs, err := sess.Jobs(name)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, listJobsOut{Session: name, Jobs: jobs})
}

func (s *Server) restMessage(w http.ResponseWriter, r *http.Request, sess APISessions, name, by string) {
	var body struct {
		Text string `json:"text"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	queued, err := sess.SendFrom(name, by, body.Text)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sendOut{OK: true, Queued: queued, Message: "queued for " + name})
}

func (s *Server) restStop(w http.ResponseWriter, r *http.Request, sess APISessions, name, by string) {
	if err := sess.Stop(r.Context(), name, by); err != nil {
		writeAPIError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, okOut{OK: true, Message: name + " is stopped"})
}

func (s *Server) restArchive(w http.ResponseWriter, r *http.Request, sess APISessions, name, by string) {
	var body struct {
		Restore bool `json:"restore"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if err := sess.Archive(name, !body.Restore, by); err != nil {
		writeAPIError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, okOut{OK: true, Message: name + " is archived"})
}

func (s *Server) restStopJob(w http.ResponseWriter, sess APISessions, name, job, by string) {
	queued, err := sess.StopJob(name, job, by)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stopJobOut{OK: true, Queued: queued, Message: "stopping job " + job})
}

// buildAPI registers the session-only tool set an external client reaches over
// MCP. It never registers a config, layout, or schedule tool. See docs/mcp/api.md.
func buildAPI(clientName string, sess APISessions) *sdk.Server {
	server := sdk.NewServer(&sdk.Implementation{Name: ServerName, Version: version}, nil)

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolList,
		Description: "List every session this client owns, running and stored.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in listIn) (*sdk.CallToolResult, listOut, error) {
		all := sess.List()
		out := make([]Session, 0, len(all))
		for _, item := range all {
			if in.LiveOnly && !item.Live {
				continue
			}
			out = append(out, item)
		}
		return nil, listOut{Sessions: out}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolMessages,
		Description: "Read the recent messages of a session this client owns, oldest first.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in messagesIn) (*sdk.CallToolResult, messagesOut, error) {
		target, err := cleanTarget(in.Session)
		if err != nil {
			return nil, messagesOut{}, err
		}
		messages, err := sess.Messages(target, clampLimit(in.Limit))
		if err != nil {
			return nil, messagesOut{}, err
		}
		return nil, messagesOut{Session: target, Messages: messages}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolListJobs,
		Description: "List the background jobs of a session this client owns.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in listJobsIn) (*sdk.CallToolResult, listJobsOut, error) {
		target, err := cleanTarget(in.Session)
		if err != nil {
			return nil, listJobsOut{}, err
		}
		jobs, err := sess.Jobs(target)
		if err != nil {
			return nil, listJobsOut{}, err
		}
		return nil, listJobsOut{Session: target, Jobs: jobs}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolRename,
		Description: "Set the display title of a session this client owns.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in apiRenameIn) (*sdk.CallToolResult, okOut, error) {
		target, err := cleanTarget(in.Session)
		if err != nil {
			return nil, okOut{}, err
		}
		if err := sess.SetTitle(target, in.Title); err != nil {
			return nil, okOut{}, err
		}
		return nil, okOut{OK: true, Message: target + " is renamed"}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolSend,
		Description: "Queue a prompt for a session this client owns. Read the answer later with " + ToolMessages + ".",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in sendIn) (*sdk.CallToolResult, sendOut, error) {
		target, err := cleanTarget(in.Session)
		if err != nil {
			return nil, sendOut{}, err
		}
		queued, err := sess.SendFrom(target, clientName, in.Text)
		if err != nil {
			return nil, sendOut{}, err
		}
		return nil, sendOut{OK: true, Queued: queued, Message: "queued for " + target}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolStop,
		Description: "End a session this client owns in a clean way. Its transcript is kept.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in targetIn) (*sdk.CallToolResult, okOut, error) {
		target, err := cleanTarget(in.Session)
		if err != nil {
			return nil, okOut{}, err
		}
		if err := sess.Stop(ctx, target, clientName); err != nil {
			return nil, okOut{}, err
		}
		return nil, okOut{OK: true, Message: target + " is stopped"}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolArchive,
		Description: "Take a stopped session this client owns out of the list, or with restore, bring it back.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in archiveIn) (*sdk.CallToolResult, okOut, error) {
		target, err := cleanTarget(in.Session)
		if err != nil {
			return nil, okOut{}, err
		}
		if err := sess.Archive(target, !in.Restore, clientName); err != nil {
			return nil, okOut{}, err
		}
		if in.Restore {
			return nil, okOut{OK: true, Message: target + " is back in the list"}, nil
		}
		return nil, okOut{OK: true, Message: target + " is archived"}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolCreate,
		Description: "Start a new session in a directory. This client owns the session it creates. It returns the name the session takes.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in createIn) (*sdk.CallToolResult, createOut, error) {
		path := strings.TrimSpace(in.Path)
		if path == "" {
			return nil, createOut{}, ErrNoPath
		}
		created, err := sess.Create(path, strings.TrimSpace(in.Name), clientName)
		if err != nil {
			return nil, createOut{}, err
		}
		return nil, createOut{OK: true, Name: created, Message: "started " + created}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolStopJob,
		Description: "Stop a running background job of a session this client owns. Give the session name and the job id.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in stopJobIn) (*sdk.CallToolResult, stopJobOut, error) {
		target, err := cleanTarget(in.Session)
		if err != nil {
			return nil, stopJobOut{}, err
		}
		job := strings.TrimSpace(in.Job)
		if job == "" {
			return nil, stopJobOut{}, ErrNoJob
		}
		queued, err := sess.StopJob(target, job, clientName)
		if err != nil {
			return nil, stopJobOut{}, err
		}
		return nil, stopJobOut{OK: true, Queued: queued, Message: "stopping job " + job + " of " + target}, nil
	})

	return server
}

type apiRenameIn struct {
	Session string `json:"session" jsonschema:"the name of the session to rename"`
	Title   string `json:"title" jsonschema:"the new display title; an empty string clears it"`
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

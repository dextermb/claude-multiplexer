package mcp

import (
	"encoding/json"
	"net/http"
	"strings"
)

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
	case len(parts) == 3 && parts[2] == "stream" && r.Method == http.MethodGet:
		s.restStream(w, r, sess, parts[1])
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
		Dir            string `json:"dir"`
		Name           string `json:"name"`
		Model          string `json:"model"`
		PermissionMode string `json:"permission_mode"`
		Effort         string `json:"effort"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if strings.TrimSpace(body.Dir) == "" {
		writeError(w, http.StatusBadRequest, "dir required")
		return
	}
	name, err := sess.Create(CreateInput{
		Dir:            body.Dir,
		Name:           strings.TrimSpace(body.Name),
		Model:          strings.TrimSpace(body.Model),
		PermissionMode: strings.TrimSpace(body.PermissionMode),
		Effort:         strings.TrimSpace(body.Effort),
	}, by)
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

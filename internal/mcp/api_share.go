package mcp

import (
	"context"
	"net/http"
	"strings"
	"time"
)

// shareRecheck is how often a live share stream re-checks that its share is
// still there and not expired, so a revoke ends the stream soon after. See
// docs/peers.md.
const shareRecheck = 3 * time.Second

// shareInfoOut is the label a spectator reads before the stream connects: the
// session title and the model. It carries no host-local session name. See
// docs/peers.md.
type shareInfoOut struct {
	Share string `json:"share"`
	Title string `json:"title,omitempty"`
	Model string `json:"model,omitempty"`
	State string `json:"state,omitempty"`
}

// handleShare serves the read-only share surface on the peer listener. The share
// token authenticates the request, and the share opens exactly one session's
// read routes. The spectator never learns the host-local session name. See
// docs/peers.md.
func (s *Server) handleShare(w http.ResponseWriter, r *http.Request) {
	if s.store == nil || s.resolveShare == nil {
		writeError(w, http.StatusNotFound, "unknown route")
		return
	}
	id, secret, ok := strings.Cut(bearer(r), ".")
	if !ok || id == "" || secret == "" {
		writeError(w, http.StatusUnauthorized, "share token required")
		return
	}
	share, err := s.store.VerifyShare(id, secret)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid share")
		return
	}
	sess := s.resolveShare(share.Session)

	rest := strings.TrimPrefix(r.URL.Path, "/api/shares/")
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	switch {
	case len(parts) == 1 && r.Method == http.MethodGet:
		s.shareInfo(w, sess, share.ID, share.Session)
	case len(parts) == 2 && parts[1] == "stream" && r.Method == http.MethodGet:
		s.shareStream(w, r, sess, share.ID, share.Session)
	case len(parts) == 2 && parts[1] == "messages" && r.Method == http.MethodGet:
		s.restMessages(w, r, sess, share.Session, share.ID)
	default:
		writeError(w, http.StatusNotFound, "unknown route")
	}
}

// shareInfo reports the session label a spectator draws, read from the one
// session the share view lists.
func (s *Server) shareInfo(w http.ResponseWriter, sess APISessions, shareID, session string) {
	out := shareInfoOut{Share: shareID}
	for _, item := range sess.List() {
		if item.Name == session {
			out.Title = item.Title
			out.Model = item.Model
			out.State = item.State
			break
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// shareStream serves the session's events read-only, and ends the stream when
// the share is revoked or expires. It re-checks the share on an interval,
// because the initial token check does not see a later revoke. See docs/peers.md.
func (s *Server) shareStream(w http.ResponseWriter, r *http.Request, sess APISessions, shareID, session string) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	go func() {
		ticker := time.NewTicker(shareRecheck)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if !s.store.ShareActive(shareID) {
					cancel()
					return
				}
			}
		}
	}()
	s.restStream(w, r.WithContext(ctx), sess, session, shareID)
}

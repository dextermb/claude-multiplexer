package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// restStream serves a session's events as server-sent events: the replay of the
// current lines first, then the live events, each as one `data:` line. The peer
// decodes them into a remote session. See docs/peers.md.
func (s *Server) restStream(w http.ResponseWriter, r *http.Request, sess APISessions, name string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "the stream needs a flushing writer")
		return
	}
	ch, err := sess.Stream(r.Context(), name)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

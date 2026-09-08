// Package usage reads the Claude usage-limit stat, the one Claude Code shows in
// /usage. It parses the anthropic-ratelimit-unified-* response headers into a
// Usage, and a poller keeps a cached copy. See docs/peers.md.
package usage

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// headerPrefix is the common prefix of the unified rate-limit headers. There is
// no official usage endpoint yet, so the parse is defensive: a header that is
// not there leaves its field unknown, never zero. See docs/peers.md.
const headerPrefix = "anthropic-ratelimit-unified-"

// Window is one usage window. A nil Remaining or ResetAt means the header for it
// was not there, so the value is unknown.
type Window struct {
	Status    string     `json:"status,omitempty"`
	Remaining *int       `json:"remaining,omitempty"`
	ResetAt   *time.Time `json:"reset_at,omitempty"`
}

// Known reports whether the window carries a remaining percent to compare
// against a reserve floor.
func (w Window) Known() bool {
	return w.Remaining != nil
}

// Usage is the parsed usage stat: the two windows, and whether the last read
// succeeded. When OK is false, the windows are the last-known values, or empty.
type Usage struct {
	FiveHour  Window    `json:"five_hour"`
	Weekly    Window    `json:"weekly"`
	FetchedAt time.Time `json:"fetched_at"`
	OK        bool      `json:"ok"`
	Error     string    `json:"error,omitempty"`
}

// Parse reads the unified headers into the two windows. It never fails: a header
// that is not there, or does not parse, leaves its field unknown.
func Parse(h http.Header) Usage {
	return Usage{
		FiveHour: parseWindow(h, "5h"),
		Weekly:   parseWindow(h, "7d"),
		OK:       true,
	}
}

func parseWindow(h http.Header, window string) Window {
	return Window{
		Status:    strings.TrimSpace(h.Get(headerPrefix + window + "-status")),
		Remaining: parsePercent(h.Get(headerPrefix + window + "-remaining")),
		ResetAt:   parseReset(h.Get(headerPrefix + window + "-reset")),
	}
}

func parsePercent(value string) *int {
	value = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(value), "%"))
	if value == "" {
		return nil
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return nil
	}
	return &n
}

// parseReset reads a reset value as an RFC3339 time, and then as unix seconds,
// so it takes either form the header may hold.
func parseReset(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return &t
	}
	if secs, err := strconv.ParseInt(value, 10, 64); err == nil {
		t := time.Unix(secs, 0).UTC()
		return &t
	}
	return nil
}

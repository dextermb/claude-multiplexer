package config

import (
	"errors"
	"strings"
	"time"
)

// The windows a last-active limit accepts. LastActiveUnset removes the limit.
// These are rolling lookbacks (1d is the last 24 hours), not the calendar
// buckets of CostWindow. See docs/config.md.
const (
	LastActiveDay   = "1d"
	LastActiveWeek  = "1w"
	LastActiveMonth = "1m"
	LastActiveYear  = "1y"
	LastActiveUnset = "unset"
)

// DefaultLastActive is the window a caller takes when it names none.
const DefaultLastActive = LastActiveDay

// ErrBadLastActive is the failure when a window is not a known one.
var ErrBadLastActive = errors.New("config: last active window must be 1d, 1w, 1m, 1y, or unset")

// ParseLastActive turns a window into a lookback duration. An empty window takes
// the default of one day, and "unset" returns zero, which means no limit.
func ParseLastActive(window string) (time.Duration, error) {
	switch strings.TrimSpace(window) {
	case "", LastActiveDay:
		return 24 * time.Hour, nil
	case LastActiveWeek:
		return 7 * 24 * time.Hour, nil
	case LastActiveMonth:
		return 30 * 24 * time.Hour, nil
	case LastActiveYear:
		return 365 * 24 * time.Hour, nil
	case LastActiveUnset:
		return 0, nil
	}
	return 0, ErrBadLastActive
}

// ArchivedWindow reads the archivedWindow setting and gives the value the
// archived list uses. An empty or unknown value takes DefaultLastActive, the
// same way CostWindow takes its default. See docs/config.md.
func ArchivedWindow(value string) string {
	clean := strings.ToLower(strings.TrimSpace(value))
	if _, err := ParseLastActive(clean); err != nil {
		return DefaultLastActive
	}
	if clean == "" {
		return DefaultLastActive
	}
	return clean
}

// ArchivedCutoff turns a window into the earliest last-active time an archived
// session may have and still show. The bool is false when the window is "unset",
// so there is no limit. An unknown window takes the default.
func ArchivedCutoff(window string, now time.Time) (time.Time, bool) {
	d, err := ParseLastActive(ArchivedWindow(window))
	if err != nil || d == 0 {
		return time.Time{}, false
	}
	return now.Add(-d), true
}

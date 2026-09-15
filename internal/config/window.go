package config

import (
	"strconv"
	"strings"
	"time"
)

// DefaultCostWindow is the window the status bar total takes when the settings
// name none: the current UTC day. See docs/cost.md.
const DefaultCostWindow = "1d"

// CostWindowAll is the value that counts the whole history, with no window.
const CostWindowAll = "all"

// CostWindow reads the window value the settings name, and gives back the
// value the rest of the program uses. An empty or unknown value takes
// DefaultCostWindow, the same way an unknown context action takes its default.
func CostWindow(value string) string {
	clean := strings.ToLower(strings.TrimSpace(value))
	if clean == CostWindowAll {
		return CostWindowAll
	}
	if _, _, ok := parseWindow(clean); !ok {
		return DefaultCostWindow
	}
	return clean
}

// WindowStart gives the first instant the window covers, and reports whether a
// window applies at all. The window is a calendar bucket in UTC: a count of 1
// starts at the top of the current day, week, or month, and a higher count adds
// that many buckets before it. See docs/cost.md.
func WindowStart(value string, now time.Time) (time.Time, bool) {
	clean := CostWindow(value)
	if clean == CostWindowAll {
		return time.Time{}, false
	}
	count, unit, ok := parseWindow(clean)
	if !ok {
		return time.Time{}, false
	}
	now = now.UTC()
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	switch unit {
	case 'd':
		return day.AddDate(0, 0, -(count - 1)), true
	case 'w':
		weekday := (int(day.Weekday()) + 6) % 7
		return day.AddDate(0, 0, -weekday-7*(count-1)), true
	case 'm':
		first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		return first.AddDate(0, -(count - 1), 0), true
	}
	return time.Time{}, false
}

func parseWindow(value string) (int, byte, bool) {
	if len(value) < 2 {
		return 0, 0, false
	}
	unit := value[len(value)-1]
	if unit != 'd' && unit != 'w' && unit != 'm' {
		return 0, 0, false
	}
	count, err := strconv.Atoi(value[:len(value)-1])
	if err != nil || count < 1 {
		return 0, 0, false
	}
	return count, unit, true
}

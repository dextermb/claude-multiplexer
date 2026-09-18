package mcp

import (
	"testing"
	"time"
)

func TestParseLastActive(t *testing.T) {
	cases := []struct {
		window string
		want   time.Duration
		bad    bool
	}{
		{"", 24 * time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"1w", 7 * 24 * time.Hour, false},
		{"1m", 30 * 24 * time.Hour, false},
		{"1y", 365 * 24 * time.Hour, false},
		{"unset", 0, false},
		{" 1w ", 7 * 24 * time.Hour, false},
		{"2d", 0, true},
		{"forever", 0, true},
	}
	for _, c := range cases {
		got, err := ParseLastActive(c.window)
		if c.bad {
			if err == nil {
				t.Errorf("ParseLastActive(%q) wanted an error", c.window)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseLastActive(%q) returned %v", c.window, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseLastActive(%q) = %v, want %v", c.window, got, c.want)
		}
	}
}

func TestListFilterLastActive(t *testing.T) {
	now := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	hoursAgo := func(h int) time.Time { return now.Add(-time.Duration(h) * time.Hour) }

	all := []Session{
		{Name: "live-old", Live: true, LastActiveAt: hoursAgo(240)},
		{Name: "stored-fresh", LastActiveAt: hoursAgo(2)},
		{Name: "stored-week", LastActiveAt: hoursAgo(100)},
		{Name: "stored-stale", LastActiveAt: hoursAgo(240)},
		{Name: "stored-zero"},
		{Name: "attic-fresh", Archived: true, LastActiveAt: hoursAgo(2)},
		{Name: "attic-stale", Archived: true, LastActiveAt: hoursAgo(240)},
	}

	names := func(in listIn) []string {
		kept, err := in.filter(all, now)
		if err != nil {
			t.Fatalf("filter returned %v", err)
		}
		out := make([]string, len(kept))
		for i, item := range kept {
			out[i] = item.Name
		}
		return out
	}

	eq := func(got, want []string) bool {
		if len(got) != len(want) {
			return false
		}
		for i := range got {
			if got[i] != want[i] {
				return false
			}
		}
		return true
	}

	if got := names(listIn{}); !eq(got, []string{"live-old"}) {
		t.Errorf("default list returned %v, want the running session alone", got)
	}

	if got := names(listIn{Stopped: true}); !eq(got, []string{"live-old", "stored-fresh", "stored-zero"}) {
		t.Errorf("stored within a day returned %v", got)
	}

	if got := names(listIn{Stopped: true, LastActive: LastActiveWeek}); !eq(got, []string{"live-old", "stored-fresh", "stored-week", "stored-zero"}) {
		t.Errorf("stored within a week returned %v", got)
	}

	if got := names(listIn{Stopped: true, LastActive: LastActiveUnset}); !eq(got, []string{"live-old", "stored-fresh", "stored-week", "stored-stale", "stored-zero"}) {
		t.Errorf("stored with no limit returned %v", got)
	}

	if got := names(listIn{Archived: true}); !eq(got, []string{"live-old", "attic-fresh"}) {
		t.Errorf("archived within a day returned %v", got)
	}

	if got := names(listIn{Stopped: true, Archived: true, LastActive: LastActiveYear}); !eq(got, []string{"live-old", "stored-fresh", "stored-week", "stored-stale", "stored-zero", "attic-fresh", "attic-stale"}) {
		t.Errorf("every category within a year returned %v", got)
	}

	if _, err := (listIn{LastActive: "2d"}).filter(all, now); err == nil {
		t.Errorf("a bad window wanted an error")
	}
}

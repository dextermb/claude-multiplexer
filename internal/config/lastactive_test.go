package config

import (
	"testing"
	"time"
)

func TestArchivedWindow(t *testing.T) {
	cases := []struct {
		value string
		want  string
	}{
		{"", DefaultLastActive},
		{"1d", "1d"},
		{"1W", "1w"},
		{" 1m ", "1m"},
		{"1y", "1y"},
		{"unset", "unset"},
		{"2d", DefaultLastActive},
		{"forever", DefaultLastActive},
	}
	for _, c := range cases {
		if got := ArchivedWindow(c.value); got != c.want {
			t.Errorf("ArchivedWindow(%q) = %q, want %q", c.value, got, c.want)
		}
	}
}

func TestArchivedCutoff(t *testing.T) {
	now := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		window string
		wantOK bool
		back   time.Duration
	}{
		{"1d", true, 24 * time.Hour},
		{"1w", true, 7 * 24 * time.Hour},
		{"1y", true, 365 * 24 * time.Hour},
		{"unset", false, 0},
		{"", true, 24 * time.Hour},
		{"forever", true, 24 * time.Hour},
	}
	for _, c := range cases {
		cutoff, ok := ArchivedCutoff(c.window, now)
		if ok != c.wantOK {
			t.Errorf("ArchivedCutoff(%q) ok = %v, want %v", c.window, ok, c.wantOK)
			continue
		}
		if !ok {
			if !cutoff.IsZero() {
				t.Errorf("ArchivedCutoff(%q) = %v, want zero", c.window, cutoff)
			}
			continue
		}
		if want := now.Add(-c.back); !cutoff.Equal(want) {
			t.Errorf("ArchivedCutoff(%q) = %v, want %v", c.window, cutoff, want)
		}
	}
}

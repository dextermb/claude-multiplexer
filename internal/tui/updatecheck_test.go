package tui

import (
	"testing"
	"time"
)

func TestUpdateVisible(t *testing.T) {
	cases := []struct {
		name     string
		outdated bool
		check    bool
		dismiss  time.Time
		want     bool
	}{
		{"out of date, check on, not dismissed", true, true, time.Time{}, true},
		{"up to date", false, true, time.Time{}, false},
		{"check off", true, false, time.Time{}, false},
		{"dismissed", true, true, time.Now().Add(time.Hour), false},
		{"dismissal expired", true, true, time.Now().Add(-time.Hour), true},
	}
	for _, c := range cases {
		m := Model{updateOutdated: c.outdated, checkUpdates: c.check, updateDismiss: c.dismiss}
		if got := m.updateVisible(); got != c.want {
			t.Errorf("%s: updateVisible = %v, want %v", c.name, got, c.want)
		}
		if got := m.bannerHeight(); (got == 1) != c.want {
			t.Errorf("%s: bannerHeight = %d, want visible=%v", c.name, got, c.want)
		}
	}
}

func TestDismissUpdateHidesBanner(t *testing.T) {
	m := Model{updateOutdated: true, checkUpdates: true}
	if !m.updateVisible() {
		t.Fatal("the banner must show before a dismissal")
	}
	next, _ := m.dismissUpdate()
	if next.(Model).updateVisible() {
		t.Error("the banner must hide after a dismissal")
	}
}

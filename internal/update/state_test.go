package update

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStateRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "update-check.json")
	want := State{
		LastCheck:      time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		Tag:            "v1.2.3",
		URL:            "https://example/x",
		PublishedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		DismissedUntil: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC),
	}
	if err := SaveState(path, want); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	got, err := LoadState(path)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if !got.LastCheck.Equal(want.LastCheck) || got.Tag != want.Tag || got.URL != want.URL ||
		!got.PublishedAt.Equal(want.PublishedAt) || !got.DismissedUntil.Equal(want.DismissedUntil) {
		t.Errorf("round trip = %+v, want %+v", got, want)
	}
}

func TestLoadStateMissing(t *testing.T) {
	got, err := LoadState(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil {
		t.Fatalf("a missing file must not be an error: %v", err)
	}
	if !got.LastCheck.IsZero() {
		t.Error("a missing file must give a zero State")
	}
}

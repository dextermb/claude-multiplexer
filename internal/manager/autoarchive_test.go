package manager

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/render"
)

func TestDueForArchive(t *testing.T) {
	now := time.Date(2026, 1, 20, 12, 0, 0, 0, time.UTC)
	old := now.Add(-10 * 24 * time.Hour)
	fresh := now.Add(-2 * 24 * time.Hour)

	metas := []Meta{
		{Name: "old", LastActiveAt: old},
		{Name: "fresh", LastActiveAt: fresh},
		{Name: "archived", LastActiveAt: old, Archived: true},
		{Name: "zero"},
	}

	got := dueForArchive(metas, 7, now)
	if len(got) != 1 || got[0] != "old" {
		t.Fatalf("dueForArchive = %v, want [old]", got)
	}
}

func TestDueForArchiveBoundary(t *testing.T) {
	now := time.Date(2026, 1, 20, 12, 0, 0, 0, time.UTC)
	metas := []Meta{
		{Name: "exact", LastActiveAt: now.Add(-7 * 24 * time.Hour)},
		{Name: "under", LastActiveAt: now.Add(-7*24*time.Hour + time.Minute)},
	}
	got := dueForArchive(metas, 7, now)
	if len(got) != 1 || got[0] != "exact" {
		t.Fatalf("dueForArchive = %v, want [exact]", got)
	}
}

func TestSweepArchiveArchivesTheIdleSession(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.json")
	days := 7
	if err := config.Write(configPath, config.Config{AutoArchiveDays: &days}); err != nil {
		t.Fatalf("write config: %v", err)
	}

	m, err := New(Options{Root: root, ClaudePath: fakeClaude,
		ConfigPaths: []string{configPath}, Renderer: render.Renderer{}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	now := time.Now()
	writeSweepMeta(t, root, "old", now.Add(-10*24*time.Hour), false)
	writeSweepMeta(t, root, "fresh", now.Add(-2*24*time.Hour), false)

	m.sweepArchive(now)

	if meta, _ := m.Meta("old"); !meta.Archived {
		t.Errorf("old was not archived")
	}
	if meta, _ := m.Meta("fresh"); meta.Archived {
		t.Errorf("fresh was archived")
	}
}

func TestSweepArchiveOffWhenUnset(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := config.Write(configPath, config.Config{}); err != nil {
		t.Fatalf("write config: %v", err)
	}

	m, err := New(Options{Root: root, ClaudePath: fakeClaude,
		ConfigPaths: []string{configPath}, Renderer: render.Renderer{}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	writeSweepMeta(t, root, "old", time.Now().Add(-100*24*time.Hour), false)
	m.sweepArchive(time.Now())

	if meta, _ := m.Meta("old"); meta.Archived {
		t.Errorf("old was archived with the setting off")
	}
}

func writeSweepMeta(t *testing.T, root, name string, lastActive time.Time, archived bool) {
	t.Helper()
	meta := Meta{
		Name:         name,
		Dir:          root,
		LastActiveAt: lastActive,
		Turns:        1,
		Archived:     archived,
	}
	if err := writeMeta(metaPath(root, name), meta); err != nil {
		t.Fatalf("write meta %s: %v", name, err)
	}
}

package manager

import (
	"path/filepath"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/render"
)

func newWatchManager(t *testing.T, configPath string) *Manager {
	t.Helper()
	m, err := New(Options{Root: t.TempDir(), ClaudePath: fakeClaude,
		ConfigPaths: []string{configPath}, Renderer: render.Renderer{}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return m
}

func TestCheckConfigReloadsOnChange(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := config.Write(configPath, config.Config{}); err != nil {
		t.Fatalf("write config: %v", err)
	}
	m := newWatchManager(t, configPath)
	m.configPrint = configFingerprint(m.opts.ConfigPaths)

	sub := m.Subscribe(1)
	defer sub.Close()

	editor := "nvim"
	if err := config.Write(configPath, config.Config{Editor: editor}); err != nil {
		t.Fatalf("rewrite config: %v", err)
	}

	m.checkConfig()

	select {
	case ev := <-sub.C:
		if !ev.Reload {
			t.Fatalf("event = %+v, want Reload true", ev)
		}
		if ev.Notice != "" {
			t.Errorf("event carries a notice %q, want none", ev.Notice)
		}
	default:
		t.Fatal("checkConfig published no reload after the file changed")
	}
}

func TestCheckConfigQuietWhenUnchanged(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := config.Write(configPath, config.Config{Editor: "nvim"}); err != nil {
		t.Fatalf("write config: %v", err)
	}
	m := newWatchManager(t, configPath)
	m.configPrint = configFingerprint(m.opts.ConfigPaths)

	sub := m.Subscribe(1)
	defer sub.Close()

	m.checkConfig()

	select {
	case ev := <-sub.C:
		t.Fatalf("checkConfig published %+v with no change", ev)
	default:
	}
}

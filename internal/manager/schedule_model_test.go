package manager

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/render"
)

func managerWithScheduleModel(t *testing.T, model string) *Manager {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), config.FileName)
	if err := config.Write(configPath, config.Config{DefaultScheduleModel: model}); err != nil {
		t.Fatalf("write config: %v", err)
	}
	m, err := New(Options{
		Root:        t.TempDir(),
		ClaudePath:  fakeClaude,
		ConfigPaths: []string{configPath},
		Renderer:    render.Renderer{},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		m.Shutdown(ctx)
	})
	return m
}

func TestCreateScheduleTakesTheDefaultModel(t *testing.T) {
	m := managerWithScheduleModel(t, "haiku")
	got := createSchedule(t, m, ScheduleSpec{Name: "poll"})
	if got.Model != "haiku" {
		t.Fatalf("Model = %q, want the setting default", got.Model)
	}
}

func TestCreateScheduleKeepsTheModelItWasGiven(t *testing.T) {
	m := managerWithScheduleModel(t, "haiku")
	got := createSchedule(t, m, ScheduleSpec{Name: "poll", Model: "opus"})
	if got.Model != "opus" {
		t.Fatalf("Model = %q, want the model of the caller", got.Model)
	}
}

func TestCreateScheduleLeavesTheModelEmptyWithoutASetting(t *testing.T) {
	m := managerWithScheduleModel(t, "")
	got := createSchedule(t, m, ScheduleSpec{Name: "poll"})
	if got.Model != "" {
		t.Fatalf("Model = %q, want an empty model", got.Model)
	}
}

func TestCreateScheduleWritesTheDefaultModelToDisk(t *testing.T) {
	m := managerWithScheduleModel(t, "haiku")
	created := createSchedule(t, m, ScheduleSpec{Name: "poll"})

	stored, err := ReadSchedule(schedulePath(m.opts.Root, created.Name))
	if err != nil {
		t.Fatalf("ReadSchedule: %v", err)
	}
	if stored.Model != "haiku" {
		t.Fatalf("stored Model = %q, want the setting default", stored.Model)
	}
}

func TestADefaultModelChangeLeavesAnExistingScheduleAlone(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), config.FileName)
	if err := config.Write(configPath, config.Config{DefaultScheduleModel: "haiku"}); err != nil {
		t.Fatalf("write config: %v", err)
	}
	m, err := New(Options{
		Root:        t.TempDir(),
		ClaudePath:  fakeClaude,
		ConfigPaths: []string{configPath},
		Renderer:    render.Renderer{},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		m.Shutdown(ctx)
	})

	first := createSchedule(t, m, ScheduleSpec{Name: "poll"})
	if err := config.Write(configPath, config.Config{DefaultScheduleModel: "opus"}); err != nil {
		t.Fatalf("rewrite config: %v", err)
	}
	second := createSchedule(t, m, ScheduleSpec{Name: "sweep"})

	for _, s := range m.ListSchedules() {
		if s.Name == first.Name && s.Model != "haiku" {
			t.Fatalf("%s Model = %q, want the model it was created with", s.Name, s.Model)
		}
		if s.Name == second.Name && s.Model != "opus" {
			t.Fatalf("%s Model = %q, want the new setting", s.Name, s.Model)
		}
	}
}

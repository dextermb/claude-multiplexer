package config

import "testing"

func ptr(n int) *int { return &n }

func TestResolveLayoutFallsBackToDefaults(t *testing.T) {
	got := ResolveLayout(nil, "", "")
	if got != DefaultLayout() {
		t.Fatalf("ResolveLayout(nil) = %+v, want %+v", got, DefaultLayout())
	}
}

func TestResolveLayoutOverlaysGlobalThenSession(t *testing.T) {
	layouts := map[string]Layout{
		"wide":    {SidebarWidth: ptr(40), DiffWidth: ptr(60)},
		"session": {SidebarWidth: ptr(30)},
	}
	got := ResolveLayout(layouts, "wide", "session")

	if got.SidebarWidth != 30 {
		t.Errorf("SidebarWidth = %d, want 30 (session wins)", got.SidebarWidth)
	}
	if got.DiffWidth != 60 {
		t.Errorf("DiffWidth = %d, want 60 (from global)", got.DiffWidth)
	}
	if got.TaskWidth != DefaultTaskWidth {
		t.Errorf("TaskWidth = %d, want default %d", got.TaskWidth, DefaultTaskWidth)
	}
}

func TestResolveLayoutIgnoresMissingName(t *testing.T) {
	layouts := map[string]Layout{"wide": {SidebarWidth: ptr(40)}}
	got := ResolveLayout(layouts, "gone", "also-gone")
	if got != DefaultLayout() {
		t.Fatalf("ResolveLayout(missing names) = %+v, want defaults", got)
	}
}

func TestResolveLayoutKeepsPromptBoundsSane(t *testing.T) {
	layouts := map[string]Layout{"bad": {PromptMin: ptr(9), PromptMax: ptr(3)}}
	got := ResolveLayout(layouts, "bad", "")
	if got.PromptMin != 9 {
		t.Errorf("PromptMin = %d, want 9", got.PromptMin)
	}
	if got.PromptMax < got.PromptMin {
		t.Errorf("PromptMax = %d, want >= PromptMin %d", got.PromptMax, got.PromptMin)
	}
}

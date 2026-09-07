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
		"wide":    {SidebarSize: ptr(40), DiffSize: ptr(60)},
		"session": {SidebarSize: ptr(30)},
	}
	got := ResolveLayout(layouts, "wide", "session")

	if got.SidebarSize != 30 {
		t.Errorf("SidebarSize = %d, want 30 (session wins)", got.SidebarSize)
	}
	if got.DiffSize != 60 {
		t.Errorf("DiffSize = %d, want 60 (from global)", got.DiffSize)
	}
	if got.TaskSize != DefaultTaskSize {
		t.Errorf("TaskSize = %d, want default %d", got.TaskSize, DefaultTaskSize)
	}
}

func TestResolveLayoutIgnoresMissingName(t *testing.T) {
	layouts := map[string]Layout{"wide": {SidebarSize: ptr(40)}}
	got := ResolveLayout(layouts, "gone", "also-gone")
	if got != DefaultLayout() {
		t.Fatalf("ResolveLayout(missing names) = %+v, want defaults", got)
	}
}

func strp(s string) *string { return &s }

func TestResolveLayoutDefaultDiffPositionAndSize(t *testing.T) {
	got := ResolveLayout(nil, "", "")
	if got.DiffPosition != DiffRight {
		t.Errorf("DiffPosition = %q, want %q", got.DiffPosition, DiffRight)
	}
	if got.DiffSize != DefaultDiffSize {
		t.Errorf("DiffSize = %d, want %d columns", got.DiffSize, DefaultDiffSize)
	}
}

func TestResolveLayoutHorizontalSizeTakesRowDefault(t *testing.T) {
	layouts := map[string]Layout{"stack": {DiffPosition: strp(DiffBottom)}}
	got := ResolveLayout(layouts, "stack", "")
	if got.DiffPosition != DiffBottom {
		t.Errorf("DiffPosition = %q, want %q", got.DiffPosition, DiffBottom)
	}
	if got.DiffSize != DefaultDiffRows {
		t.Errorf("DiffSize = %d, want %d rows (horizontal default)", got.DiffSize, DefaultDiffRows)
	}
}

func TestResolveLayoutKeepsExplicitDiffSize(t *testing.T) {
	layouts := map[string]Layout{"stack": {DiffPosition: strp(DiffTop), DiffSize: ptr(20)}}
	got := ResolveLayout(layouts, "stack", "")
	if got.DiffSize != 20 {
		t.Errorf("DiffSize = %d, want 20 (explicit)", got.DiffSize)
	}
}

func TestResolveLayoutBadPositionFallsToRight(t *testing.T) {
	layouts := map[string]Layout{"bad": {DiffPosition: strp("sideways")}}
	got := ResolveLayout(layouts, "bad", "")
	if got.DiffPosition != DiffRight {
		t.Errorf("DiffPosition = %q, want %q", got.DiffPosition, DiffRight)
	}
}

func TestResolveLayoutHorizontalSizeFloor(t *testing.T) {
	layouts := map[string]Layout{"tiny": {DiffPosition: strp(DiffTop), DiffSize: ptr(1)}}
	got := ResolveLayout(layouts, "tiny", "")
	if got.DiffSize < 4 {
		t.Errorf("DiffSize = %d, want at least the row floor 4", got.DiffSize)
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

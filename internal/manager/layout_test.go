package manager

import (
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/config"
)

func TestSaveLayoutCapturesDefaultsAndKeepsOverrides(t *testing.T) {
	m, path := withConfig(t)
	forty := 40
	if _, err := m.SaveLayout("wide", config.Layout{SidebarWidth: &forty}, ""); err != nil {
		t.Fatalf("SaveLayout: %v", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := cfg.Layouts["wide"]
	if !ok {
		t.Fatal("the layout was not saved")
	}
	if got.SidebarWidth == nil || *got.SidebarWidth != 40 {
		t.Fatalf("sidebar = %v, want the override 40", got.SidebarWidth)
	}
	if got.TaskWidth == nil || *got.TaskWidth != config.DefaultTaskWidth {
		t.Fatalf("task = %v, want the captured default %d", got.TaskWidth, config.DefaultTaskWidth)
	}
}

func TestSetActiveLayoutRequiresAnExistingLayout(t *testing.T) {
	m, _ := withConfig(t)
	if _, err := m.SetActiveLayout("nope"); err == nil {
		t.Fatal("SetActiveLayout must fail for an unknown layout")
	}
	if _, err := m.SaveLayout("wide", config.Layout{}, ""); err != nil {
		t.Fatalf("SaveLayout: %v", err)
	}
	if _, err := m.SetActiveLayout("wide"); err != nil {
		t.Fatalf("SetActiveLayout: %v", err)
	}
	list, err := m.LayoutList("")
	if err != nil {
		t.Fatal(err)
	}
	if list.ActiveGlobal != "wide" {
		t.Fatalf("active global = %q, want wide", list.ActiveGlobal)
	}
	if len(list.Layouts) != 1 || list.Layouts[0].Name != "wide" {
		t.Fatalf("layouts = %+v, want one named wide", list.Layouts)
	}
}

func TestUnsetActiveLayoutReportsWhetherOneWasSet(t *testing.T) {
	m, _ := withConfig(t)
	if _, changed, err := m.UnsetActiveLayout(); err != nil || changed {
		t.Fatalf("nothing set: changed=%v err=%v, want false nil", changed, err)
	}
	if _, err := m.SaveLayout("wide", config.Layout{}, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := m.SetActiveLayout("wide"); err != nil {
		t.Fatal(err)
	}
	_, changed, err := m.UnsetActiveLayout()
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("the active layout was set, change must be true")
	}
}

func TestDeleteLayoutReportsWhetherItWasThere(t *testing.T) {
	m, _ := withConfig(t)
	if _, changed, err := m.DeleteLayout("ghost"); err != nil || changed {
		t.Fatalf("nothing to delete: changed=%v err=%v, want false nil", changed, err)
	}
	if _, err := m.SaveLayout("wide", config.Layout{}, ""); err != nil {
		t.Fatal(err)
	}
	_, changed, err := m.DeleteLayout("wide")
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("the layout was there, change must be true")
	}
}

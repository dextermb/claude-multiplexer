package config

import (
	"encoding/json"
	"testing"
)

func TestSetPathSetsAScalar(t *testing.T) {
	got, err := SetPath(Config{}, "editor", json.RawMessage(`"nvim"`))
	if err != nil {
		t.Fatalf("SetPath: %v", err)
	}
	if got.Editor != "nvim" {
		t.Fatalf("editor = %q, want nvim", got.Editor)
	}
}

func TestSetPathReachesAMapKey(t *testing.T) {
	got, err := SetPath(Config{}, "blockCaps.tool", json.RawMessage(`3`))
	if err != nil {
		t.Fatalf("SetPath: %v", err)
	}
	rows, ok := got.BlockCaps["tool"]
	if !ok || rows == nil || *rows != 3 {
		t.Fatalf("blockCaps.tool = %v, want 3", got.BlockCaps["tool"])
	}
}

func TestSetPathMakesNestedObjects(t *testing.T) {
	got, err := SetPath(Config{}, "layouts.wide.sidebarSize", json.RawMessage(`30`))
	if err != nil {
		t.Fatalf("SetPath: %v", err)
	}
	layout, ok := got.Layouts["wide"]
	if !ok || layout.SidebarSize == nil || *layout.SidebarSize != 30 {
		t.Fatalf("layouts.wide.sidebarSize = %v, want 30", layout.SidebarSize)
	}
}

func TestSetPathTakesAnExplicitNull(t *testing.T) {
	got, err := SetPath(Config{}, "blockCaps.tool", json.RawMessage(`null`))
	if err != nil {
		t.Fatalf("SetPath: %v", err)
	}
	rows, ok := got.BlockCaps["tool"]
	if !ok || rows != nil {
		t.Fatalf("blockCaps.tool = %v, want null", rows)
	}
}

func TestSetPathRejectsAnUnknownKey(t *testing.T) {
	if _, err := SetPath(Config{}, "edtior", json.RawMessage(`"nvim"`)); err == nil {
		t.Fatal("SetPath took an unknown key, want an error")
	}
}

func TestSetPathRejectsAnUnknownLayoutField(t *testing.T) {
	if _, err := SetPath(Config{}, "layouts.wide.sidebarSizes", json.RawMessage(`30`)); err == nil {
		t.Fatal("SetPath took an unknown layout field, want an error")
	}
}

func TestSetPathRejectsAWrongType(t *testing.T) {
	if _, err := SetPath(Config{}, "blockCap", json.RawMessage(`"lots"`)); err == nil {
		t.Fatal("SetPath took a string for blockCap, want an error")
	}
}

func TestSetPathRejectsANonObjectStep(t *testing.T) {
	cfg := Config{Editor: "nvim"}
	if _, err := SetPath(cfg, "editor.terminal", json.RawMessage(`true`)); err == nil {
		t.Fatal("SetPath went through a string, want an error")
	}
}

func TestSetPathRejectsAnEmptyPath(t *testing.T) {
	if _, err := SetPath(Config{}, "", json.RawMessage(`1`)); err == nil {
		t.Fatal("SetPath took an empty path, want an error")
	}
}

func TestUnsetPathRemovesAKey(t *testing.T) {
	rows := 40
	cfg := Config{BlockCap: &rows}
	got, changed, err := UnsetPath(cfg, "blockCap")
	if err != nil {
		t.Fatalf("UnsetPath: %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}
	if got.BlockCap != nil {
		t.Fatalf("blockCap = %v, want nil", got.BlockCap)
	}
}

func TestUnsetPathReachesAMapKey(t *testing.T) {
	three := 3
	cfg := Config{BlockCaps: map[string]*int{"tool": &three}}
	got, changed, err := UnsetPath(cfg, "blockCaps.tool")
	if err != nil {
		t.Fatalf("UnsetPath: %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}
	if _, ok := got.BlockCaps["tool"]; ok {
		t.Fatal("blockCaps.tool is still there, want it gone")
	}
}

func TestUnsetPathReportsAMissingKey(t *testing.T) {
	_, changed, err := UnsetPath(Config{}, "blockCap")
	if err != nil {
		t.Fatalf("UnsetPath: %v", err)
	}
	if changed {
		t.Fatal("changed = true, want false for a key that is not there")
	}
}

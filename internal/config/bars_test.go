package config

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestDefaultBarSpecMatchesTheBuiltinOrder(t *testing.T) {
	session, err := DefaultBarSpec(BarSession)
	if err != nil {
		t.Fatalf("DefaultBarSpec(session): %v", err)
	}
	wantLeft := []string{"name", "control", "model", "mode", "effort", "workItem"}
	if got := ids(session.Left); !equalStrings(got, wantLeft) {
		t.Fatalf("session left = %v, want %v", got, wantLeft)
	}
	wantRight := []string{"state", "diff", "pr", "context", "raw", "scroll", "jobs", "queued", "tokens", "cache", "cost"}
	if got := ids(session.Right); !equalStrings(got, wantRight) {
		t.Fatalf("session right = %v, want %v", got, wantRight)
	}

	status, err := DefaultBarSpec(BarStatus)
	if err != nil {
		t.Fatalf("DefaultBarSpec(status): %v", err)
	}
	if got := ids(status.Left); !equalStrings(got, []string{"sessions", "busy", "cost", "status"}) {
		t.Fatalf("status left = %v", got)
	}
	if got := ids(status.Right); !equalStrings(got, []string{"hints"}) {
		t.Fatalf("status right = %v", got)
	}
}

func TestDefaultBarSpecMatchesTheBuiltinIDs(t *testing.T) {
	for _, bar := range []string{BarSession, BarStatus} {
		spec, err := DefaultBarSpec(bar)
		if err != nil {
			t.Fatalf("DefaultBarSpec(%s): %v", bar, err)
		}
		for side, elements := range map[string][]BarElement{"left": spec.Left, "right": spec.Right} {
			for _, e := range elements {
				if !validBuiltinID(bar, side, e.ID) {
					t.Fatalf("default %s.%s has %q, which is not a built-in id", bar, side, e.ID)
				}
			}
		}
	}
}

func TestBarElementReadsAStringOrAnObject(t *testing.T) {
	var spec BarSpec
	raw := `{"left":["name",{"script":"x.sh","label":"branch","refresh":"2s"}]}`
	if err := json.Unmarshal([]byte(raw), &spec); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(spec.Left) != 2 {
		t.Fatalf("left has %d elements, want 2", len(spec.Left))
	}
	if spec.Left[0].ID != "name" || spec.Left[0].Custom() {
		t.Fatalf("first element = %+v, want the built-in name", spec.Left[0])
	}
	if !spec.Left[1].Custom() || spec.Left[1].Label != "branch" {
		t.Fatalf("second element = %+v, want a custom branch element", spec.Left[1])
	}
}

func TestResolveBarSpecFallsBackToTheDefault(t *testing.T) {
	spec, err := ResolveBarSpec(nil, BarSession)
	if err != nil {
		t.Fatalf("ResolveBarSpec(nil): %v", err)
	}
	if ids(spec.Left)[0] != "name" {
		t.Fatalf("left = %v, want the default", ids(spec.Left))
	}
}

func TestResolveBarSpecReplacesOneSide(t *testing.T) {
	bars := &Bars{Session: &BarSpec{Left: []BarElement{{ID: "cost"}}}}
	spec, err := ResolveBarSpec(bars, BarSession)
	if err != nil {
		t.Fatalf("ResolveBarSpec: %v", err)
	}
	if got := ids(spec.Left); !equalStrings(got, []string{"cost"}) {
		t.Fatalf("left = %v, want the configured [cost]", got)
	}
	if ids(spec.Right)[0] != "state" {
		t.Fatalf("right = %v, want the default right", ids(spec.Right))
	}
}

func TestResolveBarSpecKeepsAnEmptySide(t *testing.T) {
	bars := &Bars{Session: &BarSpec{Right: []BarElement{}}}
	spec, err := ResolveBarSpec(bars, BarSession)
	if err != nil {
		t.Fatalf("ResolveBarSpec: %v", err)
	}
	if len(spec.Right) != 0 {
		t.Fatalf("right = %v, want an empty side", ids(spec.Right))
	}
	if ids(spec.Left)[0] != "name" {
		t.Fatalf("left = %v, want the default left", ids(spec.Left))
	}
}

func TestValidateBarsRejectsAnUnknownID(t *testing.T) {
	bars := &Bars{Session: &BarSpec{Left: []BarElement{{ID: "hints"}}}}
	if err := ValidateBars(bars); err == nil {
		t.Fatal("ValidateBars accepted hints on the session left, want an error")
	}
}

func TestValidateBarsAcceptsASessionIDOnTheStatusBar(t *testing.T) {
	bars := &Bars{Status: &BarSpec{Right: []BarElement{{ID: "cache"}}, Left: []BarElement{{ID: "model"}}}}
	if err := ValidateBars(bars); err != nil {
		t.Fatalf("ValidateBars rejected a session id on the status bar: %v", err)
	}
}

func TestValidateBarsRejectsAStatusIDOnTheSessionBar(t *testing.T) {
	bars := &Bars{Session: &BarSpec{Right: []BarElement{{ID: "sessions"}}}}
	if err := ValidateBars(bars); err == nil {
		t.Fatal("ValidateBars accepted a status id on the session bar, want an error")
	}
}

func TestValidateBarsRejectsABadExtension(t *testing.T) {
	bars := &Bars{Status: &BarSpec{Right: []BarElement{{Script: "x.rb"}}}}
	if err := ValidateBars(bars); err == nil {
		t.Fatal("ValidateBars accepted a .rb script, want an error")
	}
}

func TestValidateBarsRejectsABadRefresh(t *testing.T) {
	bars := &Bars{Status: &BarSpec{Right: []BarElement{{Script: "x.sh", Refresh: "soon"}}}}
	if err := ValidateBars(bars); err == nil {
		t.Fatal("ValidateBars accepted a non-duration refresh, want an error")
	}
}

func TestBarScriptRunnerReadsTheExtension(t *testing.T) {
	cases := map[string][]string{
		"a/branch.sh": {"bash"},
		"weather.py":  {"python3"},
		"stat.go":     {"go", "run"},
	}
	for path, want := range cases {
		cmd, ok := BarScriptRunner(path)
		if !ok || !equalStrings(cmd, want) {
			t.Fatalf("BarScriptRunner(%q) = %v %v, want %v", path, cmd, ok, want)
		}
	}
	if _, ok := BarScriptRunner("x.rb"); ok {
		t.Fatal("BarScriptRunner(.rb) reported a runner, want none")
	}
}

func TestBarRefreshTakesTheDefaultAndTheFloor(t *testing.T) {
	if d := (BarElement{}).BarRefresh(); d != DefaultBarRefresh {
		t.Fatalf("empty refresh = %v, want %v", d, DefaultBarRefresh)
	}
	if d := (BarElement{Refresh: "10ms"}).BarRefresh(); d != MinBarRefresh {
		t.Fatalf("10ms refresh = %v, want the floor %v", d, MinBarRefresh)
	}
	if d := (BarElement{Refresh: "10s"}).BarRefresh(); d != 10*time.Second {
		t.Fatalf("10s refresh = %v, want 10s", d)
	}
}

func TestLoadRejectsABadBar(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/config.json"
	if err := os.WriteFile(path, []byte(`{"bars":{"session":{"left":["nope"]}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("Load accepted an unknown bar id, want an error")
	}
}

func ids(elements []BarElement) []string {
	out := make([]string, len(elements))
	for i, e := range elements {
		out[i] = e.ID
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

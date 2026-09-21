package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/manager"
)

func TestFirstLineTrimsAndCuts(t *testing.T) {
	if got := firstLine([]byte("main\nextra\n")); got != "main" {
		t.Fatalf("firstLine = %q, want main", got)
	}
	if got := firstLine([]byte("  spaced  ")); got != "spaced" {
		t.Fatalf("firstLine = %q, want the trimmed text", got)
	}
}

func TestExpandBarPathResolvesRelativeAndAbsolute(t *testing.T) {
	if got := expandBarPath("branch.sh", "/cfg"); got != "/cfg/branch.sh" {
		t.Fatalf("relative path = %q, want /cfg/branch.sh", got)
	}
	if got := expandBarPath("/abs/x.sh", "/cfg"); got != "/abs/x.sh" {
		t.Fatalf("absolute path = %q, want it unchanged", got)
	}
}

func TestRunBarScriptReadsTheStdinPayload(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "echo.sh")
	body := "#!/bin/sh\ngrep -o '\"bar\":\"[a-z]*\"'\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	job := barJob{bar: config.BarSession, label: "x", script: script, payload: []byte(`{"bar":"session"}`)}
	msg, ok := runBarScript(job, "")().(barOutputMsg)
	if !ok {
		t.Fatalf("run gave %T, want barOutputMsg", msg)
	}
	if msg.err != nil {
		t.Fatalf("run error: %v", msg.err)
	}
	if msg.text != `"bar":"session"` {
		t.Fatalf("text = %q, want the payload echoed back", msg.text)
	}
}

func TestSessionPayloadCarriesTheRow(t *testing.T) {
	item := busyRow()
	item.name = "api"
	item.model = "claude-opus-4-8"
	item.prs = []manager.PRBadge{{Provider: "github", Number: 7, State: "open", Unresolved: 3}}
	item.workItem = manager.WorkItemBadge{Provider: "linear", Key: "ENG-1", Status: "In Review"}

	var m Model
	var payload barPayload
	if err := json.Unmarshal(m.sessionPayload(item), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.Bar != "session" || payload.Session == nil {
		t.Fatalf("payload = %+v, want a session payload", payload)
	}
	if payload.Session.Name != "api" || payload.Session.Model != "claude-opus-4-8" {
		t.Fatalf("session = %+v, want the name and the model", payload.Session)
	}
	if payload.Session.Tokens.CacheRead != item.cacheRead {
		t.Fatalf("tokens = %+v, want the cache read", payload.Session.Tokens)
	}
	if len(payload.Session.PRs) != 1 || payload.Session.PRs[0].Number != 7 {
		t.Fatalf("prs = %+v, want one pull request", payload.Session.PRs)
	}
	if payload.Session.WorkItem == nil || payload.Session.WorkItem.Key != "ENG-1" {
		t.Fatalf("workItem = %+v, want the linked key", payload.Session.WorkItem)
	}
}

func TestStatusPayloadCountsTheSessions(t *testing.T) {
	m := Model{cost: 0.5, costWindow: "1d"}
	var payload barPayload
	if err := json.Unmarshal(m.statusPayload(), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.Bar != "status" || payload.Totals == nil {
		t.Fatalf("payload = %+v, want a status payload", payload)
	}
	if payload.Totals.Cost != 0.5 || payload.Totals.CostWindow != "1d" {
		t.Fatalf("totals = %+v, want the cost and the window", payload.Totals)
	}
}

func TestHasBarScriptsReadsTheSpecs(t *testing.T) {
	plain := Model{barSpecs: map[string]config.BarSpec{
		config.BarSession: {Right: []config.BarElement{{ID: "cost"}}},
	}}
	if plain.hasBarScripts() {
		t.Fatal("hasBarScripts reported a script with only built-ins")
	}
	scripted := Model{barSpecs: map[string]config.BarSpec{
		config.BarStatus: {Right: []config.BarElement{{Script: "x.sh", Label: "x"}}},
	}}
	if !scripted.hasBarScripts() {
		t.Fatal("hasBarScripts missed a custom element")
	}
}

func TestBarDueHonoursTheRefresh(t *testing.T) {
	now := time.Unix(1000, 0)
	m := Model{barRuns: map[string]time.Time{
		barOutputKey(config.BarStatus, "", "x"): now,
	}}
	if m.barDue(config.BarStatus, "", "x", 3*time.Second, now.Add(time.Second)) {
		t.Fatal("barDue said due one second after a three-second refresh")
	}
	if !m.barDue(config.BarStatus, "", "x", 3*time.Second, now.Add(4*time.Second)) {
		t.Fatal("barDue said not due four seconds after a three-second refresh")
	}
	if !m.barDue(config.BarStatus, "", "new", time.Second, now) {
		t.Fatal("barDue said an unrun element is not due")
	}
}

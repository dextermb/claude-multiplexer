package manager

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/mcp"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

func withMCP(t *testing.T) *Manager {
	t.Helper()
	m := newTestManager(t)
	if err := m.StartMCP(); err != nil {
		t.Fatalf("StartMCP: %v", err)
	}
	return m
}

func TestEquipToolsWritesTheConfigAndNamesTheTools(t *testing.T) {
	m := withMCP(t)

	var cfg session.Config
	token, err := m.equipTools(&cfg, "docs", false)
	if err != nil {
		t.Fatalf("equipTools: %v", err)
	}
	if token == "" {
		t.Fatal("no token")
	}

	path := mcpConfigPath(m.Root(), "docs")
	if len(cfg.ExtraArgs) != 2 || cfg.ExtraArgs[0] != "--mcp-config" || cfg.ExtraArgs[1] != path {
		t.Fatalf("ExtraArgs = %v", cfg.ExtraArgs)
	}
	if got, want := len(cfg.AllowedTools), len(mcp.OpenTools); got != want {
		t.Fatalf("AllowedTools = %v, want %d", cfg.AllowedTools, want)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var doc struct {
		MCPServers map[string]struct {
			URL     string            `json:"url"`
			Headers map[string]string `json:"headers"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse config: %v", err)
	}
	entry, ok := doc.MCPServers["mux"]
	if !ok {
		t.Fatalf("no mux server in %s", data)
	}
	if entry.URL != m.MCPURL() {
		t.Fatalf("url = %q, want %q", entry.URL, m.MCPURL())
	}
	if entry.Headers["Authorization"] != "Bearer "+token {
		t.Fatalf("header = %q", entry.Headers["Authorization"])
	}
}

func TestEquipToolsAddsTheControlToolsOnlyWithTheGrant(t *testing.T) {
	m := withMCP(t)

	var open, control session.Config
	if _, err := m.equipTools(&open, "docs", false); err != nil {
		t.Fatalf("equipTools: %v", err)
	}
	if _, err := m.equipTools(&control, "api", true); err != nil {
		t.Fatalf("equipTools: %v", err)
	}
	if len(control.AllowedTools)-len(open.AllowedTools) != len(mcp.ControlTools) {
		t.Fatalf("open = %v, control = %v", open.AllowedTools, control.AllowedTools)
	}
	for _, name := range mcp.ControlTools {
		if !contains(control.AllowedTools, mcp.Qualify(name)) {
			t.Errorf("%s is missing from the granted session", name)
		}
		if contains(open.AllowedTools, mcp.Qualify(name)) {
			t.Errorf("%s reached a session without the grant", name)
		}
	}
}

func TestSpawnWithoutTheServerAddsNoTools(t *testing.T) {
	m := newTestManager(t)
	var cfg session.Config
	token, err := m.equipTools(&cfg, "docs", true)
	if err != nil {
		t.Fatalf("equipTools: %v", err)
	}
	if token != "" || cfg.ExtraArgs != nil || cfg.AllowedTools != nil {
		t.Fatalf("token = %q, args = %v, tools = %v", token, cfg.ExtraArgs, cfg.AllowedTools)
	}
}

func TestTheGrantSurvivesAResume(t *testing.T) {
	m := withMCP(t)
	ctx := context.Background()

	name, err := m.Spawn(ctx, Spec{Dir: t.TempDir(), Name: "docs", Control: true})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	runOneTurn(t, m, name, "hello")
	meta := waitForMeta(t, m, name)
	if !meta.Control {
		t.Fatal("the grant was not stored")
	}
	retire(t, m, name)

	again, err := m.Resume(ctx, meta)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if !m.Grants()[again] {
		t.Fatal("the resumed session lost the grant")
	}
}

func TestMessagesReadTheTranscript(t *testing.T) {
	m := withMCP(t)
	name, err := m.Spawn(context.Background(), Spec{Dir: t.TempDir(), Name: "docs"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	runOneTurn(t, m, name, "one")
	if err := m.Send(name, "two"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	waitFor(t, 10*time.Second, func() bool {
		snap, err := m.Snapshot(name)
		return err == nil && snap.Turns >= 2
	})

	messages, err := m.Messages(name, 20)
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}
	var roles, text []string
	for _, item := range messages {
		roles = append(roles, item.Role)
		text = append(text, item.Text)
	}
	joined := strings.Join(text, "|")
	if !strings.Contains(joined, "one") || !strings.Contains(joined, "echo: two") {
		t.Fatalf("messages = %v", messages)
	}
	if !contains(roles, "user") || !contains(roles, "assistant") || !contains(roles, "result") {
		t.Fatalf("roles = %v", roles)
	}

	last, err := m.Messages(name, 1)
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}
	if len(last) != 1 || last[0] != messages[len(messages)-1] {
		t.Fatalf("limit 1 gave %v", last)
	}

	if _, err := m.Messages("gone", 20); err == nil {
		t.Fatal("an unknown session returned messages")
	}
}

func TestJobsReportsARunningJob(t *testing.T) {
	t.Setenv("FAKECLAUDE_MODE", "jobs")
	m := withMCP(t)
	name, err := m.Spawn(context.Background(), Spec{Dir: t.TempDir(), Name: "api"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	waitFor(t, 10*time.Second, func() bool {
		jobs, _ := m.Jobs(name)
		return len(jobs) == 1
	})
	jobs, err := m.Jobs(name)
	if err != nil {
		t.Fatalf("Jobs: %v", err)
	}
	job := jobs[0]
	if job.ID != "job-1" || job.Description != "sleep and echo" || !job.Running || job.Status != "running" {
		t.Fatalf("unexpected job: %+v", job)
	}
}

func TestStopJobFromMarksThePaneAndQueuesTheKill(t *testing.T) {
	t.Setenv("FAKECLAUDE_MODE", "jobs")
	m := withMCP(t)
	name, err := m.Spawn(context.Background(), Spec{Dir: t.TempDir(), Name: "api"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	waitFor(t, 10*time.Second, func() bool {
		jobs, _ := m.Jobs(name)
		return len(jobs) == 1
	})

	if _, err := m.StopJobFrom(name, "docs", "job-1"); err != nil {
		t.Fatalf("StopJobFrom: %v", err)
	}
	waitFor(t, 10*time.Second, func() bool {
		for _, line := range m.Lines(name) {
			if strings.Contains(line.Text, "← stop job job-1 from docs") {
				return true
			}
		}
		return false
	})
	waitFor(t, 10*time.Second, func() bool {
		for _, line := range m.Lines(name) {
			if strings.Contains(line.Text, "KillShell") {
				return true
			}
		}
		return false
	})
}

func TestStopJobFromRefusesUnknownSessionAndJob(t *testing.T) {
	t.Setenv("FAKECLAUDE_MODE", "jobs")
	m := withMCP(t)
	name, err := m.Spawn(context.Background(), Spec{Dir: t.TempDir(), Name: "api"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	waitFor(t, 10*time.Second, func() bool {
		jobs, _ := m.Jobs(name)
		return len(jobs) == 1
	})

	if _, err := m.StopJobFrom("gone", "docs", "job-1"); !errors.Is(err, ErrUnknownSession) {
		t.Fatalf("StopJobFrom on an unknown session = %v", err)
	}
	if _, err := m.StopJobFrom(name, "docs", "nope"); !errors.Is(err, ErrUnknownJob) {
		t.Fatalf("StopJobFrom on an unknown job = %v", err)
	}
}

func TestJobsForUnknownSessionErrors(t *testing.T) {
	m := withMCP(t)
	if _, err := m.Jobs("gone"); !errors.Is(err, ErrUnknownSession) {
		t.Fatalf("Jobs on an unknown session = %v", err)
	}
}

func TestKillPromptNamesTheShell(t *testing.T) {
	with := killPrompt(session.Job{ID: "bao4ntmse", Description: "sleep 20"})
	if !strings.Contains(with, `"bao4ntmse"`) || !strings.Contains(with, "KillShell") || !strings.Contains(with, "sleep 20") {
		t.Fatalf("prompt = %q", with)
	}
	without := killPrompt(session.Job{ID: "x"})
	if !strings.Contains(without, `"x"`) || strings.Contains(without, "The job is") {
		t.Fatalf("prompt = %q", without)
	}
}

func TestFindJob(t *testing.T) {
	jobs := []session.Job{{ID: "a"}, {ID: "b"}}
	if job, ok := findJob(jobs, "b"); !ok || job.ID != "b" {
		t.Fatalf("findJob(b) = %+v, %v", job, ok)
	}
	if _, ok := findJob(jobs, "z"); ok {
		t.Fatal("findJob found a job that is not there")
	}
}

func TestSendFromMarksThePaneWithTheSender(t *testing.T) {
	m := withMCP(t)
	name, err := m.Spawn(context.Background(), Spec{Dir: t.TempDir(), Name: "api"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	sub := m.Subscribe(256)
	defer sub.Close()

	if _, err := m.SendFrom(name, "docs", "take the billing work"); err != nil {
		t.Fatalf("SendFrom: %v", err)
	}
	waitFor(t, 10*time.Second, func() bool {
		for _, line := range m.Lines(name) {
			if strings.Contains(line.Text, "← prompt from docs") {
				return true
			}
		}
		return false
	})
	if _, err := m.SendFrom("gone", "docs", "hello"); err == nil {
		t.Fatal("a prompt reached an unknown session")
	}
}

func TestAToolChangeReachesTheBusAsANotice(t *testing.T) {
	m := withMCP(t)
	ctx := context.Background()
	name, err := m.Spawn(ctx, Spec{Dir: t.TempDir(), Name: "landing"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	runOneTurn(t, m, name, "hello")
	retire(t, m, name)

	sub := m.Subscribe(256)
	defer sub.Close()

	tools := &bridge{m: m}
	if err := tools.Archive(name, true, "docs"); err != nil {
		t.Fatalf("Archive: %v", err)
	}

	ev := awaitNotice(t, sub)
	if !strings.Contains(ev.Notice, "docs archived landing") {
		t.Fatalf("notice = %q", ev.Notice)
	}
	if !ev.Reload {
		t.Fatal("the archive did not ask for a reload")
	}
}

func TestARenameThroughTheToolCarriesANotice(t *testing.T) {
	m := withMCP(t)
	name, err := m.Spawn(context.Background(), Spec{Dir: t.TempDir(), Name: "docs"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	sub := m.Subscribe(256)
	defer sub.Close()

	tools := &bridge{m: m}
	if err := tools.SetTitle(name, "Billing rewrite"); err != nil {
		t.Fatalf("SetTitle: %v", err)
	}
	ev := awaitNotice(t, sub)
	if !strings.Contains(ev.Notice, "docs renamed itself to Billing rewrite") {
		t.Fatalf("notice = %q", ev.Notice)
	}
}

func awaitNotice(t *testing.T, sub *Subscription) Event {
	t.Helper()
	deadline := time.After(10 * time.Second)
	for {
		select {
		case ev := <-sub.C:
			if ev.Notice != "" {
				return ev
			}
		case <-deadline:
			t.Fatal("no notice reached the bus")
		}
	}
}

func TestASessionRenamesItselfThroughTheTool(t *testing.T) {
	t.Setenv("FAKECLAUDE_MODE", "mcp")
	m := withMCP(t)

	name, err := m.Spawn(context.Background(), Spec{Dir: t.TempDir(), Name: "docs"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	runOneTurn(t, m, name, `rename_session {"title":"Billing rewrite"}`)

	waitFor(t, 10*time.Second, func() bool {
		snap, err := m.Snapshot(name)
		return err == nil && snap.Title == "Billing rewrite"
	})
}

func TestOneSessionPromptsAnotherThroughTheTool(t *testing.T) {
	t.Setenv("FAKECLAUDE_MODE", "mcp")
	m := withMCP(t)
	ctx := context.Background()

	dir := t.TempDir()
	if _, err := m.Spawn(ctx, Spec{Dir: dir, Name: "api"}); err != nil {
		t.Fatalf("Spawn api: %v", err)
	}
	if _, err := m.Spawn(ctx, Spec{Dir: dir, Name: "docs", Control: true}); err != nil {
		t.Fatalf("Spawn docs: %v", err)
	}

	runOneTurn(t, m, "docs", `send_message {"session":"api","text":"list_sessions {}"}`)

	waitFor(t, 10*time.Second, func() bool {
		var marked bool
		for _, line := range m.Lines("api") {
			if strings.Contains(line.Text, "← prompt from docs") {
				marked = true
			}
		}
		snap, err := m.Snapshot("api")
		return marked && err == nil && snap.Turns >= 1
	})

	messages, err := m.Messages("api", 20)
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}
	var joined strings.Builder
	for _, item := range messages {
		joined.WriteString(item.Text)
	}
	if !strings.Contains(joined.String(), `"name":"docs"`) {
		t.Fatalf("api did not list its neighbours: %s", joined.String())
	}
}

func TestASessionWithoutTheGrantCannotPromptAnother(t *testing.T) {
	t.Setenv("FAKECLAUDE_MODE", "mcp")
	m := withMCP(t)
	ctx := context.Background()

	dir := t.TempDir()
	if _, err := m.Spawn(ctx, Spec{Dir: dir, Name: "api"}); err != nil {
		t.Fatalf("Spawn api: %v", err)
	}
	if _, err := m.Spawn(ctx, Spec{Dir: dir, Name: "docs"}); err != nil {
		t.Fatalf("Spawn docs: %v", err)
	}

	runOneTurn(t, m, "docs", `send_message {"session":"api","text":"hello"}`)

	snap, err := m.Snapshot("api")
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if snap.Turns != 0 {
		t.Fatal("a session without the grant prompted its neighbour")
	}
}

func TestAControlSessionCreatesASession(t *testing.T) {
	t.Setenv("FAKECLAUDE_MODE", "mcp")
	m := withMCP(t)

	if _, err := m.Spawn(context.Background(), Spec{Dir: t.TempDir(), Name: "boss", Control: true}); err != nil {
		t.Fatalf("Spawn boss: %v", err)
	}
	dir := t.TempDir()
	runOneTurn(t, m, "boss", `create_session {"path":"`+dir+`","name":"api"}`)

	waitFor(t, 10*time.Second, func() bool { return liveNamed(m, "api") })
}

func TestCreateSessionNeedsAPath(t *testing.T) {
	t.Setenv("FAKECLAUDE_MODE", "mcp")
	m := withMCP(t)

	if _, err := m.Spawn(context.Background(), Spec{Dir: t.TempDir(), Name: "boss", Control: true}); err != nil {
		t.Fatalf("Spawn boss: %v", err)
	}
	runOneTurn(t, m, "boss", `create_session {"path":"   "}`)

	if !mentions(t, m, "boss", "needs a directory path") {
		t.Fatal("boss did not see the no-path error")
	}
	if len(m.Snapshots()) != 1 {
		t.Fatal("a create with no path started a session")
	}
}

func TestCreateSessionRejectsAMissingPath(t *testing.T) {
	t.Setenv("FAKECLAUDE_MODE", "mcp")
	m := withMCP(t)

	if _, err := m.Spawn(context.Background(), Spec{Dir: t.TempDir(), Name: "boss", Control: true}); err != nil {
		t.Fatalf("Spawn boss: %v", err)
	}
	missing := filepath.Join(t.TempDir(), "nope")
	runOneTurn(t, m, "boss", `create_session {"path":"`+missing+`"}`)

	if !mentions(t, m, "boss", "not a directory") {
		t.Fatal("boss did not see the not-a-directory error")
	}
	if len(m.Snapshots()) != 1 {
		t.Fatal("a create with a missing path started a session")
	}
}

func TestASessionWithoutTheGrantCannotCreate(t *testing.T) {
	t.Setenv("FAKECLAUDE_MODE", "mcp")
	m := withMCP(t)

	if _, err := m.Spawn(context.Background(), Spec{Dir: t.TempDir(), Name: "boss"}); err != nil {
		t.Fatalf("Spawn boss: %v", err)
	}
	runOneTurn(t, m, "boss", `create_session {"path":"`+t.TempDir()+`","name":"api"}`)

	if liveNamed(m, "api") {
		t.Fatal("a session without the grant created a neighbour")
	}
}

func liveNamed(m *Manager, name string) bool {
	for _, snap := range m.Snapshots() {
		if snap.Name == name {
			return true
		}
	}
	return false
}

func mentions(t *testing.T, m *Manager, name, want string) bool {
	t.Helper()
	messages, err := m.Messages(name, 20)
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}
	for _, item := range messages {
		if strings.Contains(item.Text, want) {
			return true
		}
	}
	return false
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func TestACreatedSessionRecordsItsCreator(t *testing.T) {
	t.Setenv("FAKECLAUDE_MODE", "mcp")
	m := withMCP(t)

	if _, err := m.Spawn(context.Background(), Spec{Dir: t.TempDir(), Name: "boss", Control: true}); err != nil {
		t.Fatalf("Spawn boss: %v", err)
	}
	runOneTurn(t, m, "boss", `create_session {"path":"`+t.TempDir()+`","name":"api"}`)

	waitFor(t, 10*time.Second, func() bool { return liveNamed(m, "api") })
	if got := m.Parents()["api"]; got != "boss" {
		t.Fatalf("Parents()[api] = %q, want boss", got)
	}
	if got := m.Parents()["boss"]; got != "" {
		t.Fatalf("Parents()[boss] = %q, want no creator", got)
	}
}

func TestTheCreatorSurvivesAResume(t *testing.T) {
	m := withMCP(t)
	ctx := context.Background()

	name, err := m.Spawn(ctx, Spec{Dir: t.TempDir(), Name: "api", Parent: "boss"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	runOneTurn(t, m, name, "hello")
	meta := waitForMeta(t, m, name)
	if meta.Parent != "boss" {
		t.Fatalf("meta.Parent = %q, want boss", meta.Parent)
	}
	retire(t, m, name)

	again, err := m.Resume(ctx, meta)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if got := m.Parents()[again]; got != "boss" {
		t.Fatalf("the resumed session lost its creator: %q", got)
	}
}

func withConfig(t *testing.T) (*Manager, string) {
	t.Helper()
	m := withMCP(t)
	path := filepath.Join(t.TempDir(), "settings", "config.json")
	m.opts.ConfigPaths = []string{path}
	return m, path
}

func TestSetEditorMakesTheSettingsFile(t *testing.T) {
	m, path := withConfig(t)
	yes := true

	got, err := m.SetEditor("nvim", &yes)
	if err != nil {
		t.Fatalf("SetEditor: %v", err)
	}
	if got != path {
		t.Fatalf("path = %q, want %q", got, path)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Editor != "nvim" {
		t.Fatalf("editor = %q, want nvim", cfg.Editor)
	}
	if cfg.EditorTerminal == nil || !*cfg.EditorTerminal {
		t.Fatalf("editorTerminal = %v, want true", cfg.EditorTerminal)
	}
}

func TestSetEditorKeepsTheFieldItIsNotGiven(t *testing.T) {
	m, path := withConfig(t)
	yes := true
	if _, err := m.SetEditor("nvim", &yes); err != nil {
		t.Fatalf("SetEditor: %v", err)
	}

	if _, err := m.SetEditor("code -n", nil); err != nil {
		t.Fatalf("SetEditor: %v", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Editor != "code -n" {
		t.Fatalf("editor = %q, want code -n", cfg.Editor)
	}
	if cfg.EditorTerminal == nil || !*cfg.EditorTerminal {
		t.Fatalf("editorTerminal = %v, want the value it had", cfg.EditorTerminal)
	}
}

func TestSetEditorThroughTheToolCarriesANotice(t *testing.T) {
	m, _ := withConfig(t)
	name, err := m.Spawn(context.Background(), Spec{Dir: t.TempDir(), Name: "docs"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	sub := m.Subscribe(256)
	defer sub.Close()

	tools := &bridge{m: m}
	if _, err := tools.SetEditor("zed", nil, name); err != nil {
		t.Fatalf("SetEditor: %v", err)
	}
	ev := awaitNotice(t, sub)
	if !strings.Contains(ev.Notice, "docs set the editor to zed") {
		t.Fatalf("notice = %q", ev.Notice)
	}
}

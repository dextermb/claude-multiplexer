package manager

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

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

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

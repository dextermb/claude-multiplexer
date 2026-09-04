package mcp_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dextermb/claude-multiplexer/internal/mcp"
)

type fakeSessions struct {
	titles      map[string]string
	sent        []string
	stopped     []string
	archived    map[string]bool
	created     []string
	list        []mcp.Session
	messages    map[string][]mcp.Message
	jobs        map[string][]mcp.Job
	stoppedJobs []string
	failStop    error
	failStopJob error
}

func newFakeSessions() *fakeSessions {
	return &fakeSessions{
		titles:   make(map[string]string),
		archived: make(map[string]bool),
		messages: make(map[string][]mcp.Message),
		jobs:     make(map[string][]mcp.Job),
	}
}

func (f *fakeSessions) SetTitle(name, title string) error {
	f.titles[name] = title
	return nil
}

func (f *fakeSessions) SendFrom(target, from, text string) (int, error) {
	f.sent = append(f.sent, from+"->"+target+":"+text)
	return len(f.sent), nil
}

func (f *fakeSessions) Stop(_ context.Context, name, by string) error {
	if f.failStop != nil {
		return f.failStop
	}
	f.stopped = append(f.stopped, name)
	return nil
}

func (f *fakeSessions) Archive(name string, archived bool, by string) error {
	f.archived[name] = archived
	return nil
}

func (f *fakeSessions) Create(dir, name, by string) (string, error) {
	if name == "" {
		name = "session"
	}
	f.created = append(f.created, by+"->"+name+"@"+dir)
	return name, nil
}

func (f *fakeSessions) List() []mcp.Session { return f.list }

func (f *fakeSessions) Messages(name string, limit int) ([]mcp.Message, error) {
	items, ok := f.messages[name]
	if !ok {
		return nil, errors.New("unknown session: " + name)
	}
	if len(items) > limit {
		items = items[len(items)-limit:]
	}
	return items, nil
}

func (f *fakeSessions) Jobs(name string) ([]mcp.Job, error) {
	return f.jobs[name], nil
}

func (f *fakeSessions) StopJob(target, jobID, by string) (int, error) {
	if f.failStopJob != nil {
		return 0, f.failStopJob
	}
	f.stoppedJobs = append(f.stoppedJobs, by+"->"+target+":"+jobID)
	return len(f.stoppedJobs), nil
}

func startServer(t *testing.T, sessions mcp.Sessions) *mcp.Server {
	t.Helper()
	server := mcp.NewServer(sessions)
	if err := server.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Close(ctx)
	})
	return server
}

func connect(t *testing.T, server *mcp.Server, token string) *sdk.ClientSession {
	t.Helper()
	client := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "0"}, nil)
	transport := &sdk.StreamableClientTransport{
		Endpoint:   server.URL(),
		HTTPClient: &http.Client{Transport: bearerTransport{token: token}},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

type bearerTransport struct{ token string }

func (b bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(clone)
}

func call(t *testing.T, session *sdk.ClientSession, name string, args map[string]any) *sdk.CallToolResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	return result
}

func resultText(result *sdk.CallToolResult) string {
	var out strings.Builder
	for _, item := range result.Content {
		if text, ok := item.(*sdk.TextContent); ok {
			out.WriteString(text.Text)
		}
	}
	return out.String()
}

func TestRenameToolTitlesTheCaller(t *testing.T) {
	sessions := newFakeSessions()
	server := startServer(t, sessions)
	token, err := server.Register("docs", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	client := connect(t, server, token)
	result := call(t, client, mcp.ToolRename, map[string]any{"title": "Billing rewrite"})
	if result.IsError {
		t.Fatalf("rename failed: %s", resultText(result))
	}
	if sessions.titles["docs"] != "Billing rewrite" {
		t.Fatalf("title = %q, want %q", sessions.titles["docs"], "Billing rewrite")
	}
}

func TestUnknownTokenIsRejected(t *testing.T) {
	server := startServer(t, newFakeSessions())
	client := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "0"}, nil)
	transport := &sdk.StreamableClientTransport{
		Endpoint:   server.URL(),
		HTTPClient: &http.Client{Transport: bearerTransport{token: "not-a-token"}},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := client.Connect(ctx, transport, nil); err == nil {
		t.Fatal("an unknown token connected")
	}
}

func TestASessionWithoutControlSeesOnlyTheOpenTools(t *testing.T) {
	server := startServer(t, newFakeSessions())
	token, err := server.Register("docs", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	client := connect(t, server, token)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tools, err := client.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	got := make(map[string]bool)
	for _, tool := range tools.Tools {
		got[tool.Name] = true
	}
	for _, name := range mcp.OpenTools {
		if !got[name] {
			t.Errorf("open tool %s is missing", name)
		}
	}
	for _, name := range mcp.ControlTools {
		if got[name] {
			t.Errorf("control tool %s reached a session without the grant", name)
		}
	}
}

func TestControlToolsDriveOtherSessions(t *testing.T) {
	sessions := newFakeSessions()
	sessions.list = []mcp.Session{{Name: "api", Live: true}, {Name: "landing"}}
	server := startServer(t, sessions)
	token, err := server.Register("docs", true)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	client := connect(t, server, token)

	if result := call(t, client, mcp.ToolSend, map[string]any{"session": "api", "text": "take the billing work"}); result.IsError {
		t.Fatalf("send failed: %s", resultText(result))
	}
	if len(sessions.sent) != 1 || sessions.sent[0] != "docs->api:take the billing work" {
		t.Fatalf("sent = %v", sessions.sent)
	}

	if result := call(t, client, mcp.ToolStop, map[string]any{"session": "api"}); result.IsError {
		t.Fatalf("stop failed: %s", resultText(result))
	}
	if len(sessions.stopped) != 1 || sessions.stopped[0] != "api" {
		t.Fatalf("stopped = %v", sessions.stopped)
	}

	if result := call(t, client, mcp.ToolArchive, map[string]any{"session": "landing"}); result.IsError {
		t.Fatalf("archive failed: %s", resultText(result))
	}
	if !sessions.archived["landing"] {
		t.Fatal("landing was not archived")
	}
	if result := call(t, client, mcp.ToolArchive, map[string]any{"session": "landing", "restore": true}); result.IsError {
		t.Fatalf("restore failed: %s", resultText(result))
	}
	if sessions.archived["landing"] {
		t.Fatal("landing was not restored")
	}
}

func TestASessionCannotSendToItselfOrStopItself(t *testing.T) {
	sessions := newFakeSessions()
	server := startServer(t, sessions)
	token, err := server.Register("docs", true)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	client := connect(t, server, token)

	result := call(t, client, mcp.ToolSend, map[string]any{"session": "docs", "text": "hello"})
	if !result.IsError {
		t.Fatal("a session sent a prompt to itself")
	}
	if len(sessions.sent) != 0 {
		t.Fatalf("sent = %v", sessions.sent)
	}

	result = call(t, client, mcp.ToolStop, map[string]any{"session": "docs"})
	if !result.IsError {
		t.Fatal("a session stopped itself")
	}
	if len(sessions.stopped) != 0 {
		t.Fatalf("stopped = %v", sessions.stopped)
	}
}

func TestListAndMessagesReadTheOtherSessions(t *testing.T) {
	sessions := newFakeSessions()
	sessions.list = []mcp.Session{
		{Name: "api", Live: true, State: "busy"},
		{Name: "landing", State: "stored"},
	}
	sessions.messages["api"] = []mcp.Message{
		{Role: "user", Text: "one"},
		{Role: "assistant", Text: "two"},
		{Role: "assistant", Text: "three"},
	}
	server := startServer(t, sessions)
	token, err := server.Register("docs", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	client := connect(t, server, token)

	result := call(t, client, mcp.ToolList, map[string]any{"live_only": true})
	text := resultText(result)
	if !strings.Contains(text, "api") || strings.Contains(text, "landing") {
		t.Fatalf("live_only list = %s", text)
	}

	result = call(t, client, mcp.ToolMessages, map[string]any{"session": "api", "limit": 2})
	text = resultText(result)
	if strings.Contains(text, "one") || !strings.Contains(text, "three") {
		t.Fatalf("messages = %s", text)
	}

	result = call(t, client, mcp.ToolMessages, map[string]any{"session": "gone"})
	if !result.IsError {
		t.Fatal("an unknown session returned messages")
	}
}

func TestListJobsReadsSelfAndANeighbour(t *testing.T) {
	sessions := newFakeSessions()
	sessions.jobs["docs"] = []mcp.Job{{ID: "d1", Description: "own job", Status: "running", Running: true}}
	sessions.jobs["api"] = []mcp.Job{{ID: "a1", Description: "build", Status: "done"}}
	server := startServer(t, sessions)
	token, err := server.Register("docs", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	client := connect(t, server, token)

	self := resultText(call(t, client, mcp.ToolListJobs, map[string]any{}))
	if !strings.Contains(self, "d1") || strings.Contains(self, "a1") {
		t.Fatalf("self jobs = %s", self)
	}

	neighbour := resultText(call(t, client, mcp.ToolListJobs, map[string]any{"session": "api"}))
	if !strings.Contains(neighbour, "a1") || !strings.Contains(neighbour, "build") {
		t.Fatalf("neighbour jobs = %s", neighbour)
	}
}

func TestStopJobDrivesAnotherSessionAndSelf(t *testing.T) {
	sessions := newFakeSessions()
	server := startServer(t, sessions)
	token, err := server.Register("docs", true)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	client := connect(t, server, token)

	if result := call(t, client, mcp.ToolStopJob, map[string]any{"session": "api", "job": "a1"}); result.IsError {
		t.Fatalf("stop_job failed: %s", resultText(result))
	}
	if result := call(t, client, mcp.ToolStopJob, map[string]any{"job": "d1"}); result.IsError {
		t.Fatalf("self stop_job failed: %s", resultText(result))
	}
	want := []string{"docs->api:a1", "docs->docs:d1"}
	if len(sessions.stoppedJobs) != 2 || sessions.stoppedJobs[0] != want[0] || sessions.stoppedJobs[1] != want[1] {
		t.Fatalf("stoppedJobs = %v, want %v", sessions.stoppedJobs, want)
	}
}

func TestStopJobNeedsAJobID(t *testing.T) {
	sessions := newFakeSessions()
	server := startServer(t, sessions)
	token, err := server.Register("docs", true)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	client := connect(t, server, token)

	if result := call(t, client, mcp.ToolStopJob, map[string]any{"session": "api"}); !result.IsError {
		t.Fatal("stop_job ran with no job id")
	}
	if len(sessions.stoppedJobs) != 0 {
		t.Fatalf("stoppedJobs = %v", sessions.stoppedJobs)
	}
}

func TestConfigNamesTheServerAndTheToken(t *testing.T) {
	server := startServer(t, newFakeSessions())
	data, err := server.Config("abc123")
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	text := string(data)
	for _, want := range []string{`"mux"`, `"http"`, server.URL(), "Bearer abc123"} {
		if !strings.Contains(text, want) {
			t.Errorf("config is missing %q:\n%s", want, text)
		}
	}
}

func TestAllowedToolsFollowTheGrant(t *testing.T) {
	open := mcp.AllowedTools(false)
	if len(open) != len(mcp.OpenTools) {
		t.Fatalf("open tools = %v", open)
	}
	if open[0] != "mcp__mux__rename_session" {
		t.Fatalf("qualified name = %q", open[0])
	}
	control := mcp.AllowedTools(true)
	if len(control) != len(mcp.OpenTools)+len(mcp.ControlTools) {
		t.Fatalf("control tools = %v", control)
	}
	if !contains(control, "mcp__mux__create_session") {
		t.Fatalf("control tools lack create_session: %v", control)
	}
	if contains(open, "mcp__mux__create_session") {
		t.Fatalf("open tools include create_session: %v", open)
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

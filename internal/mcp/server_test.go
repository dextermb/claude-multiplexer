package mcp_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dextermb/claude-multiplexer/internal/mcp"
)

type fakeSessions struct {
	titles        map[string]string
	sent          []string
	stopped       []string
	archived      map[string]bool
	created       []string
	list          []mcp.Session
	messages      map[string][]mcp.Message
	jobs          map[string][]mcp.Job
	stoppedJobs   []string
	configSet     map[string]json.RawMessage
	configUnset   []string
	editor        string
	terminal      *bool
	editorPath    string
	cleared       []string
	blockCap      *int
	blockCaps     map[string]*int
	workingDir    string
	project       []string
	layouts       map[string]mcp.LayoutDims
	activeLayout  string
	sessionLayout map[string]string
	failWorkDir   error
	failStop      error
	failStopJob   error
	schedules     map[string]mcp.Schedule
	scheduleRuns  []string
	failSchedule  error
	lastControl   bool
}

func newFakeSessions() *fakeSessions {
	return &fakeSessions{
		titles:        make(map[string]string),
		archived:      make(map[string]bool),
		messages:      make(map[string][]mcp.Message),
		jobs:          make(map[string][]mcp.Job),
		layouts:       make(map[string]mcp.LayoutDims),
		sessionLayout: make(map[string]string),
		schedules:     make(map[string]mcp.Schedule),
	}
}

func (f *fakeSessions) Layouts(session string) (mcp.LayoutList, error) {
	out := mcp.LayoutList{Session: session, ActiveGlobal: f.activeLayout, ActiveSession: f.sessionLayout[session]}
	for name, dims := range f.layouts {
		out.Layouts = append(out.Layouts, mcp.LayoutInfo{Name: name, LayoutDims: dims})
	}
	return out, nil
}

func (f *fakeSessions) SaveLayout(name string, dims mcp.LayoutDims, by string) (string, error) {
	f.layouts[name] = dims
	return "/tmp/config.json", nil
}

func (f *fakeSessions) DeleteLayout(name, by string) (string, bool, error) {
	_, ok := f.layouts[name]
	delete(f.layouts, name)
	return "/tmp/config.json", ok, nil
}

func (f *fakeSessions) SetLayout(name, scope, by string) (string, error) {
	if scope == mcp.ScopeAll {
		f.activeLayout = name
		return "/tmp/config.json", nil
	}
	f.sessionLayout[by] = name
	return "", nil
}

func (f *fakeSessions) UnsetLayout(scope, by string) (string, bool, error) {
	if scope == mcp.ScopeAll {
		had := f.activeLayout != ""
		f.activeLayout = ""
		return "/tmp/config.json", had, nil
	}
	_, had := f.sessionLayout[by]
	delete(f.sessionLayout, by)
	return "", had, nil
}

func (f *fakeSessions) SetConfig(path string, value json.RawMessage, by string) (string, error) {
	if f.configSet == nil {
		f.configSet = make(map[string]json.RawMessage)
	}
	f.configSet[path] = value
	return "/tmp/config.json", nil
}

func (f *fakeSessions) UnsetConfig(path, by string) (string, bool, error) {
	f.configUnset = append(f.configUnset, path)
	return "/tmp/config.json", true, nil
}

func (f *fakeSessions) SetEditor(editor string, terminal *bool, by string) (string, error) {
	if editor != "" {
		f.editor = editor
	}
	if terminal != nil {
		f.terminal = terminal
	}
	f.editorPath = "/tmp/config.json"
	return f.editorPath, nil
}

func (f *fakeSessions) ConfigPath() mcp.ConfigPath {
	return mcp.ConfigPath{
		Paths:  []string{"/tmp/claude-multiplexer/config.json", "/tmp/multiplexier/config.json"},
		Active: "/tmp/multiplexier/config.json",
		Target: "/tmp/multiplexier/config.json",
	}
}

func (f *fakeSessions) SchedulePath() mcp.SchedulePath {
	return mcp.SchedulePath{Dir: "/home/dexter/.claude-multiplexer/schedules"}
}

func (f *fakeSessions) TemplatePath(name string) (mcp.TemplatePath, error) {
	if name != "docs" {
		return mcp.TemplatePath{}, errors.New("no such session")
	}
	return mcp.TemplatePath{
		Session: name,
		Root:    "/home/dexter/.claude-multiplexer",
		Dir:     "/work/api",
		Dirs: []string{
			"/home/dexter/.claude-multiplexer/templates",
			"/work/api/.multiplexer/templates",
		},
	}, nil
}

func (f *fakeSessions) SetBlockCap(bucket string, rows *int, by string) (string, error) {
	if bucket == "" {
		f.blockCap = rows
		return "/tmp/config.json", nil
	}
	if f.blockCaps == nil {
		f.blockCaps = map[string]*int{}
	}
	f.blockCaps[bucket] = rows
	return "/tmp/config.json", nil
}

func (f *fakeSessions) UnsetBlockCap(bucket, by string) (string, bool, error) {
	if bucket == "" {
		changed := f.blockCap != nil
		f.blockCap = nil
		return "/tmp/config.json", changed, nil
	}
	_, changed := f.blockCaps[bucket]
	delete(f.blockCaps, bucket)
	return "/tmp/config.json", changed, nil
}

func (f *fakeSessions) UnsetEditor(field, by string) (string, bool, error) {
	f.cleared = append(f.cleared, field)
	changed := f.editor != "" || f.terminal != nil
	switch field {
	case "editor":
		f.editor = ""
	case "terminal":
		f.terminal = nil
	default:
		f.editor = ""
		f.terminal = nil
	}
	return "/tmp/config.json", changed, nil
}

func (f *fakeSessions) SetWorkingDir(path, by string) (string, error) {
	if f.failWorkDir != nil {
		return "", f.failWorkDir
	}
	f.workingDir = "/repo/" + path
	return f.workingDir, nil
}

func (f *fakeSessions) UnsetWorkingDir(by string) (bool, error) {
	if f.workingDir == "" {
		return false, nil
	}
	f.workingDir = ""
	return true, nil
}

func (f *fakeSessions) Project(session string) ([]string, error) {
	return f.project, nil
}

func (f *fakeSessions) SetProject(paths []string, by string) ([]string, error) {
	if f.failWorkDir != nil {
		return nil, f.failWorkDir
	}
	f.project = nil
	for _, path := range paths {
		f.project = append(f.project, "/repo/"+path)
	}
	return f.project, nil
}

func (f *fakeSessions) AddProjectDir(path, by string) ([]string, error) {
	if f.failWorkDir != nil {
		return nil, f.failWorkDir
	}
	f.project = append(f.project, "/repo/"+path)
	return f.project, nil
}

func (f *fakeSessions) RemoveProjectDir(path, by string) ([]string, error) {
	full := "/repo/" + path
	var kept []string
	for _, dir := range f.project {
		if dir != full {
			kept = append(kept, dir)
		}
	}
	f.project = kept
	return f.project, nil
}

func (f *fakeSessions) ClearProject(by string) (bool, error) {
	if len(f.project) == 0 {
		return false, nil
	}
	f.project = nil
	return true, nil
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

func (f *fakeSessions) CreateSchedule(in mcp.ScheduleInput, by string) (mcp.Schedule, error) {
	if f.failSchedule != nil {
		return mcp.Schedule{}, f.failSchedule
	}
	name := in.Name
	if name == "" {
		name = "schedule"
	}
	f.lastControl = in.Control
	sched := mcp.Schedule{Name: name, Cron: in.Cron, Dir: in.Dir, Prompt: in.Prompt, Session: in.Session, Model: in.Model, Enabled: true}
	f.schedules[name] = sched
	return sched, nil
}

func (f *fakeSessions) UpdateSchedule(name string, up mcp.ScheduleEdit, by string) (mcp.Schedule, error) {
	if f.failSchedule != nil {
		return mcp.Schedule{}, f.failSchedule
	}
	sched, ok := f.schedules[name]
	if !ok {
		return mcp.Schedule{}, errors.New("unknown schedule: " + name)
	}
	if up.Cron != nil {
		sched.Cron = *up.Cron
	}
	if up.Dir != nil {
		sched.Dir = *up.Dir
	}
	if up.Prompt != nil {
		sched.Prompt = *up.Prompt
	}
	if up.Session != nil {
		sched.Session = *up.Session
	}
	if up.Model != nil {
		sched.Model = *up.Model
	}
	if up.Control != nil {
		f.lastControl = *up.Control
	}
	f.schedules[name] = sched
	return sched, nil
}

func (f *fakeSessions) ListSchedules() []mcp.Schedule {
	out := make([]mcp.Schedule, 0, len(f.schedules))
	for _, sched := range f.schedules {
		out = append(out, sched)
	}
	return out
}

func (f *fakeSessions) DeleteSchedule(name, by string) (bool, error) {
	if f.failSchedule != nil {
		return false, f.failSchedule
	}
	_, ok := f.schedules[name]
	delete(f.schedules, name)
	return ok, nil
}

func (f *fakeSessions) SetScheduleEnabled(name string, on bool, by string) (mcp.Schedule, error) {
	if f.failSchedule != nil {
		return mcp.Schedule{}, f.failSchedule
	}
	sched := f.schedules[name]
	sched.Name = name
	sched.Enabled = on
	f.schedules[name] = sched
	return sched, nil
}

func (f *fakeSessions) RunSchedule(name, by string) (string, error) {
	if f.failSchedule != nil {
		return "", f.failSchedule
	}
	f.scheduleRuns = append(f.scheduleRuns, by+"->"+name)
	return name + "-run", nil
}

func (f *fakeSessions) CreateAPIAdmin() (string, error) { return "admin-secret", nil }

func (f *fakeSessions) RotateAPIAdmin() (string, error) { return "admin-secret-2", nil }

func (f *fakeSessions) RevokeAPIAdmin() error { return nil }

func (f *fakeSessions) CreateAPIClient(name string) (mcp.APIClient, string, error) {
	return mcp.APIClient{ClientID: "client-1", Name: name}, "client-secret", nil
}

func (f *fakeSessions) UpdateAPIClient(id string, name *string, disabled *bool) (mcp.APIClient, error) {
	out := mcp.APIClient{ClientID: id}
	if name != nil {
		out.Name = *name
	}
	if disabled != nil {
		out.Disabled = *disabled
	}
	return out, nil
}

func (f *fakeSessions) RotateAPIClient(id string) (string, error) { return "client-secret-2", nil }

func (f *fakeSessions) RevokeAPIClient(id string) error { return nil }

func (f *fakeSessions) ListAPIClients() []mcp.APIClient { return nil }

func (f *fakeSessions) APIEndpoint() mcp.APIEndpoint {
	return mcp.APIEndpoint{URL: "http://127.0.0.1:0", PortStart: 51890, PortEnd: 51899}
}

func startServer(t *testing.T, sessions mcp.Sessions) *mcp.Server {
	t.Helper()
	server := mcp.NewServer(sessions)
	if err := server.Start(0, 0); err != nil {
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
	for _, want := range []string{`"cmux"`, `"http"`, server.URL(), "Bearer abc123"} {
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
	if open[0] != "mcp__cmux__rename_session" {
		t.Fatalf("qualified name = %q", open[0])
	}
	control := mcp.AllowedTools(true)
	if len(control) != len(mcp.OpenTools)+len(mcp.ControlTools) {
		t.Fatalf("control tools = %v", control)
	}
	if !contains(control, "mcp__cmux__create_session") {
		t.Fatalf("control tools lack create_session: %v", control)
	}
	if contains(open, "mcp__cmux__create_session") {
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

func TestSetConfigToolPassesTheRawValue(t *testing.T) {
	sessions := newFakeSessions()
	server := startServer(t, sessions)
	token, err := server.Register("docs", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	client := connect(t, server, token)
	result := call(t, client, mcp.ToolSetConfig, map[string]any{"path": "blockCaps.tool", "value": 3})
	if result.IsError {
		t.Fatalf("set_config failed: %s", resultText(result))
	}
	got, ok := sessions.configSet["blockCaps.tool"]
	if !ok {
		t.Fatal("set_config did not record the path")
	}
	if string(got) != "3" {
		t.Fatalf("value = %s, want 3", got)
	}
}

func TestSetConfigToolNeedsAPath(t *testing.T) {
	sessions := newFakeSessions()
	server := startServer(t, sessions)
	token, err := server.Register("docs", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	client := connect(t, server, token)
	if result := call(t, client, mcp.ToolSetConfig, map[string]any{"path": "  ", "value": 1}); !result.IsError {
		t.Fatal("set_config took an empty path, want an error")
	}
}

func TestUnsetConfigTool(t *testing.T) {
	sessions := newFakeSessions()
	server := startServer(t, sessions)
	token, err := server.Register("docs", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	client := connect(t, server, token)
	if result := call(t, client, mcp.ToolUnsetConfig, map[string]any{"path": "blockCap"}); result.IsError {
		t.Fatalf("unset_config failed: %s", resultText(result))
	}
	if len(sessions.configUnset) != 1 || sessions.configUnset[0] != "blockCap" {
		t.Fatalf("configUnset = %v, want [blockCap]", sessions.configUnset)
	}
}

func TestSetEditorToolWritesBothFields(t *testing.T) {
	sessions := newFakeSessions()
	server := startServer(t, sessions)
	token, err := server.Register("docs", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	client := connect(t, server, token)
	result := call(t, client, mcp.ToolSetEditor, map[string]any{"editor": "code -n", "terminal": false})
	if result.IsError {
		t.Fatalf("set_editor failed: %s", resultText(result))
	}
	if sessions.editor != "code -n" {
		t.Fatalf("editor = %q, want %q", sessions.editor, "code -n")
	}
	if sessions.terminal == nil || *sessions.terminal {
		t.Fatalf("terminal = %v, want false", sessions.terminal)
	}
	if !strings.Contains(resultText(result), "/tmp/config.json") {
		t.Fatalf("the answer does not name the file:\n%s", resultText(result))
	}
}

func TestSetEditorToolKeepsTheFieldItIsNotGiven(t *testing.T) {
	sessions := newFakeSessions()
	server := startServer(t, sessions)
	token, err := server.Register("docs", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	client := connect(t, server, token)
	if result := call(t, client, mcp.ToolSetEditor, map[string]any{"terminal": true}); result.IsError {
		t.Fatalf("set_editor failed: %s", resultText(result))
	}
	if sessions.editor != "" {
		t.Fatalf("editor = %q, want it untouched", sessions.editor)
	}
	if sessions.terminal == nil || !*sessions.terminal {
		t.Fatalf("terminal = %v, want true", sessions.terminal)
	}
}

func TestSetEditorToolNeedsOneField(t *testing.T) {
	sessions := newFakeSessions()
	server := startServer(t, sessions)
	token, err := server.Register("docs", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	client := connect(t, server, token)
	result := call(t, client, mcp.ToolSetEditor, map[string]any{})
	if !result.IsError {
		t.Fatal("a call with no field must be an error")
	}
	if sessions.editorPath != "" {
		t.Fatal("a call with no field must write nothing")
	}
}

func TestEverySessionGetsTheEditorTool(t *testing.T) {
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
	for _, tool := range tools.Tools {
		if tool.Name == mcp.ToolSetEditor {
			return
		}
	}
	t.Fatal("a session without the grant must still see set_editor")
}

func TestUnsetEditorToolClearsBothFields(t *testing.T) {
	sessions := newFakeSessions()
	sessions.editor = "zed"
	yes := true
	sessions.terminal = &yes
	server := startServer(t, sessions)
	token, err := server.Register("docs", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	client := connect(t, server, token)
	result := call(t, client, mcp.ToolUnsetEditor, map[string]any{})
	if result.IsError {
		t.Fatalf("unset_editor failed: %s", resultText(result))
	}
	if sessions.editor != "" || sessions.terminal != nil {
		t.Fatalf("editor = %q, terminal = %v, want both cleared", sessions.editor, sessions.terminal)
	}
	if !strings.Contains(resultText(result), "no longer set") {
		t.Fatalf("the answer does not say what it did:\n%s", resultText(result))
	}
}

func TestUnsetEditorToolClearsOneField(t *testing.T) {
	sessions := newFakeSessions()
	sessions.editor = "zed"
	yes := true
	sessions.terminal = &yes
	server := startServer(t, sessions)
	token, err := server.Register("docs", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	client := connect(t, server, token)
	if result := call(t, client, mcp.ToolUnsetEditor, map[string]any{"field": "terminal"}); result.IsError {
		t.Fatalf("unset_editor failed: %s", resultText(result))
	}
	if sessions.editor != "zed" {
		t.Fatalf("editor = %q, want it untouched", sessions.editor)
	}
	if sessions.terminal != nil {
		t.Fatalf("terminal = %v, want it cleared", sessions.terminal)
	}
}

func TestUnsetEditorToolSaysWhenNothingWasSet(t *testing.T) {
	sessions := newFakeSessions()
	server := startServer(t, sessions)
	token, err := server.Register("docs", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	client := connect(t, server, token)
	result := call(t, client, mcp.ToolUnsetEditor, map[string]any{})
	if result.IsError {
		t.Fatalf("unset_editor failed: %s", resultText(result))
	}
	if !strings.Contains(resultText(result), "was not set") {
		t.Fatalf("the answer does not say that nothing was set:\n%s", resultText(result))
	}
}

func TestEverySessionGetsTheUnsetEditorTool(t *testing.T) {
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
	for _, tool := range tools.Tools {
		if tool.Name == mcp.ToolUnsetEditor {
			return
		}
	}
	t.Fatal("a session without the grant must still see unset_editor")
}

func workingDirClient(t *testing.T, sessions *fakeSessions) *sdk.ClientSession {
	t.Helper()
	server := startServer(t, sessions)
	token, err := server.Register("docs", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	return connect(t, server, token)
}

func TestSetWorkingDirToolPointsTheSessionAtADirectory(t *testing.T) {
	sessions := newFakeSessions()
	client := workingDirClient(t, sessions)

	result := call(t, client, mcp.ToolSetWorkingDir, map[string]any{"path": ".worktrees/feature"})
	if result.IsError {
		t.Fatalf("set_working_dir failed: %s", resultText(result))
	}
	if sessions.workingDir != "/repo/.worktrees/feature" {
		t.Fatalf("working directory = %q, want the resolved path", sessions.workingDir)
	}
	if !strings.Contains(resultText(result), "/repo/.worktrees/feature") {
		t.Fatalf("the answer does not name the directory:\n%s", resultText(result))
	}
}

func TestSetWorkingDirToolNeedsAPath(t *testing.T) {
	sessions := newFakeSessions()
	client := workingDirClient(t, sessions)

	if result := call(t, client, mcp.ToolSetWorkingDir, map[string]any{"path": "  "}); !result.IsError {
		t.Fatal("an empty path must be an error")
	}
	if sessions.workingDir != "" {
		t.Fatalf("working directory = %q, want none", sessions.workingDir)
	}
}

func TestSetWorkingDirToolPassesOnTheFailure(t *testing.T) {
	sessions := newFakeSessions()
	sessions.failWorkDir = errors.New("no such directory")
	client := workingDirClient(t, sessions)

	result := call(t, client, mcp.ToolSetWorkingDir, map[string]any{"path": "gone"})
	if !result.IsError {
		t.Fatal("a directory that is not there must be an error")
	}
	if !strings.Contains(resultText(result), "no such directory") {
		t.Fatalf("the answer does not carry the reason:\n%s", resultText(result))
	}
}

func TestUnsetWorkingDirToolClearsIt(t *testing.T) {
	sessions := newFakeSessions()
	client := workingDirClient(t, sessions)

	if result := call(t, client, mcp.ToolSetWorkingDir, map[string]any{"path": "sub"}); result.IsError {
		t.Fatalf("set_working_dir failed: %s", resultText(result))
	}
	result := call(t, client, mcp.ToolUnsetWorkingDir, map[string]any{})
	if result.IsError {
		t.Fatalf("unset_working_dir failed: %s", resultText(result))
	}
	if sessions.workingDir != "" {
		t.Fatalf("working directory = %q, want none", sessions.workingDir)
	}

	again := call(t, client, mcp.ToolUnsetWorkingDir, map[string]any{})
	if !strings.Contains(resultText(again), "had no working directory") {
		t.Fatalf("a second call must say that there was none:\n%s", resultText(again))
	}
}

func TestEverySessionGetsTheWorkingDirTools(t *testing.T) {
	client := workingDirClient(t, newFakeSessions())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tools, err := client.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}

	want := map[string]bool{mcp.ToolSetWorkingDir: false, mcp.ToolUnsetWorkingDir: false}
	for _, tool := range tools.Tools {
		if _, ok := want[tool.Name]; ok {
			want[tool.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("a session without the grant must still see %s", name)
		}
	}
}

func TestProjectToolsAddRemoveAndClear(t *testing.T) {
	sessions := newFakeSessions()
	client := workingDirClient(t, sessions)

	if result := call(t, client, mcp.ToolAddProjectDir, map[string]any{"path": "one"}); result.IsError {
		t.Fatalf("add_project_dir failed: %s", resultText(result))
	}
	if result := call(t, client, mcp.ToolAddProjectDir, map[string]any{"path": "two"}); result.IsError {
		t.Fatalf("add_project_dir failed: %s", resultText(result))
	}
	if len(sessions.project) != 2 {
		t.Fatalf("project = %v, want two directories", sessions.project)
	}

	list := call(t, client, mcp.ToolListProject, map[string]any{})
	if !strings.Contains(resultText(list), "2 directories") {
		t.Fatalf("list_project does not report the count:\n%s", resultText(list))
	}

	if result := call(t, client, mcp.ToolRemoveProject, map[string]any{"path": "one"}); result.IsError {
		t.Fatalf("remove_project_dir failed: %s", resultText(result))
	}
	if len(sessions.project) != 1 || sessions.project[0] != "/repo/two" {
		t.Fatalf("project = %v, want [/repo/two]", sessions.project)
	}

	if result := call(t, client, mcp.ToolClearProject, map[string]any{}); result.IsError {
		t.Fatalf("clear_project failed: %s", resultText(result))
	}
	if len(sessions.project) != 0 {
		t.Fatalf("project = %v, want none after clear", sessions.project)
	}
	again := call(t, client, mcp.ToolClearProject, map[string]any{})
	if !strings.Contains(resultText(again), "had no project") {
		t.Fatalf("a second clear must say there was none:\n%s", resultText(again))
	}
}

func TestSetProjectToolReplacesTheSet(t *testing.T) {
	sessions := newFakeSessions()
	client := workingDirClient(t, sessions)

	result := call(t, client, mcp.ToolSetProject, map[string]any{"paths": []any{"one", "two"}})
	if result.IsError {
		t.Fatalf("set_project failed: %s", resultText(result))
	}
	if len(sessions.project) != 2 || sessions.project[0] != "/repo/one" {
		t.Fatalf("project = %v, want the two resolved paths", sessions.project)
	}
}

func TestAddProjectDirToolNeedsAPath(t *testing.T) {
	sessions := newFakeSessions()
	client := workingDirClient(t, sessions)

	if result := call(t, client, mcp.ToolAddProjectDir, map[string]any{"path": "  "}); !result.IsError {
		t.Fatal("an empty path must be an error")
	}
	if len(sessions.project) != 0 {
		t.Fatalf("project = %v, want none", sessions.project)
	}
}

func TestEverySessionGetsTheProjectTools(t *testing.T) {
	client := workingDirClient(t, newFakeSessions())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tools, err := client.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}

	want := map[string]bool{
		mcp.ToolListProject: false, mcp.ToolAddProjectDir: false, mcp.ToolRemoveProject: false,
		mcp.ToolSetProject: false, mcp.ToolClearProject: false,
	}
	for _, tool := range tools.Tools {
		if _, ok := want[tool.Name]; ok {
			want[tool.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("a session without the grant must still see %s", name)
		}
	}
}

func TestSaveLayoutToolCarriesTheDiffPositionAndSize(t *testing.T) {
	sessions := newFakeSessions()
	server := startServer(t, sessions)
	token, err := server.Register("docs", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	client := connect(t, server, token)

	result := call(t, client, mcp.ToolSaveLayout, map[string]any{
		"name": "stack", "diffPosition": "bottom", "diffSize": 16,
	})
	if result.IsError {
		t.Fatalf("save_layout failed: %s", resultText(result))
	}
	dims, ok := sessions.layouts["stack"]
	if !ok {
		t.Fatal("the layout was not saved")
	}
	if dims.DiffPosition == nil || *dims.DiffPosition != "bottom" {
		t.Fatalf("diff position = %v, want bottom", dims.DiffPosition)
	}
	if dims.DiffSize == nil || *dims.DiffSize != 16 {
		t.Fatalf("diff size = %v, want 16", dims.DiffSize)
	}

	list := call(t, client, mcp.ToolListLayouts, map[string]any{})
	if list.IsError {
		t.Fatalf("list_layouts failed: %s", resultText(list))
	}
	if !strings.Contains(resultText(list), "bottom") {
		t.Fatalf("list_layouts must report the position:\n%s", resultText(list))
	}
}

func TestSaveLayoutToolRefusesABadDiffPosition(t *testing.T) {
	sessions := newFakeSessions()
	server := startServer(t, sessions)
	token, err := server.Register("docs", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	client := connect(t, server, token)

	if result := call(t, client, mcp.ToolSaveLayout, map[string]any{
		"name": "bad", "diffPosition": "sideways",
	}); !result.IsError {
		t.Fatal("an unknown diff position must be an error")
	}
	if _, ok := sessions.layouts["bad"]; ok {
		t.Fatal("a rejected layout must not be saved")
	}
}

func TestSetBlockCapToolWritesTheCap(t *testing.T) {
	sessions := newFakeSessions()
	server := startServer(t, sessions)
	token, err := server.Register("docs", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	client := connect(t, server, token)

	result := call(t, client, mcp.ToolSetBlockCap, map[string]any{"rows": 40})
	if result.IsError {
		t.Fatalf("set_block_cap failed: %s", resultText(result))
	}
	if sessions.blockCap == nil || *sessions.blockCap != 40 {
		t.Fatalf("blockCap = %v, want 40", sessions.blockCap)
	}
	if !strings.Contains(resultText(result), "/tmp/config.json") {
		t.Fatalf("the answer does not name the file:\n%s", resultText(result))
	}

	if result := call(t, client, mcp.ToolSetBlockCap, map[string]any{"rows": 0}); result.IsError {
		t.Fatalf("a cap of zero is allowed: %s", resultText(result))
	}
	if sessions.blockCap == nil || *sessions.blockCap != 0 {
		t.Fatalf("blockCap = %v, want 0", sessions.blockCap)
	}
}

func TestSetBlockCapToolRefusesALessThanZeroCap(t *testing.T) {
	sessions := newFakeSessions()
	server := startServer(t, sessions)
	token, err := server.Register("docs", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	client := connect(t, server, token)

	if result := call(t, client, mcp.ToolSetBlockCap, map[string]any{"rows": -2}); !result.IsError {
		t.Fatal("a cap below zero must be an error")
	}
	if sessions.blockCap != nil {
		t.Fatalf("blockCap = %v, want none written", sessions.blockCap)
	}
}

func TestUnsetBlockCapToolClearsTheCap(t *testing.T) {
	sessions := newFakeSessions()
	server := startServer(t, sessions)
	token, err := server.Register("docs", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	client := connect(t, server, token)

	if result := call(t, client, mcp.ToolSetBlockCap, map[string]any{"rows": 40}); result.IsError {
		t.Fatalf("set_block_cap failed: %s", resultText(result))
	}
	result := call(t, client, mcp.ToolUnsetBlockCap, map[string]any{})
	if result.IsError {
		t.Fatalf("unset_block_cap failed: %s", resultText(result))
	}
	if sessions.blockCap != nil {
		t.Fatalf("blockCap = %v, want none", sessions.blockCap)
	}
	if !strings.Contains(resultText(result), "20") {
		t.Fatalf("the answer must name the default:\n%s", resultText(result))
	}
}

func TestSetBlockCapToolWritesOneType(t *testing.T) {
	sessions := newFakeSessions()
	server := startServer(t, sessions)
	token, err := server.Register("docs", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	client := connect(t, server, token)

	if result := call(t, client, mcp.ToolSetBlockCap, map[string]any{"type": "meta", "rows": 0}); result.IsError {
		t.Fatalf("set_block_cap for a type failed: %s", resultText(result))
	}
	rows, ok := sessions.blockCaps["meta"]
	if !ok || rows == nil || *rows != 0 {
		t.Fatalf("meta cap = %v, want 0", sessions.blockCaps["meta"])
	}
	if sessions.blockCap != nil {
		t.Fatalf("the default must stay unset, got %v", sessions.blockCap)
	}
}

func TestSetBlockCapToolWritesAnUnlimitedType(t *testing.T) {
	sessions := newFakeSessions()
	server := startServer(t, sessions)
	token, err := server.Register("docs", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	client := connect(t, server, token)

	if result := call(t, client, mcp.ToolSetBlockCap, map[string]any{"type": "message", "unlimited": true}); result.IsError {
		t.Fatalf("set_block_cap unlimited failed: %s", resultText(result))
	}
	rows, ok := sessions.blockCaps["message"]
	if !ok || rows != nil {
		t.Fatalf("message cap = %v, want null", sessions.blockCaps["message"])
	}
}

func TestSetBlockCapToolRefusesAnUnknownType(t *testing.T) {
	sessions := newFakeSessions()
	server := startServer(t, sessions)
	token, err := server.Register("docs", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	client := connect(t, server, token)

	if result := call(t, client, mcp.ToolSetBlockCap, map[string]any{"type": "banana", "rows": 5}); !result.IsError {
		t.Fatal("an unknown type must be an error")
	}
}

func TestSetBlockCapToolRefusesRowsWithUnlimited(t *testing.T) {
	sessions := newFakeSessions()
	server := startServer(t, sessions)
	token, err := server.Register("docs", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	client := connect(t, server, token)

	result := call(t, client, mcp.ToolSetBlockCap, map[string]any{"type": "tool", "rows": 5, "unlimited": true})
	if !result.IsError {
		t.Fatal("rows with unlimited must be an error")
	}
}

func TestUnsetBlockCapToolClearsOneType(t *testing.T) {
	sessions := newFakeSessions()
	server := startServer(t, sessions)
	token, err := server.Register("docs", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	client := connect(t, server, token)

	if result := call(t, client, mcp.ToolSetBlockCap, map[string]any{"type": "bash", "rows": 3}); result.IsError {
		t.Fatalf("set_block_cap for a type failed: %s", resultText(result))
	}
	if result := call(t, client, mcp.ToolUnsetBlockCap, map[string]any{"type": "bash"}); result.IsError {
		t.Fatalf("unset_block_cap for a type failed: %s", resultText(result))
	}
	if _, ok := sessions.blockCaps["bash"]; ok {
		t.Fatalf("the bash cap must be cleared, got %v", sessions.blockCaps["bash"])
	}
}

func TestConfigPathToolNamesEveryFileInOrder(t *testing.T) {
	sessions := newFakeSessions()
	server := startServer(t, sessions)
	token, err := server.Register("docs", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	client := connect(t, server, token)

	result := call(t, client, mcp.ToolConfigPath, map[string]any{})
	if result.IsError {
		t.Fatalf("get_config_path failed: %s", resultText(result))
	}
	text := resultText(result)
	for _, want := range []string{"/tmp/claude-multiplexer/config.json", "/tmp/multiplexier/config.json", "active", "target"} {
		if !strings.Contains(text, want) {
			t.Errorf("the answer is missing %q:\n%s", want, text)
		}
	}
}

func TestTemplatePathToolReadsTheCallerByDefault(t *testing.T) {
	sessions := newFakeSessions()
	server := startServer(t, sessions)
	token, err := server.Register("docs", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	client := connect(t, server, token)

	result := call(t, client, mcp.ToolTemplatePath, map[string]any{})
	if result.IsError {
		t.Fatalf("get_template_path failed: %s", resultText(result))
	}
	text := resultText(result)
	for _, want := range []string{"docs", "/home/dexter/.claude-multiplexer/templates", "/work/api/.multiplexer/templates"} {
		if !strings.Contains(text, want) {
			t.Errorf("the answer is missing %q:\n%s", want, text)
		}
	}
}

func TestTemplatePathToolTakesAnotherSession(t *testing.T) {
	sessions := newFakeSessions()
	server := startServer(t, sessions)
	token, err := server.Register("api", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	client := connect(t, server, token)

	if result := call(t, client, mcp.ToolTemplatePath, map[string]any{"session": "docs"}); result.IsError {
		t.Fatalf("get_template_path failed: %s", resultText(result))
	}
	if result := call(t, client, mcp.ToolTemplatePath, map[string]any{"session": "nope"}); !result.IsError {
		t.Fatal("a session that is not there must be an error")
	}
}

func TestSchedulePathToolNamesTheDirectory(t *testing.T) {
	sessions := newFakeSessions()
	server := startServer(t, sessions)
	token, err := server.Register("docs", false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	client := connect(t, server, token)

	result := call(t, client, mcp.ToolSchedulePath, map[string]any{})
	if result.IsError {
		t.Fatalf("get_schedule_path failed: %s", resultText(result))
	}
	if text := resultText(result); !strings.Contains(text, "/home/dexter/.claude-multiplexer/schedules") {
		t.Errorf("the answer is missing the schedule directory:\n%s", text)
	}
}

func TestBothPathToolsAreOpenToEverySession(t *testing.T) {
	for _, name := range []string{mcp.ToolConfigPath, mcp.ToolTemplatePath, mcp.ToolSchedulePath} {
		if !contains(mcp.OpenTools, name) {
			t.Errorf("%s must be open to every session", name)
		}
	}
}

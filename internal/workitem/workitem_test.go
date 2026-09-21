package workitem_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/workitem"
)

// fakeState holds the current status of the one issue each fake server serves,
// so a set is seen by the next read.
type fakeState struct {
	mu     sync.Mutex
	status string
}

func (f *fakeState) get() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.status
}

func (f *fakeState) set(status string) {
	f.mu.Lock()
	f.status = status
	f.mu.Unlock()
}

func serve(t *testing.T, server *sdk.Server) string {
	t.Helper()
	handler := sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return server }, nil)
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	return ts.URL
}

func linearServer(state *fakeState) *sdk.Server {
	server := sdk.NewServer(&sdk.Implementation{Name: "linear", Version: "test"}, nil)
	type idIn struct {
		ID string `json:"id"`
	}
	sdk.AddTool(server, &sdk.Tool{Name: "get_issue"}, func(_ context.Context, _ *sdk.CallToolRequest, in idIn) (*sdk.CallToolResult, map[string]any, error) {
		return nil, map[string]any{
			"identifier": in.ID,
			"title":      "Fix the bug",
			"url":        "https://linear.app/acme/issue/" + in.ID,
			"state":      map[string]any{"name": state.get()},
			"team":       map[string]any{"id": "TEAM-1"},
		}, nil
	})
	type teamIn struct {
		Team string `json:"team"`
	}
	sdk.AddTool(server, &sdk.Tool{Name: "list_issue_statuses"}, func(_ context.Context, _ *sdk.CallToolRequest, _ teamIn) (*sdk.CallToolResult, []map[string]any, error) {
		return nil, []map[string]any{
			{"id": "s1", "name": "Backlog"},
			{"id": "s2", "name": "In Progress"},
			{"id": "s3", "name": "In Review"},
			{"id": "s4", "name": "Done"},
		}, nil
	})
	type saveIn struct {
		ID    string `json:"id"`
		State string `json:"state"`
	}
	sdk.AddTool(server, &sdk.Tool{Name: "save_issue"}, func(_ context.Context, _ *sdk.CallToolRequest, in saveIn) (*sdk.CallToolResult, map[string]any, error) {
		state.set(in.State)
		return nil, map[string]any{"ok": true}, nil
	})
	return server
}

func jiraServer(state *fakeState, resources any) *sdk.Server {
	server := sdk.NewServer(&sdk.Implementation{Name: "jira", Version: "test"}, nil)
	sdk.AddTool(server, &sdk.Tool{Name: "getAccessibleAtlassianResources"}, func(_ context.Context, _ *sdk.CallToolRequest, _ map[string]any) (*sdk.CallToolResult, any, error) {
		return nil, resources, nil
	})
	type issueIn struct {
		CloudID      string `json:"cloudId"`
		IssueIDOrKey string `json:"issueIdOrKey"`
	}
	sdk.AddTool(server, &sdk.Tool{Name: "getJiraIssue"}, func(_ context.Context, _ *sdk.CallToolRequest, _ issueIn) (*sdk.CallToolResult, map[string]any, error) {
		return nil, map[string]any{
			"key": "PROJ-1",
			"fields": map[string]any{
				"summary": "Fix the bug",
				"status":  map[string]any{"name": state.get()},
			},
		}, nil
	})
	sdk.AddTool(server, &sdk.Tool{Name: "getTransitionsForJiraIssue"}, func(_ context.Context, _ *sdk.CallToolRequest, _ issueIn) (*sdk.CallToolResult, map[string]any, error) {
		return nil, map[string]any{"transitions": []map[string]any{
			{"id": "11", "name": "Start", "to": map[string]any{"name": "In Progress"}},
			{"id": "21", "name": "Review", "to": map[string]any{"name": "In Review"}},
			{"id": "31", "name": "Finish", "to": map[string]any{"name": "Done"}},
		}}, nil
	})
	type transitionIn struct {
		CloudID      string         `json:"cloudId"`
		IssueIDOrKey string         `json:"issueIdOrKey"`
		Transition   map[string]any `json:"transition"`
	}
	sdk.AddTool(server, &sdk.Tool{Name: "transitionJiraIssue"}, func(_ context.Context, _ *sdk.CallToolRequest, in transitionIn) (*sdk.CallToolResult, map[string]any, error) {
		switch in.Transition["id"] {
		case "11":
			state.set("In Progress")
		case "21":
			state.set("In Review")
		case "31":
			state.set("Done")
		}
		return nil, map[string]any{"ok": true}, nil
	})
	return server
}

func linearSet(t *testing.T, state *fakeState) *workitem.Set {
	url := serve(t, linearServer(state))
	return workitem.NewSet(&config.WorkItems{Linear: &config.WorkItemProvider{Token: "t", URL: url}})
}

// jiraResourcesV1 is the flat array an older Atlassian MCP endpoint returns.
var jiraResourcesV1 = []map[string]any{{"id": "cloud-1", "url": "https://acme.atlassian.net"}}

// jiraResourcesV2 is the shape the Rovo v2 endpoint returns: the list nested
// under data.resources, and the id named cloudId.
var jiraResourcesV2 = map[string]any{"data": map[string]any{"resources": []map[string]any{{
	"cloudId":  "cloud-1",
	"url":      "https://acme.atlassian.net",
	"products": []map[string]any{{"id": "jira", "access": "read-write"}},
}}}}

func jiraSet(t *testing.T, state *fakeState) *workitem.Set {
	return jiraSetWith(t, state, jiraResourcesV1)
}

func jiraSetWith(t *testing.T, state *fakeState, resources any) *workitem.Set {
	url := serve(t, jiraServer(state, resources))
	return workitem.NewSet(&config.WorkItems{Jira: &config.WorkItemProvider{Token: "t", URL: url}})
}

func TestLinearResolve(t *testing.T) {
	state := &fakeState{status: "In Progress"}
	set := linearSet(t, state)
	item, err := set.Resolve(context.Background(), "", "LIN-1")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if item.Provider != "linear" || item.Key != "LIN-1" {
		t.Fatalf("provider/key: %+v", item)
	}
	if item.Title != "Fix the bug" || item.Status != "In Progress" {
		t.Fatalf("title/status: %+v", item)
	}
	if item.URL == "" {
		t.Fatalf("want a url, got none")
	}
}

func TestLinearSetStatus(t *testing.T) {
	state := &fakeState{status: "Backlog"}
	set := linearSet(t, state)
	item, err := set.SetStatus(context.Background(), "linear", "LIN-1", "in review")
	if err != nil {
		t.Fatalf("set: %v", err)
	}
	if item.Status != "In Review" {
		t.Fatalf("want In Review, got %q", item.Status)
	}
	if state.get() != "In Review" {
		t.Fatalf("server status not moved: %q", state.get())
	}
}

func TestLinearSetStatusRejectsUnknown(t *testing.T) {
	state := &fakeState{status: "Backlog"}
	set := linearSet(t, state)
	_, err := set.SetStatus(context.Background(), "", "LIN-1", "Shipped")
	if err == nil {
		t.Fatal("want an error for an unknown status")
	}
	if state.get() != "Backlog" {
		t.Fatalf("status changed on a bad target: %q", state.get())
	}
}

func TestJiraStatusesAndSet(t *testing.T) {
	state := &fakeState{status: "To Do"}
	set := jiraSet(t, state)
	statuses, err := set.Statuses(context.Background(), "jira", "PROJ-1")
	if err != nil {
		t.Fatalf("statuses: %v", err)
	}
	if _, ok := findName(statuses, "In Review"); !ok {
		t.Fatalf("want In Review among %+v", statuses)
	}
	item, err := set.SetStatus(context.Background(), "jira", "PROJ-1", "In Review")
	if err != nil {
		t.Fatalf("set: %v", err)
	}
	if item.Status != "In Review" {
		t.Fatalf("want In Review, got %q", item.Status)
	}
	if item.URL != "https://acme.atlassian.net/browse/PROJ-1" {
		t.Fatalf("browse url: %q", item.URL)
	}
}

func TestSetProviderSelection(t *testing.T) {
	disabled := workitem.NewSet(nil)
	if disabled.Enabled() {
		t.Fatal("nil config must be disabled")
	}
	if _, err := disabled.Resolve(context.Background(), "", "X-1"); err == nil {
		t.Fatal("want ErrDisabled")
	}

	both := workitem.NewSet(&config.WorkItems{
		Jira:   &config.WorkItemProvider{Token: "t"},
		Linear: &config.WorkItemProvider{Token: "t"},
	})
	if _, err := both.Resolve(context.Background(), "", "X-1"); err != workitem.ErrNoProvider {
		t.Fatalf("want ErrNoProvider with two providers, got %v", err)
	}
	if _, err := both.Resolve(context.Background(), "bitbucket", "X-1"); err != workitem.ErrUnknownProvider {
		t.Fatalf("want ErrUnknownProvider, got %v", err)
	}
}

func findName(list []workitem.Status, name string) (workitem.Status, bool) {
	for _, s := range list {
		if s.Name == name {
			return s, true
		}
	}
	return workitem.Status{}, false
}

func TestJiraResolveAcrossResourceShapes(t *testing.T) {
	for name, resources := range map[string]any{"v1": jiraResourcesV1, "rovo-v2": jiraResourcesV2} {
		t.Run(name, func(t *testing.T) {
			set := jiraSetWith(t, &fakeState{status: "To Do"}, resources)
			item, err := set.Resolve(context.Background(), "jira", "PROJ-1")
			if err != nil {
				t.Fatalf("resolve: %v", err)
			}
			if item.Status != "To Do" || item.Title != "Fix the bug" {
				t.Fatalf("title/status: %+v", item)
			}
			if item.URL != "https://acme.atlassian.net/browse/PROJ-1" {
				t.Fatalf("url: %q", item.URL)
			}
		})
	}
}

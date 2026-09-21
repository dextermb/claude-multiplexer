package workitem_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/workitem"
)

// jiraRESTServer serves the three Jira REST routes the transport calls, and
// holds the status of the one issue so a set is seen by the next read.
func jiraRESTServer(t *testing.T, state *fakeState) (string, *[]string) {
	t.Helper()
	var auth []string
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/3/issue/PROJ-1", func(w http.ResponseWriter, r *http.Request) {
		auth = append(auth, r.Header.Get("Authorization"))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"key": "PROJ-1",
			"fields": map[string]any{
				"summary": "Fix the bug",
				"status":  map[string]any{"name": state.get()},
			},
		})
	})
	mux.HandleFunc("/rest/api/3/issue/PROJ-1/transitions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var body struct {
				Transition struct {
					ID string `json:"id"`
				} `json:"transition"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			switch body.Transition.ID {
			case "11":
				state.set("In Progress")
			case "21":
				state.set("In Review")
			case "31":
				state.set("Done")
			default:
				http.Error(w, "unknown transition", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"transitions": []map[string]any{
			{"id": "11", "name": "Start", "to": map[string]any{"name": "In Progress"}},
			{"id": "21", "name": "Review", "to": map[string]any{"name": "In Review"}},
			{"id": "31", "name": "Finish", "to": map[string]any{"name": "Done"}},
		}})
	})
	mux.HandleFunc("/rest/api/3/issue/MISSING-1", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errorMessages": []string{"Issue does not exist or you do not have permission to see it."},
			"errors":        map[string]string{},
		})
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts.URL, &auth
}

func jiraRESTSet(t *testing.T, state *fakeState) (*workitem.Set, *[]string) {
	t.Helper()
	url, auth := jiraRESTServer(t, state)
	set := workitem.NewSet(&config.WorkItems{Jira: &config.WorkItemProvider{
		Token: "t",
		Email: "user@acme.com",
		URL:   url,
	}})
	return set, auth
}

func TestJiraRESTResolve(t *testing.T) {
	set, auth := jiraRESTSet(t, &fakeState{status: "To Do"})
	item, err := set.Resolve(context.Background(), "jira", "PROJ-1")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if item.Provider != "jira" || item.Key != "PROJ-1" {
		t.Fatalf("provider/key: %+v", item)
	}
	if item.Title != "Fix the bug" || item.Status != "To Do" {
		t.Fatalf("title/status: %+v", item)
	}
	if !strings.HasSuffix(item.URL, "/browse/PROJ-1") {
		t.Fatalf("url: %q", item.URL)
	}
	if len(*auth) == 0 || !strings.HasPrefix((*auth)[0], "Basic ") {
		t.Fatalf("want basic auth, got %v", *auth)
	}
}

func TestJiraRESTStatuses(t *testing.T) {
	set, _ := jiraRESTSet(t, &fakeState{status: "To Do"})
	statuses, err := set.Statuses(context.Background(), "jira", "PROJ-1")
	if err != nil {
		t.Fatalf("statuses: %v", err)
	}
	want := []string{"In Progress", "In Review", "Done"}
	if len(statuses) != len(want) {
		t.Fatalf("want %d statuses, got %+v", len(want), statuses)
	}
	for i, name := range want {
		if statuses[i].Name != name {
			t.Fatalf("status %d: want %q, got %q", i, name, statuses[i].Name)
		}
	}
}

func TestJiraRESTSetStatus(t *testing.T) {
	state := &fakeState{status: "To Do"}
	set, _ := jiraRESTSet(t, state)
	item, err := set.SetStatus(context.Background(), "jira", "PROJ-1", "in review")
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

func TestJiraRESTSetStatusRejectsUnknown(t *testing.T) {
	state := &fakeState{status: "To Do"}
	set, _ := jiraRESTSet(t, state)
	_, err := set.SetStatus(context.Background(), "jira", "PROJ-1", "Shipped")
	if err == nil {
		t.Fatal("want an error for an unknown status")
	}
	if state.get() != "To Do" {
		t.Fatalf("status changed on a bad target: %q", state.get())
	}
}

func TestJiraRESTReportsTheJiraError(t *testing.T) {
	set, _ := jiraRESTSet(t, &fakeState{status: "To Do"})
	_, err := set.Resolve(context.Background(), "jira", "MISSING-1")
	if err == nil {
		t.Fatal("want an error for a missing issue")
	}
	if !strings.Contains(err.Error(), "Issue does not exist") {
		t.Fatalf("want the Jira message, got %v", err)
	}
}

// TestJiraEndpointSelectsTheTransport pins the rule that picks a transport: a
// url whose path ends in /mcp is the MCP server, and any other url is a site
// base the REST transport reads.
func TestJiraEndpointSelectsTheTransport(t *testing.T) {
	state := &fakeState{status: "To Do"}
	mcpURL := serve(t, jiraServer(state, jiraResourcesV1))
	restURL, _ := jiraRESTServer(t, state)

	restSet := workitem.NewSet(&config.WorkItems{Jira: &config.WorkItemProvider{Token: "t", Email: "u@acme.com", URL: restURL}})
	if _, err := restSet.Resolve(context.Background(), "jira", "PROJ-1"); err != nil {
		t.Fatalf("a site base must use the REST transport: %v", err)
	}

	mcpSet := workitem.NewSet(&config.WorkItems{Jira: &config.WorkItemProvider{Token: "t", URL: mcpURL}})
	if _, err := mcpSet.Resolve(context.Background(), "jira", "PROJ-1"); err != nil {
		t.Fatalf("a /mcp url must use the MCP transport: %v", err)
	}
}

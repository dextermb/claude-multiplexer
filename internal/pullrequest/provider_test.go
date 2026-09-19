package pullrequest

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildGitHubDeclaresOneAliasPerQuery(t *testing.T) {
	queries := []Query{
		{Repo: Repo{Owner: "o", Name: "n"}, Branch: "feat"},
		{Repo: Repo{Owner: "o2", Name: "n2"}, Branch: "fix"},
	}
	query, vars := buildGitHub(queries)
	for _, want := range []string{"a0: repository", "a1: repository", "$o0", "$b1"} {
		if !strings.Contains(query, want) {
			t.Errorf("query missing %q:\n%s", want, query)
		}
	}
	if vars["o0"] != "o" || vars["b0"] != "feat" || vars["n1"] != "n2" {
		t.Fatalf("vars = %+v", vars)
	}
}

func TestParseGitHubReadsStatesAndUnresolved(t *testing.T) {
	data := json.RawMessage(`{
		"a0": {"pullRequests": {"nodes": [{"number": 1045, "url": "u", "state": "OPEN", "isDraft": false, "title": "t",
			"reviewThreads": {"nodes": [{"isResolved": false}, {"isResolved": true}, {"isResolved": false}]}}]}},
		"a1": {"pullRequests": {"nodes": [{"number": 7, "state": "OPEN", "isDraft": true,
			"reviewThreads": {"nodes": [{"isResolved": false}]}}]}},
		"a2": {"pullRequests": {"nodes": [{"number": 3, "state": "MERGED",
			"reviewThreads": {"nodes": [{"isResolved": false}]}}]}},
		"a3": {"pullRequests": {"nodes": []}}
	}`)
	queries := make([]Query, 4)
	out := parseGitHub(data, queries)

	if !out[0].Found || out[0].PR.Number != 1045 || out[0].PR.State != StateOpen || out[0].PR.Unresolved != 2 {
		t.Errorf("a0 = %+v, want open 1045 with 2 unresolved", out[0])
	}
	if out[1].PR.State != StateDraft || out[1].PR.Unresolved != 1 {
		t.Errorf("a1 = %+v, want draft with 1 unresolved", out[1])
	}
	if out[2].PR.State != StateMerged || out[2].PR.Unresolved != 0 {
		t.Errorf("a2 = %+v, want merged with 0 unresolved", out[2])
	}
	if out[3].Found {
		t.Errorf("a3 = %+v, want not found", out[3])
	}
}

func TestParseGitLabReadsStatesAndUnresolved(t *testing.T) {
	data := json.RawMessage(`{
		"a0": {"mergeRequests": {"nodes": [{"iid": "1045", "webUrl": "u", "state": "opened", "draft": false, "title": "t",
			"resolvableDiscussionsCount": 5, "resolvedDiscussionsCount": 2}]}},
		"a1": {"mergeRequests": {"nodes": [{"iid": "9", "state": "merged",
			"resolvableDiscussionsCount": 3, "resolvedDiscussionsCount": 0}]}},
		"a2": {"mergeRequests": {"nodes": []}}
	}`)
	queries := make([]Query, 3)
	out := parseGitLab(data, queries)

	if !out[0].Found || out[0].PR.Number != 1045 || out[0].PR.State != StateOpen || out[0].PR.Unresolved != 3 {
		t.Errorf("a0 = %+v, want open 1045 with 3 unresolved", out[0])
	}
	if out[1].PR.State != StateMerged || out[1].PR.Unresolved != 0 {
		t.Errorf("a1 = %+v, want merged with 0 unresolved", out[1])
	}
	if out[2].Found {
		t.Errorf("a2 = %+v, want not found", out[2])
	}
}

func TestDecodeGraphQLReadsErrorsWhenDataIsAbsent(t *testing.T) {
	if _, err := decodeGraphQL([]byte(`{"errors":[{"message":"bad"}]}`)); err == nil {
		t.Fatal("want an error when the data is absent")
	}
	data, err := decodeGraphQL([]byte(`{"data":{"a0":null},"errors":[{"message":"partial"}]}`))
	if err != nil {
		t.Fatalf("partial data must survive an error: %v", err)
	}
	if string(data) != `{"a0":null}` {
		t.Fatalf("data = %s", data)
	}
}

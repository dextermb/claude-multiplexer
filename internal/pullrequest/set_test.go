package pullrequest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/config"
)

// fakeGraphQL answers a batched query: for every b{i} variable it returns a
// found node, numbered by the alias index. It records the requests it received.
type fakeGraphQL struct {
	mu       sync.Mutex
	requests int
	aliases  int
	kind     string // "github" | "gitlab"
}

func (f *fakeGraphQL) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Variables map[string]string `json:"variables"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		var indices []int
		for k := range body.Variables {
			if strings.HasPrefix(k, "b") {
				indices = append(indices, atoiSafe(strings.TrimPrefix(k, "b")))
			}
		}
		f.mu.Lock()
		f.requests++
		f.aliases += len(indices)
		f.mu.Unlock()

		data := map[string]json.RawMessage{}
		for _, i := range indices {
			var node string
			if f.kind == "gitlab" {
				node = fmt.Sprintf(`{"mergeRequests":{"nodes":[{"iid":"%d","state":"opened","resolvableDiscussionsCount":1,"resolvedDiscussionsCount":0}]}}`, 100+i)
			} else {
				node = fmt.Sprintf(`{"pullRequests":{"nodes":[{"number":%d,"state":"OPEN","reviewThreads":{"nodes":[{"isResolved":false}]}}]}}`, 100+i)
			}
			data[aliasOf(i)] = json.RawMessage(node)
		}
		payload, _ := json.Marshal(map[string]any{"data": data})
		w.Write(payload)
	}
}

func TestLookupGroupsDedupesAndChunks(t *testing.T) {
	gh := &fakeGraphQL{kind: "github"}
	gl := &fakeGraphQL{kind: "gitlab"}
	ghServer := httptest.NewServer(gh.handler())
	glServer := httptest.NewServer(gl.handler())
	defer ghServer.Close()
	defer glServer.Close()

	set := NewSet(&config.PullRequests{
		GitHub: &config.PRProvider{Token: "t", Mode: config.PullRequestModeAPI, URL: ghServer.URL},
		GitLab: &config.PRProvider{Token: "t", Mode: config.PullRequestModeAPI, URL: glServer.URL},
	})

	var refs []BranchRef
	// 25 distinct GitHub branches, to force two chunks (20 + 5).
	for i := 0; i < 25; i++ {
		refs = append(refs, BranchRef{RemoteURL: "git@github.com:owner/name.git", Branch: fmt.Sprintf("b%d", i)})
	}
	// A duplicate of the first GitHub ref, to test dedupe.
	refs = append(refs, BranchRef{RemoteURL: "git@github.com:owner/name.git", Branch: "b0"})
	// One GitLab ref, a separate group.
	refs = append(refs, BranchRef{RemoteURL: "https://gitlab.com/group/name.git", Branch: "mr"})
	// One unknown host, tracked by no provider.
	refs = append(refs, BranchRef{RemoteURL: "git@example.com:o/n.git", Branch: "x"})

	out := set.Lookup(context.Background(), refs)

	if len(out) != len(refs) {
		t.Fatalf("results = %d, want %d", len(out), len(refs))
	}
	if !out[0].Found || out[0].PR.Provider != config.PullRequestGitHub || out[0].PR.Unresolved != 1 {
		t.Errorf("first github result = %+v", out[0])
	}
	// The duplicate ref (index 25) resolves to the same PR as index 0.
	if out[25].PR.Number != out[0].PR.Number || !out[25].Found {
		t.Errorf("duplicate ref = %+v, want the same as %+v", out[25], out[0])
	}
	if !out[26].Found || out[26].PR.Provider != config.PullRequestGitLab {
		t.Errorf("gitlab result = %+v", out[26])
	}
	if out[27].Found {
		t.Errorf("unknown host = %+v, want not found", out[27])
	}

	// 26 GitHub refs dedupe to 25 slots, chunked 20 + 5 = two requests.
	if gh.requests != 2 || gh.aliases != 25 {
		t.Errorf("github requests = %d aliases = %d, want 2 and 25", gh.requests, gh.aliases)
	}
	if gl.requests != 1 || gl.aliases != 1 {
		t.Errorf("gitlab requests = %d aliases = %d, want 1 and 1", gl.requests, gl.aliases)
	}
}

func TestEnabledReadsTokenAndCLI(t *testing.T) {
	old := lookPath
	defer func() { lookPath = old }()
	lookPath = func(string) (string, error) { return "", errNoCLI }

	if NewSet(nil).Enabled() {
		t.Error("no config block must be disabled, even with a CLI")
	}
	set := NewSet(&config.PullRequests{GitHub: &config.PRProvider{Token: "t", Mode: config.PullRequestModeAPI}})
	if !set.Enabled() {
		t.Error("a token in api mode must enable the feature")
	}
	// A block with no token, in auto mode, needs the CLI on the PATH.
	cliOnly := NewSet(&config.PullRequests{GitHub: &config.PRProvider{Mode: config.PullRequestModeAuto}})
	if cliOnly.Enabled() {
		t.Error("a block with no token and no CLI must be disabled")
	}
	lookPath = func(string) (string, error) { return "/usr/bin/gh", nil }
	if !cliOnly.Enabled() {
		t.Error("a block with the CLI on the PATH must enable the feature in auto mode")
	}
}

type cliMissing struct{}

func (cliMissing) Error() string { return "not found" }

var errNoCLI = cliMissing{}

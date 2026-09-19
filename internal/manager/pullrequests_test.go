package manager

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/pullrequest"
)

func found(dir, branch, provider string, number, unresolved int) dirResult {
	return dirResult{
		Dir:    dir,
		Branch: branch,
		Res: pullrequest.Result{
			Found: true,
			PR: pullrequest.PR{
				Provider:   provider,
				Number:     number,
				State:      pullrequest.StateOpen,
				Unresolved: unresolved,
			},
		},
	}
}

func TestPRBadgesCarryTheURLAndDir(t *testing.T) {
	badges := PRBadges([]PRMirror{
		{Dir: "/repo/api", Number: 1, URL: "https://host/api/1"},
		{Dir: "/repo/web", Number: 0, URL: "https://host/web/0"},
	})

	if len(badges) != 1 {
		t.Fatalf("badges = %d, want one for the mirror with a number", len(badges))
	}
	if badges[0].URL != "https://host/api/1" || badges[0].Dir != "/repo/api" {
		t.Fatalf("badge = %+v, want the url and the code base of the mirror", badges[0])
	}
}

func TestNextPRsBuildsKeepsAndDrops(t *testing.T) {
	prev := map[string]PRMirror{
		"/a": {Dir: "/a", Provider: "github", Number: 7, State: "open"},
	}
	out := nextPRs([]dirResult{
		{Dir: "/a", Keep: true},
		found("/b", "feat", "gitlab", 42, 2),
		{Dir: "/c", Res: pullrequest.Result{Found: false}},
	}, prev)
	if len(out) != 2 {
		t.Fatalf("want 2 mirrors, got %d: %+v", len(out), out)
	}
	if out[0].Dir != "/a" || out[0].Number != 7 {
		t.Fatalf("a Keep must retain the prior mirror: %+v", out[0])
	}
	if out[1].Dir != "/b" || out[1].Number != 42 || out[1].Provider != "gitlab" {
		t.Fatalf("a found result must mirror: %+v", out[1])
	}
}

func TestApplyPullRequestsTracksEachCodeBase(t *testing.T) {
	m := newBridgeManager(t)
	name, err := m.Spawn(context.Background(), Spec{Dir: m.opts.Root, Name: "s"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	m.applyPullRequests(name, []dirResult{
		found("/a", "feat-a", "github", 10, 1),
		found("/b", "feat-b", "gitlab", 20, 0),
	})
	badges := m.PullRequests()[name]
	if len(badges) != 2 {
		t.Fatalf("want 2 badges, got %d: %+v", len(badges), badges)
	}
	if badges[0].Number != 10 || badges[1].Number != 20 {
		t.Fatalf("badges out of order: %+v", badges)
	}

	// A transient error on one code base keeps its mirror.
	m.applyPullRequests(name, []dirResult{
		{Dir: "/a", Res: pullrequest.Result{Err: errors.New("network")}},
		found("/b", "feat-b", "gitlab", 20, 3),
	})
	badges = m.PullRequests()[name]
	if len(badges) != 2 || badges[0].Number != 10 {
		t.Fatalf("a transient error must keep the mirror: %+v", badges)
	}
	if badges[1].Unresolved != 3 {
		t.Fatalf("the other code base must update: %+v", badges)
	}

	// A code base with no PR drops from the list.
	m.applyPullRequests(name, []dirResult{
		{Dir: "/a", Res: pullrequest.Result{Found: false}},
		found("/b", "feat-b", "gitlab", 20, 3),
	})
	badges = m.PullRequests()[name]
	if len(badges) != 1 || badges[0].Number != 20 {
		t.Fatalf("a not-found code base must drop: %+v", badges)
	}

	// A code base that leaves the project drops with it.
	m.applyPullRequests(name, []dirResult{
		found("/b", "feat-b", "gitlab", 20, 3),
	})
	if badges := m.PullRequests()[name]; len(badges) != 1 {
		t.Fatalf("a removed code base must not linger: %+v", badges)
	}
}

func TestMigratePRFoldsTheFlatFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "meta.json")
	old := `{
		"name": "s",
		"dir": "/repo",
		"pr_provider": "github",
		"pr_number": 1045,
		"pr_state": "open",
		"pr_unresolved": 3,
		"pr_branch": "feature/pr"
	}`
	if err := os.WriteFile(path, []byte(old), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	meta, err := ReadMeta(path)
	if err != nil {
		t.Fatalf("ReadMeta: %v", err)
	}
	if len(meta.PRs) != 1 {
		t.Fatalf("want 1 folded mirror, got %d", len(meta.PRs))
	}
	pr := meta.PRs[0]
	if pr.Number != 1045 || pr.Provider != "github" || pr.Branch != "feature/pr" || pr.Dir != "/repo" {
		t.Fatalf("fold = %+v", pr)
	}
}

func TestConfigurePullRequestRejectsAnUnknownProvider(t *testing.T) {
	m := newBridgeManager(t)
	if _, err := m.ConfigurePullRequest("bitbucket", "t", "", ""); !errors.Is(err, ErrUnknownPRProvider) {
		t.Fatalf("err = %v, want ErrUnknownPRProvider", err)
	}
}

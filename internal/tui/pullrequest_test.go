package tui

import (
	"strings"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/git"
	"github.com/dextermb/claude-multiplexer/internal/manager"
)

// prDiffModel builds a diff model with a project of code bases, each with a
// tracked pull request, and one changed file per code base.
func prDiffModel(prs []manager.PRBadge, groups []dirDiff) Model {
	m := diffModel()
	m.rows = []row{{name: "a", prs: prs}}
	m.diffs["a"] = projectDiff{groups: groups}
	return m
}

func TestOpenSelectedPROpensTheCurrentCodeBase(t *testing.T) {
	seen := recordLaunches(t)
	m := prDiffModel(
		[]manager.PRBadge{
			{Number: 1, URL: "https://host/api/1", Dir: "/repo/api"},
			{Number: 2, URL: "https://host/web/2", Dir: "/repo/web"},
		},
		[]dirDiff{
			{dir: "/repo/api", repo: true, files: []git.FileChange{{Status: "M", Path: "main.go"}}},
			{dir: "/repo/web", repo: true, files: []git.FileChange{{Status: "M", Path: "main.go"}}},
		},
	)
	m.diffSel = 1

	next, _ := m.openSelectedPR()
	m = next.(Model)

	if len(*seen) != 1 {
		t.Fatalf("opened %d pull requests, want 1", len(*seen))
	}
	if got := (*seen)[0].line; !strings.HasSuffix(got, "https://host/web/2") {
		t.Fatalf("opened %q, want the pull request of the selected file's code base", got)
	}
	if (*seen)[0].terminal {
		t.Error("a pull request must not take the terminal")
	}
	if m.errText != "" {
		t.Errorf("errText = %q, want none", m.errText)
	}
}

func TestOpenSelectedPRReportsNoPullRequest(t *testing.T) {
	seen := recordLaunches(t)
	m := prDiffModel(nil, []dirDiff{
		{dir: "/repo/api", repo: true, files: []git.FileChange{{Status: "M", Path: "main.go"}}},
	})

	next, cmd := m.openSelectedPR()
	m = next.(Model)

	if cmd != nil {
		t.Fatal("a code base with no pull request must start nothing")
	}
	if len(*seen) != 0 {
		t.Fatalf("opened %d pull requests, want none", len(*seen))
	}
	if !strings.Contains(m.errText, "no pull request") {
		t.Fatalf("errText = %q, want it to say there is no pull request", m.errText)
	}
}

func TestOpenSelectedPRReportsNoSelectedFile(t *testing.T) {
	seen := recordLaunches(t)
	m := prDiffModel(
		[]manager.PRBadge{{Number: 1, URL: "https://host/api/1", Dir: "/repo/api"}},
		[]dirDiff{{dir: "/repo/api", repo: true}},
	)

	next, cmd := m.openSelectedPR()
	m = next.(Model)

	if cmd != nil {
		t.Fatal("an empty diff must start nothing")
	}
	if len(*seen) != 0 {
		t.Fatalf("opened %d pull requests, want none", len(*seen))
	}
	if !strings.Contains(m.errText, "no file") {
		t.Fatalf("errText = %q, want it to say no file is selected", m.errText)
	}
}

func TestOpenAllPRsOpensEveryCodeBase(t *testing.T) {
	seen := recordLaunches(t)
	m := prDiffModel([]manager.PRBadge{
		{Number: 1, URL: "https://host/api/1", Dir: "/repo/api"},
		{Number: 2, URL: "https://host/web/2", Dir: "/repo/web"},
	}, nil)

	next, _ := m.openAllPRs()
	m = next.(Model)

	if len(*seen) != 2 {
		t.Fatalf("opened %d pull requests, want one for each code base", len(*seen))
	}
	if !strings.HasSuffix((*seen)[0].line, "https://host/api/1") || !strings.HasSuffix((*seen)[1].line, "https://host/web/2") {
		t.Fatalf("opened %v, want the api and the web pull request", *seen)
	}
	if m.errText != "" {
		t.Errorf("errText = %q, want none", m.errText)
	}
}

func TestOpenAllPRsSkipsEmptyAndDeduplicates(t *testing.T) {
	seen := recordLaunches(t)
	m := prDiffModel([]manager.PRBadge{
		{Number: 1, URL: "https://host/api/1", Dir: "/repo/api"},
		{Number: 1, URL: "https://host/api/1", Dir: "/repo/api-copy"},
		{Number: 0, URL: "", Dir: "/repo/none"},
	}, nil)

	next, _ := m.openAllPRs()
	m = next.(Model)

	if len(*seen) != 1 {
		t.Fatalf("opened %d pull requests, want one after the dedup and the skip", len(*seen))
	}
}

func TestOpenAllPRsReportsNone(t *testing.T) {
	seen := recordLaunches(t)
	m := prDiffModel(nil, nil)

	next, cmd := m.openAllPRs()
	m = next.(Model)

	if cmd != nil {
		t.Fatal("a session with no pull request must start nothing")
	}
	if len(*seen) != 0 {
		t.Fatalf("opened %d pull requests, want none", len(*seen))
	}
	if !strings.Contains(m.errText, "no pull requests") {
		t.Fatalf("errText = %q, want it to say there are no pull requests", m.errText)
	}
}

func TestThePullRequestKeysAreRegistered(t *testing.T) {
	for _, key := range []string{"d p", "d P"} {
		if sequenceActions[key] == nil {
			t.Errorf("no action for %q", key)
		}
	}
}

func TestTheDiffKeyListNamesThePullRequestKeys(t *testing.T) {
	m, _ := openModel(t, "")
	m = start(t, m, 160, 60)
	m, _ = step(t, m, key("?"))

	view := m.View()
	for _, want := range []string{"d p", "d P", "pull request"} {
		if !strings.Contains(view, want) {
			t.Errorf("the key list does not name %q:\n%s", want, view)
		}
	}
}

package manager

import (
	"context"
	"errors"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/pullrequest"
)

func TestApplyAndClearPR(t *testing.T) {
	var meta Meta
	applyPR(&meta, pullrequest.PR{
		Provider:   "github",
		Number:     1045,
		URL:        "https://github.com/o/n/pull/1045",
		State:      pullrequest.StateOpen,
		Title:      "t",
		Unresolved: 3,
	}, "feature/pr")
	if meta.PRNumber != 1045 || meta.PRUnresolved != 3 || meta.PRBranch != "feature/pr" {
		t.Fatalf("apply did not mirror: %+v", meta)
	}
	view := prView(meta)
	if !view.Found || view.Provider != "github" || view.Unresolved != 3 {
		t.Fatalf("view = %+v", view)
	}

	clearPR(&meta)
	if meta.PRNumber != 0 || meta.PRState != "" || meta.PRBranch != "" {
		t.Fatalf("clear left data: %+v", meta)
	}
	if prView(meta).Found {
		t.Fatal("a cleared mirror must not read as found")
	}
}

func TestApplyPullRequestMirrorsAndClears(t *testing.T) {
	m := newBridgeManager(t)
	name, err := m.Spawn(context.Background(), Spec{Dir: m.opts.Root, Name: "s"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	m.applyPullRequest(name, "feat", pullrequest.Result{
		Found: true,
		PR:    pullrequest.PR{Provider: "gitlab", Number: 42, State: pullrequest.StateOpen, Unresolved: 2},
	})
	badges := m.PullRequests()
	if b, ok := badges[name]; !ok || b.Number != 42 || b.Unresolved != 2 || b.Provider != "gitlab" {
		t.Fatalf("badge = %+v ok=%v, want number 42", badges[name], ok)
	}

	// A transient error keeps the last known mirror.
	m.applyPullRequest(name, "feat", pullrequest.Result{Err: errors.New("network")})
	if _, ok := m.PullRequests()[name]; !ok {
		t.Fatal("a transient error must keep the mirror")
	}

	// A not-found result clears the mirror.
	m.applyPullRequest(name, "feat", pullrequest.Result{Found: false})
	if _, ok := m.PullRequests()[name]; ok {
		t.Fatal("a not-found result must clear the mirror")
	}
}

func TestConfigurePullRequestRejectsAnUnknownProvider(t *testing.T) {
	m := newBridgeManager(t)
	if _, err := m.ConfigurePullRequest("bitbucket", "t", "", ""); !errors.Is(err, ErrUnknownPRProvider) {
		t.Fatalf("err = %v, want ErrUnknownPRProvider", err)
	}
}

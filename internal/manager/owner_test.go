package manager

import (
	"context"
	"errors"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/mcp"
)

func TestOwnerScopedForward(t *testing.T) {
	m := newTestManager(t)
	view := m.apiSessions("c1", "bruno")

	name, err := view.Create(mcp.CreateInput{Dir: t.TempDir(), Name: "owned"}, "bruno")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := view.SetTitle(name, "Title"); err != nil {
		t.Fatalf("SetTitle: %v", err)
	}
	if _, err := view.Jobs(name); err != nil {
		t.Fatalf("Jobs: %v", err)
	}
	if _, err := view.SendFrom(name, "bruno", "hi"); err != nil {
		t.Fatalf("SendFrom: %v", err)
	}
	if _, err := view.StopJob(name, "no-such-job", "bruno"); err == nil {
		t.Fatal("StopJob on an unknown job did not error")
	}

	other := m.apiSessions("c2", "other")
	if err := other.SetTitle(name, "x"); !errors.Is(err, mcp.ErrNotFound) {
		t.Fatalf("SetTitle on an unowned session: want ErrNotFound, got %v", err)
	}
	if _, err := other.SendFrom(name, "other", "hi"); !errors.Is(err, mcp.ErrNotFound) {
		t.Fatalf("SendFrom to an unowned session: want ErrNotFound, got %v", err)
	}
	if err := other.Archive(name, true, "other"); !errors.Is(err, mcp.ErrNotFound) {
		t.Fatalf("Archive on an unowned session: want ErrNotFound, got %v", err)
	}
	if err := other.Stop(context.Background(), name, "other"); !errors.Is(err, mcp.ErrNotFound) {
		t.Fatalf("Stop on an unowned session: want ErrNotFound, got %v", err)
	}
}

func TestOwnerScopedList(t *testing.T) {
	m := newTestManager(t)
	root := m.opts.Root
	writeStoredMeta(t, root, "a", "c1")
	writeStoredMeta(t, root, "b", "c2")
	writeStoredMeta(t, root, "c", "")

	view := m.apiSessions("c1", "bruno")
	list := view.List()
	if len(list) != 1 || list[0].Name != "a" {
		t.Fatalf("want only session a, got %v", names(list))
	}
}

func TestOwnerScopedGuard(t *testing.T) {
	m := newTestManager(t)
	root := m.opts.Root
	writeStoredMeta(t, root, "mine", "c1")
	writeStoredMeta(t, root, "theirs", "c2")
	writeStoredMeta(t, root, "human", "")

	view := m.apiSessions("c1", "bruno")

	if _, err := view.Messages("theirs", 0); !errors.Is(err, mcp.ErrNotFound) {
		t.Fatalf("another client's session: want ErrNotFound, got %v", err)
	}
	if _, err := view.Messages("human", 0); !errors.Is(err, mcp.ErrNotFound) {
		t.Fatalf("a human's session: want ErrNotFound, got %v", err)
	}
	if _, err := view.Messages("missing", 0); !errors.Is(err, mcp.ErrNotFound) {
		t.Fatalf("an unknown session: want ErrNotFound, got %v", err)
	}
	if _, err := view.Messages("mine", 0); errors.Is(err, mcp.ErrNotFound) {
		t.Fatal("the client's own session is hidden from it")
	}
}

func TestCreateTagsOwner(t *testing.T) {
	m := newTestManager(t)
	view := m.apiSessions("c1", "bruno")

	name, err := view.Create(mcp.CreateInput{Dir: t.TempDir(), Name: "owned"}, "bruno")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	owner, ok := m.ownerOf(name)
	if !ok || owner != "c1" {
		t.Fatalf("want owner c1, got %q (found %v)", owner, ok)
	}

	other := m.apiSessions("c2", "other")
	if _, err := other.Messages(name, 0); !errors.Is(err, mcp.ErrNotFound) {
		t.Fatal("another client sees a session it does not own")
	}
	if list := other.List(); len(list) != 0 {
		t.Fatalf("another client lists a session it does not own: %v", names(list))
	}
}

func TestCredentialTokensRevokedOnRotate(t *testing.T) {
	m := newTestManager(t)
	if err := m.StartMCP(); err != nil {
		t.Fatalf("start mcp: %v", err)
	}
	client, _, err := m.CreateAPIClient("bruno")
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	if _, err := m.RotateAPIClient(client.ClientID); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if err := m.RevokeAPIClient(client.ClientID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
}

func writeStoredMeta(t *testing.T, root, name, owner string) {
	t.Helper()
	meta := Meta{Name: name, Dir: t.TempDir(), Owner: owner}
	if err := writeMeta(metaPath(root, name), meta); err != nil {
		t.Fatalf("write meta %s: %v", name, err)
	}
}

func names(list []mcp.Session) []string {
	out := make([]string, len(list))
	for i, s := range list {
		out[i] = s.Name
	}
	return out
}

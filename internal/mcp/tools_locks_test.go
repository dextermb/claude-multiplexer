package mcp_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/mcp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// lockPortFake implements only mcp.LockPort. It embeds mcp.Sessions, so it
// satisfies the composite the server takes, but every method outside the lock
// port is nil and panics if a lock test reaches past its port. So the lock tools
// run on a fake of six methods, not the whole session surface.
type lockPortFake struct {
	mcp.Sessions
	locks map[string][]string
	list  []mcp.Session
}

func newLockFake() *lockPortFake {
	return &lockPortFake{locks: map[string][]string{}}
}

func (f *lockPortFake) Locks(session string) ([]string, error) {
	return f.locks[session], nil
}

func (f *lockPortFake) SetLocks(labels []string, by string) ([]string, error) {
	f.locks[by] = slices.Clone(labels)
	return f.locks[by], nil
}

func (f *lockPortFake) AddLock(label, by string) ([]string, error) {
	if !slices.Contains(f.locks[by], label) {
		f.locks[by] = append(f.locks[by], label)
	}
	return f.locks[by], nil
}

func (f *lockPortFake) RemoveLock(label, by string) ([]string, error) {
	var kept []string
	for _, held := range f.locks[by] {
		if held != label {
			kept = append(kept, held)
		}
	}
	f.locks[by] = kept
	return kept, nil
}

func (f *lockPortFake) ClearLocks(by string) (bool, error) {
	if len(f.locks[by]) == 0 {
		return false, nil
	}
	delete(f.locks, by)
	return true, nil
}

func (f *lockPortFake) FindLocked(labels []string, live bool) ([]mcp.Session, error) {
	if len(labels) == 0 {
		return nil, mcp.ErrNoLock
	}
	var out []mcp.Session
	for _, item := range f.list {
		if live && !item.Live {
			continue
		}
		held := item.Locks
		if extra, ok := f.locks[item.Name]; ok {
			held = extra
		}
		if holdsEvery(held, labels) {
			out = append(out, item)
		}
	}
	return out, nil
}

// lockClient serves the lock tools of session "docs" over a fake that fills only
// the lock port.
func lockClient(t *testing.T, fake *lockPortFake) *sdk.ClientSession {
	t.Helper()
	server := startServer(t, fake)
	token, err := server.Register("docs", mcp.DefaultProfile, false)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	return connect(t, server, token)
}

func TestLockToolsTakeAndRelease(t *testing.T) {
	sessions := newLockFake()
	client := lockClient(t, sessions)

	if result := call(t, client, mcp.ToolAddLock, map[string]any{"lock": "deploy"}); result.IsError {
		t.Fatalf("add_lock failed: %s", resultText(result))
	}
	again := call(t, client, mcp.ToolAddLock, map[string]any{"lock": "deploy"})
	if again.IsError {
		t.Fatalf("a repeat add_lock failed: %s", resultText(again))
	}
	if len(sessions.locks["docs"]) != 1 {
		t.Fatalf("locks = %v, want one", sessions.locks["docs"])
	}

	if result := call(t, client, mcp.ToolAddLock, map[string]any{"lock": "repo:cmux"}); result.IsError {
		t.Fatalf("add_lock failed: %s", resultText(result))
	}
	list := call(t, client, mcp.ToolListLocks, map[string]any{})
	if !strings.Contains(resultText(list), "holds 2 locks") {
		t.Fatalf("list_locks does not report the count:\n%s", resultText(list))
	}

	if result := call(t, client, mcp.ToolRemoveLock, map[string]any{"lock": "deploy"}); result.IsError {
		t.Fatalf("remove_lock failed: %s", resultText(result))
	}
	if len(sessions.locks["docs"]) != 1 || sessions.locks["docs"][0] != "repo:cmux" {
		t.Fatalf("locks = %v, want [repo:cmux]", sessions.locks["docs"])
	}

	if result := call(t, client, mcp.ToolClearLocks, map[string]any{}); result.IsError {
		t.Fatalf("clear_locks failed: %s", resultText(result))
	}
	empty := call(t, client, mcp.ToolClearLocks, map[string]any{})
	if !strings.Contains(resultText(empty), "held no lock") {
		t.Fatalf("a second clear must say there was none:\n%s", resultText(empty))
	}
}

func TestSetLocksToolReplacesTheSet(t *testing.T) {
	sessions := newLockFake()
	client := lockClient(t, sessions)

	result := call(t, client, mcp.ToolSetLocks, map[string]any{"locks": []any{"one", "two"}})
	if result.IsError {
		t.Fatalf("set_locks failed: %s", resultText(result))
	}
	if len(sessions.locks["docs"]) != 2 || sessions.locks["docs"][0] != "one" {
		t.Fatalf("locks = %v, want [one two]", sessions.locks["docs"])
	}
}

func TestAddLockToolNeedsALabel(t *testing.T) {
	sessions := newLockFake()
	client := lockClient(t, sessions)

	if result := call(t, client, mcp.ToolAddLock, map[string]any{"lock": "  "}); !result.IsError {
		t.Fatal("an empty label must be an error")
	}
	if len(sessions.locks["docs"]) != 0 {
		t.Fatalf("locks = %v, want none", sessions.locks["docs"])
	}
}

func TestAddLockToolNamesTheOtherHolders(t *testing.T) {
	sessions := newFakeSessions()
	sessions.list = []mcp.Session{
		{Name: "api", Live: true, Locks: []string{"deploy"}},
		{Name: "docs", Live: true},
	}
	client := workingDirClient(t, sessions)

	result := call(t, client, mcp.ToolAddLock, map[string]any{"lock": "deploy"})
	if result.IsError {
		t.Fatalf("add_lock failed: %s", resultText(result))
	}
	text := resultText(result)
	if !strings.Contains(text, "api") {
		t.Fatalf("add_lock does not name the other holder:\n%s", text)
	}
	if strings.Contains(text, "no other session") {
		t.Fatalf("add_lock says nobody else holds it, but api does:\n%s", text)
	}
}

func TestFindLockedSessionsToolMatchesEveryLabel(t *testing.T) {
	sessions := newFakeSessions()
	sessions.list = []mcp.Session{
		{Name: "api", Live: true, Locks: []string{"deploy", "repo:cmux"}},
		{Name: "docs", Live: true, Locks: []string{"deploy"}},
		{Name: "old", Locks: []string{"deploy"}},
	}
	client := workingDirClient(t, sessions)

	one := resultText(call(t, client, mcp.ToolFindLocked, map[string]any{"locks": []any{"deploy"}}))
	for _, name := range []string{"api", "docs", "old"} {
		if !strings.Contains(one, name) {
			t.Fatalf("find_locked_sessions does not name %q:\n%s", name, one)
		}
	}

	both := resultText(call(t, client, mcp.ToolFindLocked, map[string]any{"locks": []any{"deploy", "repo:cmux"}}))
	if !strings.Contains(both, "api") || strings.Contains(both, "docs") {
		t.Fatalf("find_locked_sessions must return only the session holding both:\n%s", both)
	}

	live := resultText(call(t, client, mcp.ToolFindLocked,
		map[string]any{"locks": []any{"deploy"}, "live": true}))
	if strings.Contains(live, "old") {
		t.Fatalf("live true must drop the stopped session:\n%s", live)
	}

	if result := call(t, client, mcp.ToolFindLocked, map[string]any{"locks": []any{}}); !result.IsError {
		t.Fatal("a search with no label must be an error")
	}
}

func TestEverySessionGetsTheLockTools(t *testing.T) {
	client := lockClient(t, newLockFake())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tools, err := client.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}

	want := map[string]bool{
		mcp.ToolListLocks: false, mcp.ToolAddLock: false, mcp.ToolRemoveLock: false,
		mcp.ToolSetLocks: false, mcp.ToolClearLocks: false, mcp.ToolFindLocked: false,
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

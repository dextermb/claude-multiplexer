package mcp

import (
	"context"
	"strings"
	"testing"
)

// fakeControlPort fills only the methods the control handlers call. It embeds
// ControlPort, so the rest of the port is nil and a handler that reaches past
// what it needs panics.
type fakeControlPort struct {
	ControlPort
	queued  int
	created string
}

func (f *fakeControlPort) SendFrom(target, from, text string) (int, error) { return f.queued, nil }
func (f *fakeControlPort) Stop(context.Context, string, string) error      { return nil }
func (f *fakeControlPort) Archive(string, bool, string) error              { return nil }
func (f *fakeControlPort) Create(CreateInput, string) (string, error)      { return f.created, nil }
func (f *fakeControlPort) StopJob(string, string, string) (int, error)     { return f.queued, nil }

func TestSendHandlerQueuesAndRefusesSelf(t *testing.T) {
	ctrl := &fakeControlPort{queued: 2}

	out, err := send(ctrl, "docs", sendIn{Session: "api", Text: "hello"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if !out.OK || out.Queued != 2 || !strings.Contains(out.Message, "queued for api") {
		t.Fatalf("out = %+v", out)
	}

	if _, err := send(ctrl, "docs", sendIn{Session: "docs", Text: "hi"}); err != ErrSelfSend {
		t.Fatalf("a self send gave %v, want ErrSelfSend", err)
	}
}

func TestStopHandlerRefusesSelf(t *testing.T) {
	ctrl := &fakeControlPort{}
	if _, err := stop(context.Background(), ctrl, "docs", targetIn{Session: "docs"}); err != ErrSelfStop {
		t.Fatalf("a self stop gave %v, want ErrSelfStop", err)
	}

	out, err := stop(context.Background(), ctrl, "docs", targetIn{Session: "api"})
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
	if !out.OK || !strings.Contains(out.Message, "api is stopped") {
		t.Fatalf("out = %+v", out)
	}
}

func TestArchiveHandlerMessages(t *testing.T) {
	ctrl := &fakeControlPort{}

	off, err := archive(ctrl, "docs", archiveIn{Session: "api"})
	if err != nil || !strings.Contains(off.Message, "archived") {
		t.Fatalf("archive off = %+v (err %v)", off, err)
	}

	on, err := archive(ctrl, "docs", archiveIn{Session: "api", Restore: true})
	if err != nil || !strings.Contains(on.Message, "back in the list") {
		t.Fatalf("archive restore = %+v (err %v)", on, err)
	}
}

func TestCreateHandlerNeedsAPath(t *testing.T) {
	ctrl := &fakeControlPort{created: "api-2"}

	if _, err := create(ctrl, "docs", createIn{}); err != ErrNoPath {
		t.Fatalf("an empty path gave %v, want ErrNoPath", err)
	}

	out, err := create(ctrl, "docs", createIn{Path: "/repo"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if out.Name != "api-2" || !strings.Contains(out.Message, "started api-2") {
		t.Fatalf("out = %+v", out)
	}
}

func TestStopJobHandlerNeedsAJob(t *testing.T) {
	ctrl := &fakeControlPort{queued: 1}

	if _, err := stopJob(ctrl, "docs", stopJobIn{Job: "  "}); err != ErrNoJob {
		t.Fatalf("an empty job gave %v, want ErrNoJob", err)
	}

	out, err := stopJob(ctrl, "docs", stopJobIn{Job: "job-1"})
	if err != nil {
		t.Fatalf("stopJob: %v", err)
	}
	if !out.OK || !strings.Contains(out.Message, "stopping job job-1 of docs") {
		t.Fatalf("out = %+v", out)
	}
}

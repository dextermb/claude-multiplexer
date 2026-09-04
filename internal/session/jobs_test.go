package session

import (
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/protocol"
)

func taskEvent(subtype string, task *protocol.Task) protocol.Event {
	return protocol.Event{Type: protocol.TypeSystem, Subtype: subtype, Task: task}
}

func TestJobStartedThenDone(t *testing.T) {
	s := &Session{}
	s.apply(taskEvent(protocol.SubtypeTaskStarted, &protocol.Task{
		TaskID: "j1", Description: "build", TaskType: "local_bash",
	}))

	jobs := s.jobList()
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].Status != JobRunning || jobs[0].Description != "build" {
		t.Fatalf("unexpected started job: %+v", jobs[0])
	}

	s.apply(taskEvent(protocol.SubtypeTaskUpdated, &protocol.Task{
		TaskID: "j1", Patch: &protocol.TaskPatch{Status: "completed", EndTime: 1788506380063},
	}))

	jobs = s.jobList()
	if jobs[0].Status != JobDone {
		t.Fatalf("status = %v, want done", jobs[0].Status)
	}
	if jobs[0].EndedAt.IsZero() {
		t.Fatal("a terminal job must carry an end time")
	}
}

func TestJobKilledByNotification(t *testing.T) {
	s := &Session{}
	s.apply(taskEvent(protocol.SubtypeTaskStarted, &protocol.Task{TaskID: "j1", Description: "sleep"}))
	s.apply(taskEvent(protocol.SubtypeTaskNotification, &protocol.Task{TaskID: "j1", Status: "stopped"}))

	if got := s.jobList()[0].Status; got != JobKilled {
		t.Fatalf("status = %v, want killed", got)
	}
}

func TestJobUnknownStatusStaysRunning(t *testing.T) {
	s := &Session{}
	s.apply(taskEvent(protocol.SubtypeTaskStarted, &protocol.Task{TaskID: "j1"}))
	s.apply(taskEvent(protocol.SubtypeTaskUpdated, &protocol.Task{
		TaskID: "j1", Patch: &protocol.TaskPatch{Status: "something_new"},
	}))

	if got := s.jobList()[0].Status; got != JobRunning {
		t.Fatalf("status = %v, want running", got)
	}
}

func TestJobNotificationDoesNotOverrideTerminal(t *testing.T) {
	s := &Session{}
	s.apply(taskEvent(protocol.SubtypeTaskStarted, &protocol.Task{TaskID: "j1"}))
	s.apply(taskEvent(protocol.SubtypeTaskUpdated, &protocol.Task{
		TaskID: "j1", Patch: &protocol.TaskPatch{Status: "completed"},
	}))
	s.apply(taskEvent(protocol.SubtypeTaskNotification, &protocol.Task{TaskID: "j1", Status: "stopped"}))

	if got := s.jobList()[0].Status; got != JobDone {
		t.Fatalf("status = %v, want done", got)
	}
}

func TestJobUpdateForUnknownIDIsIgnored(t *testing.T) {
	s := &Session{}
	s.apply(taskEvent(protocol.SubtypeTaskUpdated, &protocol.Task{
		TaskID: "ghost", Patch: &protocol.TaskPatch{Status: "completed"},
	}))
	if got := len(s.jobList()); got != 0 {
		t.Fatalf("an update for an unknown job must add nothing, got %d jobs", got)
	}
}

func TestJobsKeepStartOrder(t *testing.T) {
	s := &Session{}
	s.apply(taskEvent(protocol.SubtypeTaskStarted, &protocol.Task{TaskID: "a"}))
	s.apply(taskEvent(protocol.SubtypeTaskStarted, &protocol.Task{TaskID: "b"}))
	s.apply(taskEvent(protocol.SubtypeTaskStarted, &protocol.Task{TaskID: "a"}))

	jobs := s.jobList()
	if len(jobs) != 2 || jobs[0].ID != "a" || jobs[1].ID != "b" {
		t.Fatalf("unexpected order or count: %+v", jobs)
	}
}

func TestRunningJobsCount(t *testing.T) {
	snap := Snapshot{Jobs: []Job{
		{ID: "a", Status: JobRunning},
		{ID: "b", Status: JobDone},
		{ID: "c", Status: JobRunning},
	}}
	if got := snap.RunningJobs(); got != 2 {
		t.Fatalf("RunningJobs = %d, want 2", got)
	}
}

func TestClassifyStatus(t *testing.T) {
	cases := map[string]struct {
		want     JobStatus
		terminal bool
	}{
		"completed": {JobDone, true},
		"failed":    {JobFailed, true},
		"killed":    {JobKilled, true},
		"stopped":   {JobKilled, true},
		"running":   {JobRunning, false},
		"weird":     {JobRunning, false},
	}
	for status, want := range cases {
		got, terminal := classifyStatus(status)
		if got != want.want || terminal != want.terminal {
			t.Fatalf("classifyStatus(%q) = %v,%v want %v,%v", status, got, terminal, want.want, want.terminal)
		}
	}
}

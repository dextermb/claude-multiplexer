package manager

import (
	"errors"
	"os"
	"testing"
	"time"
)

func backdate(t *testing.T, m *Manager, name string, d time.Duration) {
	t.Helper()
	m.schedMu.Lock()
	defer m.schedMu.Unlock()
	s, ok := m.schedules[name]
	if !ok {
		t.Fatalf("schedule %q not found", name)
	}
	s.CreatedAt = time.Now().Add(-d)
	s.LastRun = time.Time{}
}

func createSchedule(t *testing.T, m *Manager, spec ScheduleSpec) Schedule {
	t.Helper()
	if spec.Cron == "" {
		spec.Cron = "* * * * *"
	}
	if spec.Dir == "" {
		spec.Dir = t.TempDir()
	}
	if spec.Prompt == "" {
		spec.Prompt = "poll the site"
	}
	s, err := m.CreateSchedule(spec)
	if err != nil {
		t.Fatalf("CreateSchedule: %v", err)
	}
	return s
}

func TestScheduleRecordRoundTrips(t *testing.T) {
	root := t.TempDir()
	want := Schedule{Name: "poll", Cron: "*/5 * * * *", Dir: "/tmp", Prompt: "check", Enabled: true, CreatedAt: time.Now().Round(time.Second)}
	if err := writeSchedule(schedulePath(root, want.Name), want); err != nil {
		t.Fatalf("writeSchedule: %v", err)
	}
	got, err := ReadSchedule(schedulePath(root, want.Name))
	if err != nil {
		t.Fatalf("ReadSchedule: %v", err)
	}
	if got.Name != want.Name || got.Cron != want.Cron || got.Prompt != want.Prompt || !got.Enabled {
		t.Fatalf("read back %+v, want %+v", got, want)
	}
	list, err := scanSchedules(root)
	if err != nil {
		t.Fatalf("scanSchedules: %v", err)
	}
	if len(list) != 1 || list[0].Name != want.Name {
		t.Fatalf("scan gave %+v, want one %q", list, want.Name)
	}
}

func TestScanSchedulesEmptyWhenNoDir(t *testing.T) {
	list, err := scanSchedules(t.TempDir())
	if err != nil {
		t.Fatalf("scanSchedules: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("scan gave %d, want 0", len(list))
	}
}

func TestCreateScheduleRejectsBadInput(t *testing.T) {
	m := newTestManager(t)
	dir := t.TempDir()
	cases := []struct {
		name string
		spec ScheduleSpec
		want error
	}{
		{"no cron", ScheduleSpec{Dir: dir, Prompt: "x"}, ErrNoCron},
		{"bad cron", ScheduleSpec{Cron: "not a cron", Dir: dir, Prompt: "x"}, ErrBadCron},
		{"no dir", ScheduleSpec{Cron: "* * * * *", Prompt: "x"}, ErrNoDirectory},
		{"missing dir", ScheduleSpec{Cron: "* * * * *", Dir: dir + "/nope", Prompt: "x"}, ErrNotDirectory},
		{"no prompt", ScheduleSpec{Cron: "* * * * *", Dir: dir}, ErrNoPrompt},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := m.CreateSchedule(tc.spec); !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestCreateSchedulePersistsAndNamesUniquely(t *testing.T) {
	m := newTestManager(t)
	dir := t.TempDir()
	first := createSchedule(t, m, ScheduleSpec{Name: "poll", Dir: dir})
	if first.Name != "poll" {
		t.Fatalf("name = %q, want poll", first.Name)
	}
	if _, err := os.Stat(schedulePath(m.Root(), "poll")); err != nil {
		t.Fatalf("schedule file: %v", err)
	}
	second := createSchedule(t, m, ScheduleSpec{Name: "poll", Dir: dir})
	if second.Name != "poll-2" {
		t.Fatalf("name = %q, want poll-2", second.Name)
	}
}

func TestLoadSchedulesReadsDiskOnNew(t *testing.T) {
	root := t.TempDir()
	writeSchedule(schedulePath(root, "poll"), Schedule{Name: "poll", Cron: "* * * * *", Dir: "/tmp", Prompt: "x", Enabled: true, CreatedAt: time.Now()})
	m, err := New(Options{Root: root, ClaudePath: fakeClaude})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	list := m.ListSchedules()
	if len(list) != 1 || list[0].Name != "poll" {
		t.Fatalf("loaded %+v, want one poll", list)
	}
}

func TestDueSchedulesFiresOneMissedSlotThenNone(t *testing.T) {
	m := newTestManager(t)
	s := createSchedule(t, m, ScheduleSpec{Name: "poll"})
	backdate(t, m, s.Name, 2*time.Minute)

	now := time.Now()
	if due := m.dueSchedules(now); len(due) != 1 {
		t.Fatalf("due = %d, want 1", len(due))
	}
	m.markRun(s.Name, now, "")
	if due := m.dueSchedules(now); len(due) != 0 {
		t.Fatalf("due after run = %d, want 0", len(due))
	}
}

func TestDueSchedulesSkipsDisabled(t *testing.T) {
	m := newTestManager(t)
	s := createSchedule(t, m, ScheduleSpec{Name: "poll"})
	backdate(t, m, s.Name, 2*time.Minute)
	if _, err := m.SetScheduleEnabled(s.Name, false); err != nil {
		t.Fatalf("SetScheduleEnabled: %v", err)
	}
	backdate(t, m, s.Name, 2*time.Minute)
	if due := m.dueSchedules(time.Now()); len(due) != 0 {
		t.Fatalf("due = %d, want 0 for a paused schedule", len(due))
	}
}

func TestRunDueSpawnsAndRecordsTheRun(t *testing.T) {
	m := newTestManager(t)
	s := createSchedule(t, m, ScheduleSpec{Name: "poll"})
	backdate(t, m, s.Name, 2*time.Minute)

	m.runDue(time.Now())

	if got := len(m.Names()); got != 1 {
		t.Fatalf("live sessions = %d, want 1", got)
	}
	list := m.ListSchedules()
	if list[0].LastRun.IsZero() || list[0].LastSession == "" {
		t.Fatalf("run not recorded: %+v", list[0])
	}
}

func TestRunDueSkipsOverlapInSpawnMode(t *testing.T) {
	m := newTestManager(t)
	s := createSchedule(t, m, ScheduleSpec{Name: "poll"})
	backdate(t, m, s.Name, 2*time.Minute)

	m.runDue(time.Now())
	if got := len(m.Names()); got != 1 {
		t.Fatalf("after first run, live = %d, want 1", got)
	}
	backdate(t, m, s.Name, 2*time.Minute)
	m.runDue(time.Now())
	if got := len(m.Names()); got != 1 {
		t.Fatalf("after overlap, live = %d, want 1 (skipped)", got)
	}
}

func TestRunDueCapsSpawnsPerTick(t *testing.T) {
	m := newTestManager(t)
	dir := t.TempDir()
	for range maxSpawnPerTick + 2 {
		s := createSchedule(t, m, ScheduleSpec{Dir: dir})
		backdate(t, m, s.Name, 2*time.Minute)
	}
	m.runDue(time.Now())
	if got := len(m.Names()); got != maxSpawnPerTick {
		t.Fatalf("live sessions = %d, want the cap of %d", got, maxSpawnPerTick)
	}
}

func TestRunScheduleFiresNow(t *testing.T) {
	m := newTestManager(t)
	s := createSchedule(t, m, ScheduleSpec{Name: "poll", Cron: "0 0 1 1 *"})
	session, err := m.RunSchedule(s.Name)
	if err != nil {
		t.Fatalf("RunSchedule: %v", err)
	}
	if session == "" {
		t.Fatal("RunSchedule returned no session")
	}
	if got := len(m.Names()); got != 1 {
		t.Fatalf("live sessions = %d, want 1", got)
	}
}

func TestDeleteScheduleRemovesFileAndMemory(t *testing.T) {
	m := newTestManager(t)
	s := createSchedule(t, m, ScheduleSpec{Name: "poll"})
	if err := m.DeleteSchedule(s.Name); err != nil {
		t.Fatalf("DeleteSchedule: %v", err)
	}
	if _, err := os.Stat(schedulePath(m.Root(), s.Name)); !os.IsNotExist(err) {
		t.Fatalf("file still there: %v", err)
	}
	if len(m.ListSchedules()) != 0 {
		t.Fatalf("schedule still in memory")
	}
	if err := m.DeleteSchedule("gone"); !errors.Is(err, ErrUnknownSchedule) {
		t.Fatalf("delete unknown err = %v, want ErrUnknownSchedule", err)
	}
}

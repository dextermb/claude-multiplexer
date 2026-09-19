package manager

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/dextermb/claude-multiplexer/internal/config"
)

// Schedule is one task the manager runs on its own clock. A recurring schedule
// holds a cron; a one-off schedule holds an empty cron and a RunAfter time. It
// holds only provider-neutral fields, so it does not depend on any one session
// provider. See docs/scheduler.md.
type Schedule struct {
	Name           string    `json:"name"`
	Cron           string    `json:"cron"`
	RunAfter       time.Time `json:"run_after,omitempty"`
	Dir            string    `json:"dir"`
	Prompt         string    `json:"prompt"`
	Session        string    `json:"session,omitempty"`
	Model          string    `json:"model,omitempty"`
	PermissionMode string    `json:"permission_mode,omitempty"`
	Effort         string    `json:"effort,omitempty"`
	Control        bool      `json:"control,omitempty"`
	Enabled        bool      `json:"enabled"`
	CreatedAt      time.Time `json:"created_at"`
	LastRun        time.Time `json:"last_run,omitempty"`
	LastSession    string    `json:"last_session,omitempty"`
}

// ScheduleSpec is the input to CreateSchedule. The manager fills the rest of the
// Schedule (the name, the created time, the enabled flag). RunAfter is the raw
// run-after time: a Go duration or an RFC3339 timestamp.
type ScheduleSpec struct {
	Name           string
	Cron           string
	RunAfter       string
	Dir            string
	Prompt         string
	Session        string
	Model          string
	PermissionMode string
	Effort         string
	Control        bool
}

// parseRunAfter reads the raw run-after time of a one-off schedule. It tries a Go
// duration first, so "30m" resolves to now plus the duration; else it tries an
// RFC3339 timestamp. See docs/scheduler.md.
func parseRunAfter(raw string, now time.Time) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if d, err := time.ParseDuration(raw); err == nil {
		return now.Add(d), nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("%w: %s", ErrBadRunAfter, raw)
}

func scheduleDir(root string) string {
	return filepath.Join(root, "schedules")
}

func schedulePath(root, name string) string {
	return filepath.Join(scheduleDir(root), name+".json")
}

func writeSchedule(path string, s Schedule) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// ReadSchedule reads one schedule record from disk.
func ReadSchedule(path string) (Schedule, error) {
	var s Schedule
	data, err := os.ReadFile(path)
	if err != nil {
		return s, err
	}
	err = json.Unmarshal(data, &s)
	return s, err
}

func scanSchedules(root string) ([]Schedule, error) {
	entries, err := os.ReadDir(scheduleDir(root))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Schedule
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		s, err := ReadSchedule(schedulePath(root, strings.TrimSuffix(entry.Name(), ".json")))
		if err != nil {
			continue
		}
		if s.Name == "" {
			s.Name = strings.TrimSuffix(entry.Name(), ".json")
		}
		out = append(out, s)
	}
	return out, nil
}

func (m *Manager) loadSchedules() {
	m.schedMu.Lock()
	defer m.schedMu.Unlock()
	if m.schedules == nil {
		m.schedules = map[string]*Schedule{}
	}
	list, err := scanSchedules(m.opts.Root)
	if err != nil {
		return
	}
	for _, s := range list {
		record := s
		m.schedules[record.Name] = &record
	}
}

// defaultScheduleModel is the model a new schedule takes when it names none. It
// reads the setting on each call, so a change needs no restart. See
// docs/scheduler.md.
func (m *Manager) defaultScheduleModel() string {
	cfg, err := config.Load(m.opts.ConfigPaths...)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cfg.DefaultScheduleModel)
}

// CreateSchedule validates the cron, the directory, and the prompt, then writes a
// new schedule to disk and holds it in memory. See docs/scheduler.md.
func (m *Manager) CreateSchedule(spec ScheduleSpec) (Schedule, error) {
	cronExpr := strings.TrimSpace(spec.Cron)
	rawAfter := strings.TrimSpace(spec.RunAfter)
	switch {
	case cronExpr == "" && rawAfter == "":
		return Schedule{}, ErrNoCron
	case cronExpr != "" && rawAfter != "":
		return Schedule{}, ErrCronAndRunAfter
	case cronExpr != "":
		if _, err := cron.ParseStandard(cronExpr); err != nil {
			return Schedule{}, fmt.Errorf("%w: %v", ErrBadCron, err)
		}
	}
	if spec.Dir == "" {
		return Schedule{}, ErrNoDirectory
	}
	dir, err := filepath.Abs(spec.Dir)
	if err != nil {
		return Schedule{}, err
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return Schedule{}, fmt.Errorf("%w: %s", ErrNotDirectory, spec.Dir)
	}
	if strings.TrimSpace(spec.Prompt) == "" {
		return Schedule{}, ErrNoPrompt
	}
	model := strings.TrimSpace(spec.Model)
	if model == "" {
		model = m.defaultScheduleModel()
	}
	now := time.Now()
	var runAfter time.Time
	if rawAfter != "" {
		runAfter, err = parseRunAfter(rawAfter, now)
		if err != nil {
			return Schedule{}, err
		}
	}

	m.schedMu.Lock()
	name := m.uniqueScheduleName(spec.Name, dir)
	record := &Schedule{
		Name:           name,
		Cron:           cronExpr,
		RunAfter:       runAfter,
		Dir:            dir,
		Prompt:         spec.Prompt,
		Session:        strings.TrimSpace(spec.Session),
		Model:          model,
		PermissionMode: spec.PermissionMode,
		Effort:         spec.Effort,
		Control:        spec.Control,
		Enabled:        true,
		CreatedAt:      now,
	}
	m.schedules[name] = record
	snapshot := *record
	m.schedMu.Unlock()

	if err := writeSchedule(schedulePath(m.opts.Root, name), snapshot); err != nil {
		m.schedMu.Lock()
		delete(m.schedules, name)
		m.schedMu.Unlock()
		return Schedule{}, err
	}
	return snapshot, nil
}

// ScheduleUpdate is the input to UpdateSchedule. A nil field stays as it is; a
// non-nil field takes its new value, so an empty string clears an optional field.
// A non-empty Cron switches the schedule to recurring; a non-empty RunAfter (a
// duration or an RFC3339 time) switches it to one-off.
type ScheduleUpdate struct {
	Cron           *string
	RunAfter       *string
	Dir            *string
	Prompt         *string
	Session        *string
	Model          *string
	PermissionMode *string
	Effort         *string
	Control        *bool
}

// UpdateSchedule changes the fields a ScheduleUpdate names, and leaves the rest.
// It validates a new cron, a new run-after time, a new directory, and a new
// prompt the same way CreateSchedule does. The cron and the run-after time are
// mutually exclusive, so setting one clears the other. See docs/scheduler.md.
func (m *Manager) UpdateSchedule(name string, up ScheduleUpdate) (Schedule, error) {
	newCron, setCron := "", false
	if up.Cron != nil {
		newCron = strings.TrimSpace(*up.Cron)
		setCron = true
		if newCron != "" {
			if _, err := cron.ParseStandard(newCron); err != nil {
				return Schedule{}, fmt.Errorf("%w: %v", ErrBadCron, err)
			}
		}
	}
	var newRunAfter time.Time
	setRunAfter, clearRunAfter := false, false
	if up.RunAfter != nil {
		if strings.TrimSpace(*up.RunAfter) == "" {
			clearRunAfter = true
		} else {
			t, err := parseRunAfter(*up.RunAfter, time.Now())
			if err != nil {
				return Schedule{}, err
			}
			newRunAfter = t
			setRunAfter = true
		}
	}
	if newCron != "" && setRunAfter {
		return Schedule{}, ErrCronAndRunAfter
	}
	var dir string
	if up.Dir != nil {
		if *up.Dir == "" {
			return Schedule{}, ErrNoDirectory
		}
		abs, err := filepath.Abs(*up.Dir)
		if err != nil {
			return Schedule{}, err
		}
		info, err := os.Stat(abs)
		if err != nil || !info.IsDir() {
			return Schedule{}, fmt.Errorf("%w: %s", ErrNotDirectory, *up.Dir)
		}
		dir = abs
	}
	if up.Prompt != nil && strings.TrimSpace(*up.Prompt) == "" {
		return Schedule{}, ErrNoPrompt
	}

	m.schedMu.Lock()
	s, ok := m.schedules[name]
	if !ok {
		m.schedMu.Unlock()
		return Schedule{}, fmt.Errorf("%w: %s", ErrUnknownSchedule, name)
	}
	resultCron, resultAfter, resultLastRun := s.Cron, s.RunAfter, s.LastRun
	if setCron {
		resultCron = newCron
		if newCron != "" {
			resultAfter = time.Time{}
		}
	}
	if setRunAfter {
		resultAfter = newRunAfter
		resultCron = ""
		resultLastRun = time.Time{}
	}
	if clearRunAfter {
		resultAfter = time.Time{}
	}
	if resultCron == "" && resultAfter.IsZero() {
		m.schedMu.Unlock()
		return Schedule{}, ErrNoCron
	}
	s.Cron = resultCron
	s.RunAfter = resultAfter
	s.LastRun = resultLastRun
	if up.Dir != nil {
		s.Dir = dir
	}
	if up.Prompt != nil {
		s.Prompt = *up.Prompt
	}
	if up.Session != nil {
		s.Session = strings.TrimSpace(*up.Session)
	}
	if up.Model != nil {
		s.Model = *up.Model
	}
	if up.PermissionMode != nil {
		s.PermissionMode = *up.PermissionMode
	}
	if up.Effort != nil {
		s.Effort = *up.Effort
	}
	if up.Control != nil {
		s.Control = *up.Control
	}
	snapshot := *s
	m.schedMu.Unlock()

	if err := writeSchedule(schedulePath(m.opts.Root, name), snapshot); err != nil {
		return Schedule{}, err
	}
	return snapshot, nil
}

// uniqueScheduleName gives a schedule a name no other schedule holds. Call it
// with the schedule mutex held.
func (m *Manager) uniqueScheduleName(want, dir string) string {
	base := sanitiseScheduleName(want)
	if base == "" {
		base = sanitiseScheduleName(filepath.Base(dir))
	}
	if base == "" {
		base = "schedule"
	}
	name := base
	for i := 2; ; i++ {
		if _, exists := m.schedules[name]; !exists {
			return name
		}
		name = fmt.Sprintf("%s-%d", base, i)
	}
}

func sanitiseScheduleName(want string) string {
	want = strings.TrimSpace(want)
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "-", ".", "-")
	return strings.Trim(replacer.Replace(want), "-")
}

// DeleteSchedule removes a schedule from memory and from disk. A session it
// started is left alone.
func (m *Manager) DeleteSchedule(name string) error {
	m.schedMu.Lock()
	_, ok := m.schedules[name]
	if ok {
		delete(m.schedules, name)
	}
	m.schedMu.Unlock()
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownSchedule, name)
	}
	if err := os.Remove(schedulePath(m.opts.Root, name)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ListSchedules gives a copy of every schedule, in name order.
func (m *Manager) ListSchedules() []Schedule {
	m.schedMu.Lock()
	defer m.schedMu.Unlock()
	out := make([]Schedule, 0, len(m.schedules))
	for _, s := range m.schedules {
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// SetScheduleEnabled turns a schedule on or off. A disabled schedule stays on
// disk, so a later enable needs no re-entry.
func (m *Manager) SetScheduleEnabled(name string, on bool) (Schedule, error) {
	m.schedMu.Lock()
	s, ok := m.schedules[name]
	if !ok {
		m.schedMu.Unlock()
		return Schedule{}, fmt.Errorf("%w: %s", ErrUnknownSchedule, name)
	}
	s.Enabled = on
	snapshot := *s
	m.schedMu.Unlock()

	if err := writeSchedule(schedulePath(m.opts.Root, name), snapshot); err != nil {
		return Schedule{}, err
	}
	return snapshot, nil
}

// RunSchedule fires one schedule now, whatever its cron says. It is for a test of
// a schedule. It returns the name of the session it ran.
func (m *Manager) RunSchedule(name string) (string, error) {
	m.schedMu.Lock()
	s, ok := m.schedules[name]
	if !ok {
		m.schedMu.Unlock()
		return "", fmt.Errorf("%w: %s", ErrUnknownSchedule, name)
	}
	snapshot := *s
	m.schedMu.Unlock()

	session, err := m.fireSchedule(snapshot)
	if err != nil {
		return "", err
	}
	m.markRun(name, time.Now(), session)
	return session, nil
}

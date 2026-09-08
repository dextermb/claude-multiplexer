package manager

import (
	"time"

	"github.com/dextermb/claude-multiplexer/internal/mcp"
)

func (b *bridge) CreateSchedule(in mcp.ScheduleInput, by string) (mcp.Schedule, error) {
	sched, err := b.m.CreateSchedule(ScheduleSpec{
		Name:           in.Name,
		Cron:           in.Cron,
		Dir:            in.Dir,
		Prompt:         in.Prompt,
		Session:        in.Session,
		Model:          in.Model,
		PermissionMode: in.PermissionMode,
		Effort:         in.Effort,
		Control:        in.Control,
	})
	if err != nil {
		return mcp.Schedule{}, err
	}
	b.m.notify(by, by+" created the schedule "+sched.Name, true)
	return scheduleView(sched), nil
}

func (b *bridge) UpdateSchedule(name string, up mcp.ScheduleEdit, by string) (mcp.Schedule, error) {
	sched, err := b.m.UpdateSchedule(name, ScheduleUpdate{
		Cron:           up.Cron,
		Dir:            up.Dir,
		Prompt:         up.Prompt,
		Session:        up.Session,
		Model:          up.Model,
		PermissionMode: up.PermissionMode,
		Effort:         up.Effort,
		Control:        up.Control,
	})
	if err != nil {
		return mcp.Schedule{}, err
	}
	b.m.notify(by, by+" updated the schedule "+name, true)
	return scheduleView(sched), nil
}

func (b *bridge) ListSchedules() []mcp.Schedule {
	list := b.m.ListSchedules()
	out := make([]mcp.Schedule, 0, len(list))
	for _, sched := range list {
		out = append(out, scheduleView(sched))
	}
	return out
}

func (b *bridge) DeleteSchedule(name, by string) (bool, error) {
	if err := b.m.DeleteSchedule(name); err != nil {
		return false, err
	}
	b.m.notify(by, by+" deleted the schedule "+name, true)
	return true, nil
}

func (b *bridge) SetScheduleEnabled(name string, on bool, by string) (mcp.Schedule, error) {
	sched, err := b.m.SetScheduleEnabled(name, on)
	if err != nil {
		return mcp.Schedule{}, err
	}
	verb := "enabled"
	if !on {
		verb = "paused"
	}
	b.m.notify(by, by+" "+verb+" the schedule "+name, true)
	return scheduleView(sched), nil
}

func (b *bridge) SchedulePath() mcp.SchedulePath { return b.m.SchedulePath() }

func (b *bridge) RunSchedule(name, by string) (string, error) {
	session, err := b.m.RunSchedule(name)
	if err != nil {
		return "", err
	}
	b.m.notify(session, by+" ran the schedule "+name, true)
	return session, nil
}

func scheduleView(s Schedule) mcp.Schedule {
	view := mcp.Schedule{
		Name:           s.Name,
		Cron:           s.Cron,
		Dir:            s.Dir,
		Prompt:         s.Prompt,
		Session:        s.Session,
		Model:          s.Model,
		PermissionMode: s.PermissionMode,
		Effort:         s.Effort,
		Control:        s.Control,
		Enabled:        s.Enabled,
		LastSession:    s.LastSession,
	}
	if !s.LastRun.IsZero() {
		view.LastRun = s.LastRun.Format(time.RFC3339)
	}
	return view
}

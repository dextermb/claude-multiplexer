package mcp

import (
	"context"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) addScheduleTools(server *sdk.Server, caller string, control bool) {
	sdk.AddTool(server, &sdk.Tool{
		Name: ToolCreateSchedule,
		Description: "Create a durable schedule that runs a prompt on a cron. " +
			"The multiplexer runs it on its own clock, and it survives a restart. " +
			"Leave 'session' empty to start a fresh session each run, or give a session name to reuse one session so it keeps its memory. " +
			"Give a 5-field cron in local time, a directory, and a prompt.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in createScheduleIn) (*sdk.CallToolResult, scheduleOut, error) {
		out, err := createSchedule(s.sessions, caller, control, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolUpdateSchedule,
		Description: "Change one or more fields of a schedule, and leave the rest. " +
			"A field left out stays as it is; an empty string clears an optional field, such as 'session' to return to a fresh session each run. " +
			"The name identifies the schedule and does not change.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in updateScheduleIn) (*sdk.CallToolResult, scheduleOut, error) {
		out, err := updateSchedule(s.sessions, caller, control, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolListSchedules,
		Description: "List every schedule, with its cron, its directory, its model, whether it runs a fresh session or reuses one, and when it last ran.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, listSchedulesOut, error) {
		return nil, listSchedulesOut{Schedules: s.sessions.ListSchedules()}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolDeleteSchedule,
		Description: "Remove a schedule. A session it already started is left alone.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in deleteScheduleIn) (*sdk.CallToolResult, deleteScheduleOut, error) {
		out, err := deleteSchedule(s.sessions, caller, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolSetScheduleEnabled,
		Description: "Turn a schedule on or off. A paused schedule stays on disk, so a later enable needs no re-entry.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in setScheduleEnabledIn) (*sdk.CallToolResult, scheduleOut, error) {
		out, err := setScheduleEnabled(s.sessions, caller, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolRunSchedule,
		Description: "Run a schedule now, whatever its cron says. Use it to test a schedule. It returns the name of the session it ran.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in runScheduleIn) (*sdk.CallToolResult, runScheduleOut, error) {
		out, err := runSchedule(s.sessions, caller, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolSchedulePath,
		Description: "The directory the multiplexer writes schedule records to, one JSON file per schedule.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, SchedulePath, error) {
		return nil, s.sessions.SchedulePath(), nil
	})
}

func createSchedule(sched SchedulePort, caller string, control bool, in createScheduleIn) (scheduleOut, error) {
	if strings.TrimSpace(in.Cron) == "" {
		return scheduleOut{}, ErrNoCron
	}
	if strings.TrimSpace(in.Dir) == "" {
		return scheduleOut{}, ErrNoDir
	}
	if strings.TrimSpace(in.Prompt) == "" {
		return scheduleOut{}, ErrNoPrompt
	}
	created, err := sched.CreateSchedule(ScheduleInput{
		Name:           strings.TrimSpace(in.Name),
		Cron:           strings.TrimSpace(in.Cron),
		Dir:            strings.TrimSpace(in.Dir),
		Prompt:         in.Prompt,
		Session:        strings.TrimSpace(in.Session),
		Model:          strings.TrimSpace(in.Model),
		PermissionMode: strings.TrimSpace(in.PermissionMode),
		Effort:         strings.TrimSpace(in.Effort),
		Control:        in.Control && control,
	}, caller)
	if err != nil {
		return scheduleOut{}, err
	}
	return scheduleOut{OK: true, Schedule: created, Message: "created the schedule " + created.Name}, nil
}

func updateSchedule(sched SchedulePort, caller string, control bool, in updateScheduleIn) (scheduleOut, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return scheduleOut{}, ErrNoSchedule
	}
	wantControl := in.Control
	if wantControl != nil && *wantControl && !control {
		off := false
		wantControl = &off
	}
	updated, err := sched.UpdateSchedule(name, ScheduleEdit{
		Cron:           in.Cron,
		Dir:            in.Dir,
		Prompt:         in.Prompt,
		Session:        in.Session,
		Model:          in.Model,
		PermissionMode: in.PermissionMode,
		Effort:         in.Effort,
		Control:        wantControl,
	}, caller)
	if err != nil {
		return scheduleOut{}, err
	}
	return scheduleOut{OK: true, Schedule: updated, Message: "updated the schedule " + name}, nil
}

func deleteSchedule(sched SchedulePort, caller string, in deleteScheduleIn) (deleteScheduleOut, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return deleteScheduleOut{}, ErrNoSchedule
	}
	changed, err := sched.DeleteSchedule(name, caller)
	if err != nil {
		return deleteScheduleOut{}, err
	}
	return deleteScheduleOut{OK: true, Changed: changed, Message: "removed the schedule " + name}, nil
}

func setScheduleEnabled(sched SchedulePort, caller string, in setScheduleEnabledIn) (scheduleOut, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return scheduleOut{}, ErrNoSchedule
	}
	updated, err := sched.SetScheduleEnabled(name, in.Enabled, caller)
	if err != nil {
		return scheduleOut{}, err
	}
	state := "enabled"
	if !in.Enabled {
		state = "paused"
	}
	return scheduleOut{OK: true, Schedule: updated, Message: "the schedule " + name + " is " + state}, nil
}

func runSchedule(sched SchedulePort, caller string, in runScheduleIn) (runScheduleOut, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return runScheduleOut{}, ErrNoSchedule
	}
	ran, err := sched.RunSchedule(name, caller)
	if err != nil {
		return runScheduleOut{}, err
	}
	return runScheduleOut{OK: true, Session: ran, Message: "the schedule " + name + " ran " + ran}, nil
}

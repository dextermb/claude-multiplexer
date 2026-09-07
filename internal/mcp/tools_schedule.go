package mcp

import (
	"context"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) addScheduleTools(server *sdk.Server, caller string) {
	sdk.AddTool(server, &sdk.Tool{
		Name: ToolCreateSchedule,
		Description: "Create a durable schedule that runs a prompt on a cron. " +
			"The multiplexer runs it on its own clock, and it survives a restart. " +
			"Leave 'session' empty to start a fresh session each run, or give a session name to reuse one session so it keeps its memory. " +
			"Give a 5-field cron in local time, a directory, and a prompt.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in createScheduleIn) (*sdk.CallToolResult, scheduleOut, error) {
		if strings.TrimSpace(in.Cron) == "" {
			return nil, scheduleOut{}, ErrNoCron
		}
		if strings.TrimSpace(in.Dir) == "" {
			return nil, scheduleOut{}, ErrNoDir
		}
		if strings.TrimSpace(in.Prompt) == "" {
			return nil, scheduleOut{}, ErrNoPrompt
		}
		sched, err := s.sessions.CreateSchedule(ScheduleInput{
			Name:           strings.TrimSpace(in.Name),
			Cron:           strings.TrimSpace(in.Cron),
			Dir:            strings.TrimSpace(in.Dir),
			Prompt:         in.Prompt,
			Session:        strings.TrimSpace(in.Session),
			Model:          strings.TrimSpace(in.Model),
			PermissionMode: strings.TrimSpace(in.PermissionMode),
			Effort:         strings.TrimSpace(in.Effort),
			Control:        in.Control,
		}, caller)
		if err != nil {
			return nil, scheduleOut{}, err
		}
		return nil, scheduleOut{OK: true, Schedule: sched, Message: "created the schedule " + sched.Name}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolListSchedules,
		Description: "List every schedule, with its cron, its directory, whether it runs a fresh session or reuses one, and when it last ran.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, listSchedulesOut, error) {
		return nil, listSchedulesOut{Schedules: s.sessions.ListSchedules()}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolDeleteSchedule,
		Description: "Remove a schedule. A session it already started is left alone.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in deleteScheduleIn) (*sdk.CallToolResult, deleteScheduleOut, error) {
		name := strings.TrimSpace(in.Name)
		if name == "" {
			return nil, deleteScheduleOut{}, ErrNoSchedule
		}
		changed, err := s.sessions.DeleteSchedule(name, caller)
		if err != nil {
			return nil, deleteScheduleOut{}, err
		}
		return nil, deleteScheduleOut{OK: true, Changed: changed, Message: "removed the schedule " + name}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolSetScheduleEnabled,
		Description: "Turn a schedule on or off. A paused schedule stays on disk, so a later enable needs no re-entry.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in setScheduleEnabledIn) (*sdk.CallToolResult, scheduleOut, error) {
		name := strings.TrimSpace(in.Name)
		if name == "" {
			return nil, scheduleOut{}, ErrNoSchedule
		}
		sched, err := s.sessions.SetScheduleEnabled(name, in.Enabled, caller)
		if err != nil {
			return nil, scheduleOut{}, err
		}
		state := "enabled"
		if !in.Enabled {
			state = "paused"
		}
		return nil, scheduleOut{OK: true, Schedule: sched, Message: "the schedule " + name + " is " + state}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolRunSchedule,
		Description: "Run a schedule now, whatever its cron says. Use it to test a schedule. It returns the name of the session it ran.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in runScheduleIn) (*sdk.CallToolResult, runScheduleOut, error) {
		name := strings.TrimSpace(in.Name)
		if name == "" {
			return nil, runScheduleOut{}, ErrNoSchedule
		}
		session, err := s.sessions.RunSchedule(name, caller)
		if err != nil {
			return nil, runScheduleOut{}, err
		}
		return nil, runScheduleOut{OK: true, Session: session, Message: "the schedule " + name + " ran " + session}, nil
	})
}

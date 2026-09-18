package mcp

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) addControlTools(server *sdk.Server, caller string) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolSend,
		Description: "Queue a prompt for another session. It returns at once, so read the answer later with " + ToolMessages + ".",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in sendIn) (*sdk.CallToolResult, sendOut, error) {
		out, err := send(s.sessions, caller, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolStop,
		Description: "End another session in a clean way. Its transcript is kept.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in targetIn) (*sdk.CallToolResult, okOut, error) {
		out, err := stop(ctx, s.sessions, caller, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolArchive,
		Description: "Take a stopped session out of the list, or with restore, bring it back. A running session cannot be archived.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in archiveIn) (*sdk.CallToolResult, okOut, error) {
		out, err := archive(s.sessions, caller, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolCreate,
		Description: "Start a new session in a directory. Give a path, and an optional name. It returns the name the session takes.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in createIn) (*sdk.CallToolResult, createOut, error) {
		out, err := create(s.sessions, caller, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolStopJob,
		Description: "Stop a running background job. It interrupts the owning session and asks it to run KillShell on that job, so the job stops on the next turn. Give the job id, and a session name or empty for this session.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in stopJobIn) (*sdk.CallToolResult, stopJobOut, error) {
		out, err := stopJob(s.sessions, caller, in)
		return nil, out, err
	})
}

func send(ctrl ControlPort, caller string, in sendIn) (sendOut, error) {
	target, err := cleanTarget(in.Session)
	if err != nil {
		return sendOut{}, err
	}
	if target == caller {
		return sendOut{}, ErrSelfSend
	}
	queued, err := ctrl.SendFrom(target, caller, in.Text)
	if err != nil {
		return sendOut{}, err
	}
	return sendOut{OK: true, Queued: queued, Message: fmt.Sprintf("queued for %s, %d waiting", target, queued)}, nil
}

func stop(ctx context.Context, ctrl ControlPort, caller string, in targetIn) (okOut, error) {
	target, err := cleanTarget(in.Session)
	if err != nil {
		return okOut{}, err
	}
	if target == caller {
		return okOut{}, ErrSelfStop
	}
	if err := ctrl.Stop(ctx, target, caller); err != nil {
		return okOut{}, err
	}
	return okOut{OK: true, Message: target + " is stopped"}, nil
}

func archive(ctrl ControlPort, caller string, in archiveIn) (okOut, error) {
	target, err := cleanTarget(in.Session)
	if err != nil {
		return okOut{}, err
	}
	if err := ctrl.Archive(target, !in.Restore, caller); err != nil {
		return okOut{}, err
	}
	if in.Restore {
		return okOut{OK: true, Message: target + " is back in the list"}, nil
	}
	return okOut{OK: true, Message: target + " is archived"}, nil
}

func create(ctrl ControlPort, caller string, in createIn) (createOut, error) {
	path := strings.TrimSpace(in.Path)
	if path == "" {
		return createOut{}, ErrNoPath
	}
	profile, err := ParseProfile(in.Profile)
	if err != nil {
		return createOut{}, err
	}
	created, err := ctrl.Create(CreateInput{
		Dir:     path,
		Name:    strings.TrimSpace(in.Name),
		Model:   strings.TrimSpace(in.Model),
		Effort:  strings.TrimSpace(in.Effort),
		Profile: string(profile),
	}, caller)
	if err != nil {
		return createOut{}, err
	}
	return createOut{OK: true, Name: created, Message: "started " + created}, nil
}

func stopJob(ctrl ControlPort, caller string, in stopJobIn) (stopJobOut, error) {
	target, err := targetOrSelf(in.Session, caller)
	if err != nil {
		return stopJobOut{}, err
	}
	job := strings.TrimSpace(in.Job)
	if job == "" {
		return stopJobOut{}, ErrNoJob
	}
	queued, err := ctrl.StopJob(target, job, caller)
	if err != nil {
		return stopJobOut{}, err
	}
	return stopJobOut{OK: true, Queued: queued, Message: fmt.Sprintf("stopping job %s of %s", job, target)}, nil
}

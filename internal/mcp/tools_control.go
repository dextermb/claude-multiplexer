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
		target, err := cleanTarget(in.Session)
		if err != nil {
			return nil, sendOut{}, err
		}
		if target == caller {
			return nil, sendOut{}, ErrSelfSend
		}
		queued, err := s.sessions.SendFrom(target, caller, in.Text)
		if err != nil {
			return nil, sendOut{}, err
		}
		return nil, sendOut{OK: true, Queued: queued, Message: fmt.Sprintf("queued for %s, %d waiting", target, queued)}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolStop,
		Description: "End another session in a clean way. Its transcript is kept.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in targetIn) (*sdk.CallToolResult, okOut, error) {
		target, err := cleanTarget(in.Session)
		if err != nil {
			return nil, okOut{}, err
		}
		if target == caller {
			return nil, okOut{}, ErrSelfStop
		}
		if err := s.sessions.Stop(ctx, target, caller); err != nil {
			return nil, okOut{}, err
		}
		return nil, okOut{OK: true, Message: target + " is stopped"}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolArchive,
		Description: "Take a stopped session out of the list, or with restore, bring it back. A running session cannot be archived.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in archiveIn) (*sdk.CallToolResult, okOut, error) {
		target, err := cleanTarget(in.Session)
		if err != nil {
			return nil, okOut{}, err
		}
		if err := s.sessions.Archive(target, !in.Restore, caller); err != nil {
			return nil, okOut{}, err
		}
		if in.Restore {
			return nil, okOut{OK: true, Message: target + " is back in the list"}, nil
		}
		return nil, okOut{OK: true, Message: target + " is archived"}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolCreate,
		Description: "Start a new session in a directory. Give a path, and an optional name. It returns the name the session takes.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in createIn) (*sdk.CallToolResult, createOut, error) {
		path := strings.TrimSpace(in.Path)
		if path == "" {
			return nil, createOut{}, ErrNoPath
		}
		created, err := s.sessions.Create(path, strings.TrimSpace(in.Name), caller)
		if err != nil {
			return nil, createOut{}, err
		}
		return nil, createOut{OK: true, Name: created, Message: "started " + created}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolStopJob,
		Description: "Stop a running background job. It interrupts the owning session and asks it to run KillShell on that job, so the job stops on the next turn. Give the job id, and a session name or empty for this session.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in stopJobIn) (*sdk.CallToolResult, stopJobOut, error) {
		target, err := targetOrSelf(in.Session, caller)
		if err != nil {
			return nil, stopJobOut{}, err
		}
		job := strings.TrimSpace(in.Job)
		if job == "" {
			return nil, stopJobOut{}, ErrNoJob
		}
		queued, err := s.sessions.StopJob(target, job, caller)
		if err != nil {
			return nil, stopJobOut{}, err
		}
		return nil, stopJobOut{OK: true, Queued: queued, Message: fmt.Sprintf("stopping job %s of %s", job, target)}, nil
	})
}

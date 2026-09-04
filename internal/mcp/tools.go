package mcp

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type renameIn struct {
	Title string `json:"title" jsonschema:"the new display title for this session; an empty string clears it"`
}

type listIn struct {
	LiveOnly bool `json:"live_only,omitempty" jsonschema:"true to leave out the sessions that are not running now"`
}

type messagesIn struct {
	Session string `json:"session" jsonschema:"the name of the session to read"`
	Limit   int    `json:"limit,omitempty" jsonschema:"how many messages to return, newest last; 20 by default"`
}

type sendIn struct {
	Session string `json:"session" jsonschema:"the name of the session to prompt"`
	Text    string `json:"text" jsonschema:"the prompt to queue for that session"`
}

type targetIn struct {
	Session string `json:"session" jsonschema:"the name of the session to act on"`
}

type archiveIn struct {
	Session string `json:"session" jsonschema:"the name of the session to act on"`
	Restore bool   `json:"restore,omitempty" jsonschema:"true to bring an archived session back into the list"`
}

type createIn struct {
	Path string `json:"path" jsonschema:"the directory the new session works in"`
	Name string `json:"name,omitempty" jsonschema:"an optional name for the new session"`
}

type listJobsIn struct {
	Session string `json:"session,omitempty" jsonschema:"the session to read the jobs of; empty means this session"`
}

type stopJobIn struct {
	Session string `json:"session,omitempty" jsonschema:"the session that owns the job; empty means this session"`
	Job     string `json:"job" jsonschema:"the id of the background job to stop"`
}

type okOut struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

type sendOut struct {
	OK      bool   `json:"ok"`
	Queued  int    `json:"queued"`
	Message string `json:"message"`
}

type listOut struct {
	Sessions []Session `json:"sessions"`
}

type messagesOut struct {
	Session  string    `json:"session"`
	Messages []Message `json:"messages"`
}

type createOut struct {
	OK      bool   `json:"ok"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

type listJobsOut struct {
	Session string `json:"session"`
	Jobs    []Job  `json:"jobs"`
}

type stopJobOut struct {
	OK      bool   `json:"ok"`
	Queued  int    `json:"queued"`
	Message string `json:"message"`
}

func (s *Server) build(caller string, control bool) *sdk.Server {
	server := sdk.NewServer(&sdk.Implementation{Name: ServerName, Version: version}, nil)

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolRename,
		Description: "Set the display title of this session, so the human sees what it works on.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in renameIn) (*sdk.CallToolResult, okOut, error) {
		if err := s.sessions.SetTitle(caller, in.Title); err != nil {
			return nil, okOut{}, err
		}
		if in.Title == "" {
			return nil, okOut{OK: true, Message: "the title of " + caller + " is cleared"}, nil
		}
		return nil, okOut{OK: true, Message: caller + " is now titled " + in.Title}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolList,
		Description: "List every session the multiplexer knows, running and stored.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in listIn) (*sdk.CallToolResult, listOut, error) {
		all := s.sessions.List()
		out := make([]Session, 0, len(all))
		for _, item := range all {
			if in.LiveOnly && !item.Live {
				continue
			}
			out = append(out, item)
		}
		return nil, listOut{Sessions: out}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolMessages,
		Description: "Read the recent messages of a session, oldest first.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in messagesIn) (*sdk.CallToolResult, messagesOut, error) {
		target, err := cleanTarget(in.Session)
		if err != nil {
			return nil, messagesOut{}, err
		}
		messages, err := s.sessions.Messages(target, clampLimit(in.Limit))
		if err != nil {
			return nil, messagesOut{}, err
		}
		return nil, messagesOut{Session: target, Messages: messages}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolListJobs,
		Description: "List the background jobs of a session: id, description, task type, and status (running, done, failed, or killed). Give a session name, or leave it empty for this session.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in listJobsIn) (*sdk.CallToolResult, listJobsOut, error) {
		target, err := targetOrSelf(in.Session, caller)
		if err != nil {
			return nil, listJobsOut{}, err
		}
		jobs, err := s.sessions.Jobs(target)
		if err != nil {
			return nil, listJobsOut{}, err
		}
		return nil, listJobsOut{Session: target, Jobs: jobs}, nil
	})

	if !control {
		return server
	}

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

	return server
}

// targetOrSelf reads a session argument, and it returns the caller when the
// argument is empty, so a job tool defaults to the calling session.
func targetOrSelf(arg, caller string) (string, error) {
	if strings.TrimSpace(arg) == "" {
		return caller, nil
	}
	return cleanTarget(arg)
}

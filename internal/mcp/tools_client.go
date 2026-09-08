package mcp

import (
	"context"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// buildAPI registers the session-only tool set an external client reaches over
// MCP. It never registers a config, layout, or schedule tool. See docs/mcp/api.md.
func buildAPI(clientName string, sess APISessions) *sdk.Server {
	server := sdk.NewServer(&sdk.Implementation{Name: ServerName, Version: version}, nil)

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolList,
		Description: "List every session this client owns, running and stored.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in listIn) (*sdk.CallToolResult, listOut, error) {
		all := sess.List()
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
		Description: "Read the recent messages of a session this client owns, oldest first.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in messagesIn) (*sdk.CallToolResult, messagesOut, error) {
		target, err := cleanTarget(in.Session)
		if err != nil {
			return nil, messagesOut{}, err
		}
		messages, err := sess.Messages(target, clampLimit(in.Limit))
		if err != nil {
			return nil, messagesOut{}, err
		}
		return nil, messagesOut{Session: target, Messages: messages}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolListJobs,
		Description: "List the background jobs of a session this client owns.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in listJobsIn) (*sdk.CallToolResult, listJobsOut, error) {
		target, err := cleanTarget(in.Session)
		if err != nil {
			return nil, listJobsOut{}, err
		}
		jobs, err := sess.Jobs(target)
		if err != nil {
			return nil, listJobsOut{}, err
		}
		return nil, listJobsOut{Session: target, Jobs: jobs}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolRename,
		Description: "Set the display title of a session this client owns.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in apiRenameIn) (*sdk.CallToolResult, okOut, error) {
		target, err := cleanTarget(in.Session)
		if err != nil {
			return nil, okOut{}, err
		}
		if err := sess.SetTitle(target, in.Title); err != nil {
			return nil, okOut{}, err
		}
		return nil, okOut{OK: true, Message: target + " is renamed"}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolSend,
		Description: "Queue a prompt for a session this client owns. Read the answer later with " + ToolMessages + ".",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in sendIn) (*sdk.CallToolResult, sendOut, error) {
		target, err := cleanTarget(in.Session)
		if err != nil {
			return nil, sendOut{}, err
		}
		queued, err := sess.SendFrom(target, clientName, in.Text)
		if err != nil {
			return nil, sendOut{}, err
		}
		return nil, sendOut{OK: true, Queued: queued, Message: "queued for " + target}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolStop,
		Description: "End a session this client owns in a clean way. Its transcript is kept.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in targetIn) (*sdk.CallToolResult, okOut, error) {
		target, err := cleanTarget(in.Session)
		if err != nil {
			return nil, okOut{}, err
		}
		if err := sess.Stop(ctx, target, clientName); err != nil {
			return nil, okOut{}, err
		}
		return nil, okOut{OK: true, Message: target + " is stopped"}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolArchive,
		Description: "Take a stopped session this client owns out of the list, or with restore, bring it back.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in archiveIn) (*sdk.CallToolResult, okOut, error) {
		target, err := cleanTarget(in.Session)
		if err != nil {
			return nil, okOut{}, err
		}
		if err := sess.Archive(target, !in.Restore, clientName); err != nil {
			return nil, okOut{}, err
		}
		if in.Restore {
			return nil, okOut{OK: true, Message: target + " is back in the list"}, nil
		}
		return nil, okOut{OK: true, Message: target + " is archived"}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolCreate,
		Description: "Start a new session in a directory. This client owns the session it creates. It returns the name the session takes.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in createIn) (*sdk.CallToolResult, createOut, error) {
		path := strings.TrimSpace(in.Path)
		if path == "" {
			return nil, createOut{}, ErrNoPath
		}
		created, err := sess.Create(CreateInput{Dir: path, Name: strings.TrimSpace(in.Name)}, clientName)
		if err != nil {
			return nil, createOut{}, err
		}
		return nil, createOut{OK: true, Name: created, Message: "started " + created}, nil
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolStopJob,
		Description: "Stop a running background job of a session this client owns. Give the session name and the job id.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in stopJobIn) (*sdk.CallToolResult, stopJobOut, error) {
		target, err := cleanTarget(in.Session)
		if err != nil {
			return nil, stopJobOut{}, err
		}
		job := strings.TrimSpace(in.Job)
		if job == "" {
			return nil, stopJobOut{}, ErrNoJob
		}
		queued, err := sess.StopJob(target, job, clientName)
		if err != nil {
			return nil, stopJobOut{}, err
		}
		return nil, stopJobOut{OK: true, Queued: queued, Message: "stopping job " + job + " of " + target}, nil
	})

	return server
}

type apiRenameIn struct {
	Session string `json:"session" jsonschema:"the name of the session to rename"`
	Title   string `json:"title" jsonschema:"the new display title; an empty string clears it"`
}

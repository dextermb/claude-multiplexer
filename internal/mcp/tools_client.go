package mcp

import (
	"context"
	"strings"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// buildAPI registers the session-only tool set an external client reaches over
// MCP. It never registers a config, layout, or schedule tool. See docs/mcp/api.md.
func buildAPI(clientName string, sess APISessions) *sdk.Server {
	server := sdk.NewServer(&sdk.Implementation{Name: ServerName, Version: version}, nil)

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolList,
		Description: "List the sessions this client owns that run now. Set stopped true to add the stored sessions, and archived true to add the archived ones. last_active drops a stored or archived session older than its window (1d, 1w, 1m, 1y, or unset; 1d by default).",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in listIn) (*sdk.CallToolResult, listOut, error) {
		out, err := listSessions(sess, in, time.Now())
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolListInactive,
		Description: "List the sessions this client owns that are stored, do not run now, and are not archived. last_active drops a session older than its window (1d, 1w, 1m, 1y, or unset; 1d by default).",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in listCategoryIn) (*sdk.CallToolResult, listOut, error) {
		out, err := listCategory(sess, in, time.Now(), isInactive)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolListArchived,
		Description: "List the archived sessions this client owns. last_active drops a session older than its window (1d, 1w, 1m, 1y, or unset; 1d by default).",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in listCategoryIn) (*sdk.CallToolResult, listOut, error) {
		out, err := listCategory(sess, in, time.Now(), isArchived)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolMessages,
		Description: "Read the recent messages of a session this client owns, oldest first.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in messagesIn) (*sdk.CallToolResult, messagesOut, error) {
		out, err := messages(sess, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolListJobs,
		Description: "List the background jobs of a session this client owns.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in listJobsIn) (*sdk.CallToolResult, listJobsOut, error) {
		out, err := apiListJobs(sess, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolRename,
		Description: "Set the display title of a session this client owns.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in apiRenameIn) (*sdk.CallToolResult, okOut, error) {
		out, err := apiRename(sess, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolSend,
		Description: "Queue a prompt for a session this client owns. Read the answer later with " + ToolMessages + ".",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in sendIn) (*sdk.CallToolResult, sendOut, error) {
		out, err := apiSend(sess, clientName, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolStop,
		Description: "End a session this client owns in a clean way. Its transcript is kept.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in targetIn) (*sdk.CallToolResult, okOut, error) {
		out, err := apiStop(ctx, sess, clientName, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolArchive,
		Description: "Take a stopped session this client owns out of the list, or with restore, bring it back.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in archiveIn) (*sdk.CallToolResult, okOut, error) {
		out, err := apiArchive(sess, clientName, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolCreate,
		Description: "Start a new session in a directory. This client owns the session it creates. It returns the name the session takes.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in createIn) (*sdk.CallToolResult, createOut, error) {
		out, err := apiCreate(sess, clientName, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolStopJob,
		Description: "Stop a running background job of a session this client owns. Give the session name and the job id.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in stopJobIn) (*sdk.CallToolResult, stopJobOut, error) {
		out, err := apiStopJob(sess, clientName, in)
		return nil, out, err
	})

	return server
}

func apiListJobs(sess APISessions, in listJobsIn) (listJobsOut, error) {
	target, err := cleanTarget(in.Session)
	if err != nil {
		return listJobsOut{}, err
	}
	jobs, err := sess.Jobs(target)
	if err != nil {
		return listJobsOut{}, err
	}
	return listJobsOut{Session: target, Jobs: jobs}, nil
}

func apiRename(sess APISessions, in apiRenameIn) (okOut, error) {
	target, err := cleanTarget(in.Session)
	if err != nil {
		return okOut{}, err
	}
	if err := sess.SetTitle(target, in.Title); err != nil {
		return okOut{}, err
	}
	return okOut{OK: true, Message: target + " is renamed"}, nil
}

func apiSend(sess APISessions, clientName string, in sendIn) (sendOut, error) {
	target, err := cleanTarget(in.Session)
	if err != nil {
		return sendOut{}, err
	}
	queued, err := sess.SendFrom(target, clientName, in.Text)
	if err != nil {
		return sendOut{}, err
	}
	return sendOut{OK: true, Queued: queued, Message: "queued for " + target}, nil
}

func apiStop(ctx context.Context, sess APISessions, clientName string, in targetIn) (okOut, error) {
	target, err := cleanTarget(in.Session)
	if err != nil {
		return okOut{}, err
	}
	if err := sess.Stop(ctx, target, clientName); err != nil {
		return okOut{}, err
	}
	return okOut{OK: true, Message: target + " is stopped"}, nil
}

func apiArchive(sess APISessions, clientName string, in archiveIn) (okOut, error) {
	target, err := cleanTarget(in.Session)
	if err != nil {
		return okOut{}, err
	}
	if err := sess.Archive(target, !in.Restore, clientName); err != nil {
		return okOut{}, err
	}
	if in.Restore {
		return okOut{OK: true, Message: target + " is back in the list"}, nil
	}
	return okOut{OK: true, Message: target + " is archived"}, nil
}

func apiCreate(sess APISessions, clientName string, in createIn) (createOut, error) {
	path := strings.TrimSpace(in.Path)
	if path == "" {
		return createOut{}, ErrNoPath
	}
	created, err := sess.Create(CreateInput{Dir: path, Name: strings.TrimSpace(in.Name), Model: strings.TrimSpace(in.Model), Effort: strings.TrimSpace(in.Effort)}, clientName)
	if err != nil {
		return createOut{}, err
	}
	return createOut{OK: true, Name: created, Message: "started " + created}, nil
}

func apiStopJob(sess APISessions, clientName string, in stopJobIn) (stopJobOut, error) {
	target, err := cleanTarget(in.Session)
	if err != nil {
		return stopJobOut{}, err
	}
	job := strings.TrimSpace(in.Job)
	if job == "" {
		return stopJobOut{}, ErrNoJob
	}
	queued, err := sess.StopJob(target, job, clientName)
	if err != nil {
		return stopJobOut{}, err
	}
	return stopJobOut{OK: true, Queued: queued, Message: "stopping job " + job + " of " + target}, nil
}

type apiRenameIn struct {
	Session string `json:"session" jsonschema:"the name of the session to rename"`
	Title   string `json:"title" jsonschema:"the new display title; an empty string clears it"`
}

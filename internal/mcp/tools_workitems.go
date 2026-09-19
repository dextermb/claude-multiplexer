package mcp

import (
	"context"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// addWorkItemConfigTools registers the always-open configure tools. A session
// calls one to set a provider token, so it can turn the feature on. See
// docs/work-items.md.
func (s *Server) addWorkItemConfigTools(server *sdk.Server, caller string) {
	sdk.AddTool(server, &sdk.Tool{
		Name: ToolConfigureJira,
		Description: "Configure the Jira work-item provider, so this multiplexer can read and change Jira issues. " +
			"Give the API token. Give the account email for a personal token (Basic auth), or leave it empty for a service-account key (bearer). " +
			"An organisation admin must first turn on API token authentication in Atlassian. It writes the settings file.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in configureJiraIn) (*sdk.CallToolResult, configureWorkItemOut, error) {
		out, err := configureJira(s.sessions, caller, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolConfigureLinear,
		Description: "Configure the Linear work-item provider, so this multiplexer can read and change Linear issues. " +
			"Give the Linear API key, with write access. It writes the settings file.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in configureLinearIn) (*sdk.CallToolResult, configureWorkItemOut, error) {
		out, err := configureLinear(s.sessions, caller, in)
		return nil, out, err
	})
}

func (s *Server) addWorkItemTools(server *sdk.Server, caller string) {
	providers := strings.Join(s.sessions.WorkItemProviders(), " or ")

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolGetWorkItem,
		Description: "The work item a session links to, with its title, url, and last known status. " +
			"Give a session name, or leave it empty for this session.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, in getWorkItemIn) (*sdk.CallToolResult, workItemOut, error) {
		out, err := getWorkItem(s.sessions, caller, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolSetWorkItem,
		Description: "Link this session to a work item on " + providers + ", by its key. " +
			"It reads the item and shows its title and current status on the session row. " +
			"Name the provider when more than one is configured.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in setWorkItemIn) (*sdk.CallToolResult, workItemOut, error) {
		out, err := setWorkItem(ctx, s.sessions, caller, in)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name:        ToolUnsetWorkItem,
		Description: "Clear the work item this session links to, so the session row shows no status.",
	}, func(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, workItemOut, error) {
		out, err := unsetWorkItem(s.sessions, caller)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolWorkItemStatuses,
		Description: "The statuses the linked work item may move to now. " +
			"Read this to pick a real status before you call " + ToolSetWorkItemStatus + ".",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, workItemStatusesOut, error) {
		out, err := workItemStatuses(ctx, s.sessions, caller)
		return nil, out, err
	})

	sdk.AddTool(server, &sdk.Tool{
		Name: ToolSetWorkItemStatus,
		Description: "Move the linked work item to a status. " +
			"The status must be one of the statuses the platform offers, from " + ToolWorkItemStatuses + ". " +
			"It changes the status on the provider, then updates the session row.",
	}, func(ctx context.Context, _ *sdk.CallToolRequest, in setWorkItemStatusIn) (*sdk.CallToolResult, workItemOut, error) {
		out, err := setWorkItemStatus(ctx, s.sessions, caller, in)
		return nil, out, err
	})
}

func configureJira(port WorkItemPort, caller string, in configureJiraIn) (configureWorkItemOut, error) {
	return configureWorkItem(port, caller, "jira", in.Token, in.Email, in.URL)
}

func configureLinear(port WorkItemPort, caller string, in configureLinearIn) (configureWorkItemOut, error) {
	return configureWorkItem(port, caller, "linear", in.Token, "", in.URL)
}

func configureWorkItem(port WorkItemPort, caller, provider, token, email, url string) (configureWorkItemOut, error) {
	if strings.TrimSpace(token) == "" {
		return configureWorkItemOut{}, ErrNoWorkItemToken
	}
	path, err := port.ConfigureWorkItem(provider, strings.TrimSpace(token), strings.TrimSpace(email), strings.TrimSpace(url), caller)
	if err != nil {
		return configureWorkItemOut{}, err
	}
	return configureWorkItemOut{OK: true, Provider: provider, Path: path,
		Message: provider + " is configured, in " + path + "; a new session carries the work-item tools"}, nil
}

func getWorkItem(port WorkItemPort, caller string, in getWorkItemIn) (workItemOut, error) {
	target, err := targetOrSelf(in.Session, caller)
	if err != nil {
		return workItemOut{}, err
	}
	item, err := port.WorkItem(target)
	if err != nil {
		return workItemOut{}, err
	}
	return workItemOut{OK: true, Linked: item.Linked, Item: item,
		Message: workItemMessage(target, item)}, nil
}

func setWorkItem(ctx context.Context, port WorkItemPort, caller string, in setWorkItemIn) (workItemOut, error) {
	key := strings.TrimSpace(in.Key)
	if key == "" {
		return workItemOut{}, ErrNoWorkItemKey
	}
	item, err := port.SetWorkItem(ctx, strings.TrimSpace(in.Provider), key, caller)
	if err != nil {
		return workItemOut{}, err
	}
	return workItemOut{OK: true, Linked: true, Item: item,
		Message: caller + " links to " + item.Key + " (" + item.Status + ")"}, nil
}

func unsetWorkItem(port WorkItemPort, caller string) (workItemOut, error) {
	changed, err := port.UnsetWorkItem(caller)
	if err != nil {
		return workItemOut{}, err
	}
	message := caller + " linked to no work item"
	if changed {
		message = caller + " no longer links to a work item"
	}
	return workItemOut{OK: true, Linked: false, Message: message}, nil
}

func workItemStatuses(ctx context.Context, port WorkItemPort, caller string) (workItemStatusesOut, error) {
	statuses, err := port.WorkItemStatuses(ctx, caller)
	if err != nil {
		return workItemStatusesOut{}, err
	}
	return workItemStatusesOut{OK: true, Statuses: statuses,
		Message: statusesMessage(caller, statuses)}, nil
}

func setWorkItemStatus(ctx context.Context, port WorkItemPort, caller string, in setWorkItemStatusIn) (workItemOut, error) {
	target := strings.TrimSpace(in.Status)
	if target == "" {
		return workItemOut{}, ErrNoWorkItemStatus
	}
	item, err := port.SetWorkItemStatus(ctx, target, caller)
	if err != nil {
		return workItemOut{}, err
	}
	return workItemOut{OK: true, Linked: true, Item: item,
		Message: item.Key + " is now " + item.Status}, nil
}

func workItemMessage(session string, item WorkItem) string {
	if !item.Linked {
		return session + " links to no work item"
	}
	return session + " links to " + item.Key + " (" + item.Status + ")"
}

func statusesMessage(session string, statuses []WorkItemStatus) string {
	if len(statuses) == 0 {
		return session + " has no target status"
	}
	names := make([]string, 0, len(statuses))
	for _, s := range statuses {
		names = append(names, s.Name)
	}
	return "the statuses are: " + strings.Join(names, ", ")
}

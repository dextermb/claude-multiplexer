package workitem

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// The Linear MCP tools the multiplexer calls. See docs/work-items.md.
const (
	linearGetIssue      = "get_issue"
	linearListStatuses  = "list_issue_statuses"
	linearSaveIssue     = "save_issue"
	linearStatusTeamKey = "team"
)

// linear reads and changes a Linear issue. The save_issue tool takes a status
// name, so the server resolves the name to a workflow state. See
// docs/work-items.md.
type linear struct {
	conn connector
}

func (l *linear) do(ctx context.Context, fn func(*conn) error) error {
	c, err := l.conn.open(ctx)
	if err != nil {
		return err
	}
	defer c.close()
	return fn(c)
}

func (l *linear) Resolve(ctx context.Context, key string) (Item, error) {
	var item Item
	err := l.do(ctx, func(c *conn) error {
		got, err := l.resolve(ctx, c, key)
		item = got
		return err
	})
	return item, err
}

func (l *linear) resolve(ctx context.Context, c *conn, key string) (Item, error) {
	raw, err := c.call(ctx, linearGetIssue, map[string]any{"id": key})
	if err != nil {
		return Item{}, err
	}
	issue := parseLinearIssue(raw)
	return Item{
		Key:    key,
		Title:  issue.title,
		URL:    issue.url,
		Status: issue.status,
	}, nil
}

func (l *linear) Statuses(ctx context.Context, key string) ([]Status, error) {
	var out []Status
	err := l.do(ctx, func(c *conn) error {
		got, err := l.statuses(ctx, c, key)
		out = got
		return err
	})
	return out, err
}

func (l *linear) statuses(ctx context.Context, c *conn, key string) ([]Status, error) {
	raw, err := c.call(ctx, linearGetIssue, map[string]any{"id": key})
	if err != nil {
		return nil, err
	}
	team := parseLinearIssue(raw).team
	if team == "" {
		return nil, fmt.Errorf("workitem: linear issue %s names no team", key)
	}
	statuses, err := c.call(ctx, linearListStatuses, map[string]any{linearStatusTeamKey: team})
	if err != nil {
		return nil, err
	}
	return parseStatusList(statuses), nil
}

func (l *linear) SetStatus(ctx context.Context, key, target string) (Item, error) {
	var item Item
	err := l.do(ctx, func(c *conn) error {
		statuses, err := l.statuses(ctx, c, key)
		if err != nil {
			return err
		}
		match, ok := matchStatus(statuses, target)
		if !ok {
			return fmt.Errorf("workitem: %q is not a status of %s; the statuses are: %s",
				target, key, statusNames(statuses))
		}
		if _, err := c.call(ctx, linearSaveIssue, map[string]any{"id": key, "state": match.Name}); err != nil {
			return err
		}
		got, err := l.resolve(ctx, c, key)
		item = got
		return err
	})
	return item, err
}

// linearIssue holds the fields the multiplexer reads from a Linear issue. The
// shapes the server sends are tolerant: a state or team is a string or an object
// with a name. See docs/work-items.md.
type linearIssue struct {
	title  string
	url    string
	status string
	team   string
}

func parseLinearIssue(raw json.RawMessage) linearIssue {
	var doc struct {
		Identifier string          `json:"identifier"`
		Title      string          `json:"title"`
		Name       string          `json:"name"`
		URL        string          `json:"url"`
		State      json.RawMessage `json:"state"`
		Status     json.RawMessage `json:"status"`
		Team       json.RawMessage `json:"team"`
		Issue      json.RawMessage `json:"issue"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return linearIssue{}
	}
	if len(doc.Issue) > 0 && doc.Title == "" && doc.Name == "" {
		return parseLinearIssue(doc.Issue)
	}
	title := doc.Title
	if title == "" {
		title = doc.Name
	}
	status := parseName(doc.State)
	if status == "" {
		status = parseName(doc.Status)
	}
	return linearIssue{
		title:  title,
		url:    doc.URL,
		status: status,
		team:   parseTeam(doc.Team),
	}
}

// parseTeam reads a team key from a string or an object. It prefers the id, so
// list_issue_statuses gets a stable key.
func parseTeam(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}
	var obj struct {
		ID   string `json:"id"`
		Key  string `json:"key"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		switch {
		case obj.ID != "":
			return obj.ID
		case obj.Key != "":
			return obj.Key
		default:
			return obj.Name
		}
	}
	return ""
}

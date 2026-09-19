package workitem

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// The Jira (Atlassian Rovo) MCP tools the multiplexer calls. Jira changes a
// status through a workflow transition, not by name, so a set reads the valid
// transitions first. See docs/work-items.md.
const (
	jiraResources   = "getAccessibleAtlassianResources"
	jiraGetIssue    = "getJiraIssue"
	jiraTransitions = "getTransitionsForJiraIssue"
	jiraTransition  = "transitionJiraIssue"
)

// jira reads and changes a Jira issue. Every call carries the cloudId of the
// site, which the multiplexer resolves once per operation. See
// docs/work-items.md.
type jira struct {
	conn connector
}

func (j *jira) do(ctx context.Context, fn func(*conn, jiraSite) error) error {
	c, err := j.conn.open(ctx)
	if err != nil {
		return err
	}
	defer c.close()
	site, err := j.site(ctx, c)
	if err != nil {
		return err
	}
	return fn(c, site)
}

func (j *jira) Resolve(ctx context.Context, key string) (Item, error) {
	var item Item
	err := j.do(ctx, func(c *conn, site jiraSite) error {
		got, err := j.resolve(ctx, c, site, key)
		item = got
		return err
	})
	return item, err
}

func (j *jira) resolve(ctx context.Context, c *conn, site jiraSite, key string) (Item, error) {
	raw, err := c.call(ctx, jiraGetIssue, map[string]any{"cloudId": site.cloudID, "issueIdOrKey": key})
	if err != nil {
		return Item{}, err
	}
	issue := parseJiraIssue(raw)
	return Item{
		Key:    key,
		Title:  issue.summary,
		URL:    site.browseURL(key),
		Status: issue.status,
	}, nil
}

func (j *jira) Statuses(ctx context.Context, key string) ([]Status, error) {
	var out []Status
	err := j.do(ctx, func(c *conn, site jiraSite) error {
		got, err := j.statuses(ctx, c, site, key)
		out = got
		return err
	})
	return out, err
}

func (j *jira) statuses(ctx context.Context, c *conn, site jiraSite, key string) ([]Status, error) {
	raw, err := c.call(ctx, jiraTransitions, map[string]any{"cloudId": site.cloudID, "issueIdOrKey": key})
	if err != nil {
		return nil, err
	}
	return parseJiraTransitions(raw), nil
}

func (j *jira) SetStatus(ctx context.Context, key, target string) (Item, error) {
	var item Item
	err := j.do(ctx, func(c *conn, site jiraSite) error {
		statuses, err := j.statuses(ctx, c, site, key)
		if err != nil {
			return err
		}
		match, ok := matchStatus(statuses, target)
		if !ok {
			return fmt.Errorf("workitem: %q is not a transition of %s; the statuses are: %s",
				target, key, statusNames(statuses))
		}
		_, err = c.call(ctx, jiraTransition, map[string]any{
			"cloudId":      site.cloudID,
			"issueIdOrKey": key,
			"transition":   map[string]any{"id": match.ID},
		})
		if err != nil {
			return err
		}
		got, err := j.resolve(ctx, c, site, key)
		item = got
		return err
	})
	return item, err
}

// jiraSite is one Atlassian site the token reaches: the cloudId every call
// carries, and the base url the browse link is built from.
type jiraSite struct {
	cloudID string
	baseURL string
}

func (s jiraSite) browseURL(key string) string {
	if s.baseURL == "" {
		return ""
	}
	return strings.TrimRight(s.baseURL, "/") + "/browse/" + key
}

// site resolves the first Atlassian site the token reaches.
func (j *jira) site(ctx context.Context, c *conn) (jiraSite, error) {
	raw, err := c.call(ctx, jiraResources, map[string]any{})
	if err != nil {
		return jiraSite{}, err
	}
	items := unwrapArray(raw)
	for _, item := range items {
		var res struct {
			ID  string `json:"id"`
			URL string `json:"url"`
		}
		if err := json.Unmarshal(item, &res); err == nil && res.ID != "" {
			return jiraSite{cloudID: res.ID, baseURL: res.URL}, nil
		}
	}
	return jiraSite{}, fmt.Errorf("workitem: the Jira token reaches no site")
}

type jiraIssue struct {
	summary string
	status  string
}

func parseJiraIssue(raw json.RawMessage) jiraIssue {
	var doc struct {
		Key    string `json:"key"`
		Fields struct {
			Summary string          `json:"summary"`
			Status  json.RawMessage `json:"status"`
		} `json:"fields"`
		Summary string          `json:"summary"`
		Status  json.RawMessage `json:"status"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return jiraIssue{}
	}
	summary := doc.Fields.Summary
	if summary == "" {
		summary = doc.Summary
	}
	status := parseName(doc.Fields.Status)
	if status == "" {
		status = parseName(doc.Status)
	}
	return jiraIssue{summary: summary, status: status}
}

// parseJiraTransitions reads the valid transitions of an issue. The status a
// transition moves to (its "to" state) is the target the multiplexer matches
// against, and the transition id is what the set call carries. See
// docs/work-items.md.
func parseJiraTransitions(raw json.RawMessage) []Status {
	items := unwrapArray(raw)
	out := make([]Status, 0, len(items))
	for _, item := range items {
		var tr struct {
			ID   string          `json:"id"`
			Name string          `json:"name"`
			To   json.RawMessage `json:"to"`
		}
		if err := json.Unmarshal(item, &tr); err != nil || tr.ID == "" {
			continue
		}
		name := parseName(tr.To)
		if name == "" {
			name = tr.Name
		}
		if name == "" {
			continue
		}
		out = append(out, Status{ID: tr.ID, Name: name})
	}
	return out
}

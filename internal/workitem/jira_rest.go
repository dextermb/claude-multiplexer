package workitem

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// jiraREST reads and changes a Jira issue through the Jira REST API, the path
// that needs no Rovo MCP admin gate. The site base comes from the configured
// url, so no cloudId is resolved. See docs/work-items.md.
type jiraREST struct {
	site  string
	auth  string
	httpc *http.Client
}

func (j *jiraREST) issuePath(key string) string {
	return "/rest/api/3/issue/" + key
}

func (j *jiraREST) Resolve(ctx context.Context, key string) (Item, error) {
	raw, err := j.do(ctx, http.MethodGet, j.issuePath(key)+"?fields=summary,status", nil)
	if err != nil {
		return Item{}, err
	}
	issue := parseJiraIssue(raw)
	return Item{
		Key:    key,
		Title:  issue.summary,
		URL:    strings.TrimRight(j.site, "/") + "/browse/" + key,
		Status: issue.status,
	}, nil
}

func (j *jiraREST) Statuses(ctx context.Context, key string) ([]Status, error) {
	raw, err := j.do(ctx, http.MethodGet, j.issuePath(key)+"/transitions", nil)
	if err != nil {
		return nil, err
	}
	return parseJiraTransitions(raw), nil
}

func (j *jiraREST) SetStatus(ctx context.Context, key, target string) (Item, error) {
	statuses, err := j.Statuses(ctx, key)
	if err != nil {
		return Item{}, err
	}
	match, ok := matchStatus(statuses, target)
	if !ok {
		return Item{}, fmt.Errorf("workitem: %q is not a transition of %s; the statuses are: %s",
			target, key, statusNames(statuses))
	}
	body := map[string]any{"transition": map[string]any{"id": match.ID}}
	if _, err := j.do(ctx, http.MethodPost, j.issuePath(key)+"/transitions", body); err != nil {
		return Item{}, err
	}
	return j.Resolve(ctx, key)
}

// do runs one REST call and returns its body as JSON. A transition returns no
// body, so an empty body reads as null.
func (j *jiraREST) do(ctx context.Context, method, path string, body any) (json.RawMessage, error) {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(j.site, "/")+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if j.auth != "" {
		req.Header.Set("Authorization", j.auth)
	}
	res, err := j.httpc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("workitem: %s %s: %w", method, path, err)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, fmt.Errorf("workitem: %s %s: %s: %s", method, path, res.Status, jiraRESTError(raw))
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return json.RawMessage("null"), nil
	}
	return json.RawMessage(raw), nil
}

// jiraRESTError reads the messages Jira returns with a failure, so the error
// names the cause instead of the whole body.
func jiraRESTError(raw []byte) string {
	var doc struct {
		ErrorMessages []string          `json:"errorMessages"`
		Errors        map[string]string `json:"errors"`
		Message       string            `json:"message"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return strings.TrimSpace(string(raw))
	}
	parts := append([]string{}, doc.ErrorMessages...)
	for field, message := range doc.Errors {
		parts = append(parts, field+": "+message)
	}
	if doc.Message != "" {
		parts = append(parts, doc.Message)
	}
	if len(parts) == 0 {
		return strings.TrimSpace(string(raw))
	}
	return strings.Join(parts, "; ")
}

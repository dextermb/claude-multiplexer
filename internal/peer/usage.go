package peer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/dextermb/claude-multiplexer/internal/usage"
)

// Usage reads the peer's usage from GET /api/usage. It runs the grant first, and
// retries once with a fresh token when the peer answers 401.
func (c *Client) Usage(ctx context.Context) (usage.Usage, error) {
	body, err := c.get(ctx, "/api/usage")
	if err != nil {
		return usage.Usage{}, err
	}
	var out usage.Usage
	if err := json.Unmarshal(body, &out); err != nil {
		return usage.Usage{}, err
	}
	return out, nil
}

// get does an authenticated GET, and retries once with a fresh token on 401.
func (c *Client) get(ctx context.Context, path string) ([]byte, error) {
	for attempt := 0; attempt < 2; attempt++ {
		resp, retry, err := c.do(ctx, http.MethodGet, path, nil, attempt)
		if err != nil {
			return nil, err
		}
		if retry {
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("peer %s: GET %s: %s", c.host.Name, path, resp.Status)
		}
		return io.ReadAll(resp.Body)
	}
	return nil, fmt.Errorf("peer %s: GET %s: unauthorized", c.host.Name, path)
}

// do runs one authenticated request. It returns retry=true when the peer answers
// 401 on the first attempt, so the caller runs the grant again with a fresh
// token. The caller closes the response body when retry is false.
func (c *Client) do(ctx context.Context, method, path string, body []byte, attempt int) (resp *http.Response, retry bool, err error) {
	token := c.shareToken
	if token == "" {
		token, err = c.accessToken(ctx, attempt == 1)
		if err != nil {
			return nil, false, err
		}
	}
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.url(path), reader)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err = c.http.Do(req)
	if err != nil {
		return nil, false, err
	}
	// A share token is fixed, so a 401 is terminal and never retried.
	if resp.StatusCode == http.StatusUnauthorized && attempt == 0 && c.shareToken == "" {
		resp.Body.Close()
		return nil, true, nil
	}
	return resp, false, nil
}

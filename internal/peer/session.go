package peer

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/dextermb/claude-multiplexer/internal/wire"
)

// CreateSpec is the input to start a session on a peer.
type CreateSpec struct {
	Dir            string `json:"dir"`
	Name           string `json:"name,omitempty"`
	Model          string `json:"model,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	Effort         string `json:"effort,omitempty"`
}

// CreateSession starts a session on the peer and returns the name it takes. The
// session is owned by this host's client, so only this host reaches it. See
// docs/peers.md.
func (c *Client) CreateSession(ctx context.Context, spec CreateSpec) (string, error) {
	body, err := c.post(ctx, "/api/sessions", spec)
	if err != nil {
		return "", err
	}
	var out struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}
	return out.Name, nil
}

// Send queues a prompt for a session on the peer.
func (c *Client) Send(ctx context.Context, name, text string) error {
	_, err := c.post(ctx, "/api/sessions/"+name+"/message", map[string]string{"text": text})
	return err
}

// Stop stops a session on the peer.
func (c *Client) Stop(ctx context.Context, name string) error {
	_, err := c.post(ctx, "/api/sessions/"+name+"/stop", map[string]string{})
	return err
}

// Stream opens the session's event stream on the peer and decodes each SSE event
// into a wire event. The channel closes when the stream ends or the context is
// done. See docs/peers.md.
func (c *Client) Stream(ctx context.Context, name string) (<-chan wire.Event, error) {
	var resp *http.Response
	for attempt := 0; attempt < 2; attempt++ {
		got, retry, err := c.do(ctx, http.MethodGet, "/api/sessions/"+name+"/stream", nil, attempt)
		if err != nil {
			return nil, err
		}
		if retry {
			continue
		}
		if got.StatusCode != http.StatusOK {
			got.Body.Close()
			return nil, fmt.Errorf("peer %s: stream %s: %s", c.host.Name, name, got.Status)
		}
		resp = got
		break
	}
	if resp == nil {
		return nil, fmt.Errorf("peer %s: stream %s: unauthorized", c.host.Name, name)
	}

	out := make(chan wire.Event)
	go func() {
		defer close(out)
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			data, ok := strings.CutPrefix(line, "data: ")
			if !ok {
				continue
			}
			var ev wire.Event
			if err := json.Unmarshal([]byte(data), &ev); err != nil {
				continue
			}
			select {
			case out <- ev:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}

// post does an authenticated POST of a JSON body, and retries once with a fresh
// token on 401.
func (c *Client) post(ctx context.Context, path string, payload any) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	for attempt := 0; attempt < 2; attempt++ {
		resp, retry, err := c.do(ctx, http.MethodPost, path, data, attempt)
		if err != nil {
			return nil, err
		}
		if retry {
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("peer %s: POST %s: %s", c.host.Name, path, resp.Status)
		}
		return io.ReadAll(resp.Body)
	}
	return nil, fmt.Errorf("peer %s: POST %s: unauthorized", c.host.Name, path)
}

// Package peer reaches another multiplexer host over its peer listener. A Client
// runs the client-credentials grant, caches the access token, and reads the
// peer's usage. See docs/peers.md.
package peer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/config"
)

// tokenSlack is the margin taken off a token's lifetime, so a request does not
// carry a token that expires in flight.
const tokenSlack = 30 * time.Second

// Client reaches one peer host. It caches the access token until it expires, so
// a repeat read does not run the grant again.
type Client struct {
	host config.PeerHost
	http *http.Client

	mu     sync.Mutex
	token  string
	expiry time.Time
}

// New builds a client for one peer host.
func New(host config.PeerHost) *Client {
	return &Client{host: host, http: &http.Client{Timeout: 10 * time.Second}}
}

// Name is the label of the peer.
func (c *Client) Name() string { return c.host.Name }

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// accessToken returns a live token, and runs the grant when the cache is empty
// or expired.
func (c *Client) accessToken(ctx context.Context, refresh bool) (string, error) {
	c.mu.Lock()
	if !refresh && c.token != "" && time.Now().Before(c.expiry) {
		token := c.token
		c.mu.Unlock()
		return token, nil
	}
	c.mu.Unlock()

	body, err := json.Marshal(map[string]string{
		"grant_type":    "client_credentials",
		"client_id":     c.host.ClientID,
		"client_secret": c.host.ClientSecret,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url("/token"), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("peer %s: token grant: %s", c.host.Name, resp.Status)
	}
	var out tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	c.mu.Lock()
	c.token = out.AccessToken
	c.expiry = time.Now().Add(time.Duration(out.ExpiresIn)*time.Second - tokenSlack)
	c.mu.Unlock()
	return out.AccessToken, nil
}

// url joins the peer base URL and a path.
func (c *Client) url(path string) string {
	return strings.TrimRight(c.host.URL, "/") + path
}

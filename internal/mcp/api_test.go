package mcp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/dextermb/claude-multiplexer/internal/api"
	"github.com/dextermb/claude-multiplexer/internal/mcp"
)

// fakeAPI is a canned owner-scoped view. It owns one session, "mine", and it
// answers ErrNotFound for any other session, the way the real view does.
type fakeAPI struct{}

func (fakeAPI) List() []mcp.Session {
	return []mcp.Session{{Name: "mine", State: "idle", Owner: "c1"}}
}

func (fakeAPI) Messages(name string, _ int) ([]mcp.Message, error) {
	if name != "mine" {
		return nil, fmt.Errorf("%w: %s", mcp.ErrNotFound, name)
	}
	return []mcp.Message{{Role: "user", Text: "hi"}}, nil
}

func (fakeAPI) Jobs(name string) ([]mcp.Job, error) {
	if name != "mine" {
		return nil, fmt.Errorf("%w: %s", mcp.ErrNotFound, name)
	}
	return nil, nil
}

func (fakeAPI) SetTitle(name, _ string) error {
	if name != "mine" {
		return fmt.Errorf("%w: %s", mcp.ErrNotFound, name)
	}
	return nil
}

func (fakeAPI) SendFrom(target, _, _ string) (int, error) {
	if target != "mine" {
		return 0, fmt.Errorf("%w: %s", mcp.ErrNotFound, target)
	}
	return 1, nil
}

func (fakeAPI) Stop(_ context.Context, name, _ string) error {
	if name != "mine" {
		return fmt.Errorf("%w: %s", mcp.ErrNotFound, name)
	}
	return nil
}

func (fakeAPI) Archive(name string, _ bool, _ string) error {
	if name != "mine" {
		return fmt.Errorf("%w: %s", mcp.ErrNotFound, name)
	}
	return nil
}

func (fakeAPI) Create(_, _, _ string) (string, error) { return "new", nil }

func (fakeAPI) StopJob(target, _, _ string) (int, error) {
	if target != "mine" {
		return 0, fmt.Errorf("%w: %s", mcp.ErrNotFound, target)
	}
	return 0, nil
}

func startAPIServer(t *testing.T) (*mcp.Server, *api.Store) {
	t.Helper()
	store, err := api.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	server := mcp.NewServer(newFakeSessions())
	server.EnableAPI(store, func(_, _ string) mcp.APISessions { return fakeAPI{} })
	if err := server.Start(0, 0); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Close(ctx)
	})
	return server, store
}

func grant(t *testing.T, base, id, secret string) string {
	t.Helper()
	form := url.Values{"grant_type": {"client_credentials"}, "client_id": {id}, "client_secret": {secret}}
	resp, err := http.Post(base+"/token", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("token status: %d", resp.StatusCode)
	}
	var body struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode token: %v", err)
	}
	if body.TokenType != "Bearer" || body.AccessToken == "" || body.ExpiresIn <= 0 {
		t.Fatalf("bad token response: %+v", body)
	}
	return body.AccessToken
}

func apiGet(t *testing.T, base, path, token string) (int, string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, base+path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get %s: %v", path, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(data)
}

func TestTokenGrant(t *testing.T) {
	server, store := startAPIServer(t)
	if _, err := store.CreateAdmin(); err != nil {
		t.Fatalf("create admin: %v", err)
	}
	client, secret, err := store.CreateClient("bruno")
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	token := grant(t, server.BaseURL(), client.ClientID, secret)
	if token == "" {
		t.Fatal("empty access token")
	}

	form := url.Values{"grant_type": {"client_credentials"}, "client_id": {client.ClientID}, "client_secret": {"wrong"}}
	resp, err := http.Post(server.BaseURL()+"/token", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong secret: want 401, got %d", resp.StatusCode)
	}
}

func TestRESTSurface(t *testing.T) {
	server, store := startAPIServer(t)
	_, _ = store.CreateAdmin()
	client, secret, _ := store.CreateClient("bruno")
	token := grant(t, server.BaseURL(), client.ClientID, secret)
	base := server.BaseURL()

	if code, _ := apiGet(t, base, "/api/sessions", ""); code != http.StatusUnauthorized {
		t.Fatalf("no token: want 401, got %d", code)
	}
	code, body := apiGet(t, base, "/api/sessions", token)
	if code != http.StatusOK {
		t.Fatalf("list: want 200, got %d", code)
	}
	if !strings.Contains(body, `"mine"`) {
		t.Fatalf("list did not hold the owned session: %s", body)
	}
	if code, _ := apiGet(t, base, "/api/sessions/mine/messages", token); code != http.StatusOK {
		t.Fatalf("owned messages: want 200, got %d", code)
	}
	if code, _ := apiGet(t, base, "/api/sessions/other/messages", token); code != http.StatusNotFound {
		t.Fatalf("unowned messages: want 404, got %d", code)
	}
}

func TestAdminSurface(t *testing.T) {
	server, store := startAPIServer(t)
	adminSecret, err := store.CreateAdmin()
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	base := server.BaseURL()

	req, _ := http.NewRequest(http.MethodPost, base+"/admin/clients", strings.NewReader(`{"name":"bruno"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create without secret: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no admin secret: want 401, got %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodPost, base+"/admin/clients", strings.NewReader(`{"name":"bruno"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminSecret)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create with secret: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin create: want 200, got %d", resp.StatusCode)
	}
	var created struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.ClientID == "" || created.ClientSecret == "" {
		t.Fatal("admin create returned an empty id or secret")
	}
}

func TestAPIToolSetIsSessionOnly(t *testing.T) {
	server, store := startAPIServer(t)
	_, _ = store.CreateAdmin()
	client, secret, _ := store.CreateClient("bruno")
	token := grant(t, server.BaseURL(), client.ClientID, secret)

	session := connect(t, server, token)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	got := make(map[string]bool)
	for _, tool := range tools.Tools {
		got[tool.Name] = true
	}
	for _, name := range mcp.APITools {
		if !got[name] {
			t.Errorf("API tool %s is missing", name)
		}
	}
	forbidden := []string{
		mcp.ToolSetConfig, mcp.ToolSetEditor, mcp.ToolSetBlockCap, mcp.ToolSetWorkingDir,
		mcp.ToolListProject, mcp.ToolSaveLayout, mcp.ToolSetLayout,
		mcp.ToolCreateSchedule, mcp.ToolListSchedules, mcp.ToolCreateAPIClient,
	}
	for _, name := range forbidden {
		if got[name] {
			t.Errorf("a local-configuration tool reached an API client: %s", name)
		}
	}
	if len(got) != len(mcp.APITools) {
		t.Errorf("the API client sees %d tools, want %d", len(got), len(mcp.APITools))
	}
}

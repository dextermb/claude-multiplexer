package manager

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/dextermb/claude-multiplexer/internal/api"
	"github.com/dextermb/claude-multiplexer/internal/config"
	"github.com/dextermb/claude-multiplexer/internal/mcp"
	"github.com/dextermb/claude-multiplexer/internal/session"
)

func TestPeerCredentialRoundTrip(t *testing.T) {
	m := newTestManager(t)
	path := filepath.Join(t.TempDir(), config.FileName)
	m.opts.ConfigPaths = []string{path}

	const value = "sk-ant-oat01-secret1234"
	if _, err := m.AddPeer(mcp.PeerHostInput{Name: "studio", CredentialType: "token", Credential: value}); err != nil {
		t.Fatal(err)
	}

	view := m.Peers()
	if len(view.Hosts) != 1 || view.Hosts[0].Credential == nil {
		t.Fatalf("the peer view must carry a credential, got %+v", view.Hosts)
	}
	cred := view.Hosts[0].Credential
	if cred.Type != "token" || cred.Last4 != "1234" {
		t.Errorf("credential view = %+v, want type token last4 1234", cred)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Peers.Hosts[0].Credential.Value != value {
		t.Error("the full credential value must be written to the borrower's config")
	}

	if _, err := m.UpdatePeer(mcp.PeerHostUpdate{Name: "studio", ClearCredential: true}); err != nil {
		t.Fatal(err)
	}
	if v := m.Peers(); v.Hosts[0].Credential != nil {
		t.Error("clear_credential must remove the credential")
	}
}

func TestAddPeerRejectsBadCredentialType(t *testing.T) {
	m := newTestManager(t)
	m.opts.ConfigPaths = []string{filepath.Join(t.TempDir(), config.FileName)}
	_, err := m.AddPeer(mcp.PeerHostInput{Name: "studio", CredentialType: "oauth", Credential: "x"})
	if !errors.Is(err, errBadCredentialType) {
		t.Errorf("a bad credential type must error, got %v", err)
	}
}

func TestHoistBuildsEnvAndSeeds(t *testing.T) {
	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "commands"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "commands", "hello.md"), []byte("hi"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, ".claude.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_CONFIG_DIR", src)

	m := newTestManager(t)
	m.opts.ConfigPaths = []string{filepath.Join(t.TempDir(), config.FileName)}
	if _, err := m.AddPeer(mcp.PeerHostInput{Name: "studio", CredentialType: "token", Credential: "sk-ant-oat01-abcd"}); err != nil {
		t.Fatal(err)
	}

	cfg := session.Config{}
	if err := m.hoist(&cfg, "studio"); err != nil {
		t.Fatal(err)
	}

	dir := filepath.Join(m.opts.Root, "hoisted", "studio")
	if !slices.Contains(cfg.Env, "CLAUDE_CODE_OAUTH_TOKEN=sk-ant-oat01-abcd") {
		t.Errorf("env must inject the token, got %v", cfg.Env)
	}
	if !slices.Contains(cfg.Env, "CLAUDE_CONFIG_DIR="+dir) {
		t.Errorf("env must set the per-lender config dir, got %v", cfg.Env)
	}
	if !slices.Equal(cfg.EnvScrub, config.AuthEnvVars) {
		t.Errorf("env scrub = %v, want %v", cfg.EnvScrub, config.AuthEnvVars)
	}
	if _, err := os.Stat(filepath.Join(dir, "commands", "hello.md")); err != nil {
		t.Error("the borrower's commands must be seeded into the config dir")
	}
	if _, err := os.Lstat(filepath.Join(dir, ".claude.json")); err != nil {
		t.Error("claude.json must be linked into the config dir")
	}
}

func TestHoistWithoutCredentialFails(t *testing.T) {
	m := newTestManager(t)
	m.opts.ConfigPaths = []string{filepath.Join(t.TempDir(), config.FileName)}
	if _, err := m.AddPeer(mcp.PeerHostInput{Name: "studio", URL: "http://host:51900", ClientID: "id", ClientSecret: "sec"}); err != nil {
		t.Fatal(err)
	}
	cfg := session.Config{}
	if err := m.hoist(&cfg, "studio"); !errors.Is(err, errNoLentCredential) {
		t.Errorf("hoist without a credential must error, got %v", err)
	}
	if err := m.hoist(&cfg, "missing"); !errors.Is(err, errUnknownPeer) {
		t.Errorf("hoist of an unknown peer must error, got %v", err)
	}
}

func TestCreateAPIKeyValidatesType(t *testing.T) {
	m := newTestManager(t)
	store, err := api.Open(m.opts.Root)
	if err != nil {
		t.Fatal(err)
	}
	m.apiStore = store
	client, _, err := m.CreateAPIClient("laptop")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := m.CreateAPIKey(client.ClientID, "bogus", "x"); !errors.Is(err, errBadCredentialType) {
		t.Errorf("a bad type must error, got %v", err)
	}
	view, err := m.CreateAPIKey(client.ClientID, "key", "sk-ant-api03-wxyz")
	if err != nil {
		t.Fatal(err)
	}
	if view.LentKey == nil || view.LentKey.Type != "key" || view.LentKey.Last4 != "wxyz" {
		t.Fatalf("the client view must show the lent key, got %+v", view.LentKey)
	}
	had, err := m.RevokeAPIKey(client.ClientID)
	if err != nil || !had {
		t.Fatalf("RevokeAPIKey = (%v, %v), want (true, nil)", had, err)
	}
}

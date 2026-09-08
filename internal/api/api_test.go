package api

import (
	"errors"
	"testing"
)

func TestAdminLifecycle(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if store.HasAdmin() {
		t.Fatal("a fresh store has an admin secret")
	}
	secret, err := store.CreateAdmin()
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	if !store.HasAdmin() {
		t.Fatal("the store has no admin secret after create")
	}
	if !store.VerifyAdmin(secret) {
		t.Fatal("the admin secret does not verify")
	}
	if store.VerifyAdmin("wrong") {
		t.Fatal("a wrong secret verified")
	}
	if _, err := store.CreateAdmin(); !errors.Is(err, ErrAdminExists) {
		t.Fatalf("second create: want ErrAdminExists, got %v", err)
	}

	next, err := store.RotateAdmin()
	if err != nil {
		t.Fatalf("rotate admin: %v", err)
	}
	if store.VerifyAdmin(secret) {
		t.Fatal("the old secret still verifies after rotate")
	}
	if !store.VerifyAdmin(next) {
		t.Fatal("the new secret does not verify after rotate")
	}
	if err := store.RevokeAdmin(); err != nil {
		t.Fatalf("revoke admin: %v", err)
	}
	if store.HasAdmin() {
		t.Fatal("the store still has an admin secret after revoke")
	}
}

func TestClientLifecycle(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, _, err := store.CreateClient("  "); !errors.Is(err, ErrNoName) {
		t.Fatalf("empty name: want ErrNoName, got %v", err)
	}

	client, secret, err := store.CreateClient("bruno")
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	if client.ClientID == "" || secret == "" {
		t.Fatal("create client returned an empty id or secret")
	}

	if _, err := store.VerifyClient(client.ClientID, secret); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if _, err := store.VerifyClient(client.ClientID, "wrong"); !errors.Is(err, ErrBadSecret) {
		t.Fatalf("wrong secret: want ErrBadSecret, got %v", err)
	}
	if _, err := store.VerifyClient("nope", secret); !errors.Is(err, ErrUnknownClient) {
		t.Fatalf("unknown client: want ErrUnknownClient, got %v", err)
	}

	disabled := true
	if _, err := store.UpdateClient(client.ClientID, nil, &disabled); err != nil {
		t.Fatalf("disable: %v", err)
	}
	if _, err := store.VerifyClient(client.ClientID, secret); !errors.Is(err, ErrClientDisabled) {
		t.Fatalf("disabled client: want ErrClientDisabled, got %v", err)
	}

	enabled := false
	if _, err := store.UpdateClient(client.ClientID, nil, &enabled); err != nil {
		t.Fatalf("enable: %v", err)
	}
	next, err := store.RotateClient(client.ClientID)
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if _, err := store.VerifyClient(client.ClientID, secret); !errors.Is(err, ErrBadSecret) {
		t.Fatal("the old secret still verifies after rotate")
	}
	if _, err := store.VerifyClient(client.ClientID, next); err != nil {
		t.Fatalf("the new secret does not verify after rotate: %v", err)
	}

	if err := store.RevokeClient(client.ClientID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, err := store.VerifyClient(client.ClientID, next); !errors.Is(err, ErrUnknownClient) {
		t.Fatalf("revoked client: want ErrUnknownClient, got %v", err)
	}
}

func TestStorePersists(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	adminSecret, err := store.CreateAdmin()
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	client, clientSecret, err := store.CreateClient("bruno")
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	reopened, err := Open(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if !reopened.VerifyAdmin(adminSecret) {
		t.Fatal("the admin secret does not verify after reopen")
	}
	if _, err := reopened.VerifyClient(client.ClientID, clientSecret); err != nil {
		t.Fatalf("the client does not verify after reopen: %v", err)
	}
	if len(reopened.ListClients()) != 1 {
		t.Fatalf("want 1 client after reopen, got %d", len(reopened.ListClients()))
	}
}

func TestListClientsSorted(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	for _, name := range []string{"zed", "amy", "bob"} {
		if _, _, err := store.CreateClient(name); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
	}
	list := store.ListClients()
	if len(list) != 3 || list[0].Name != "amy" || list[1].Name != "bob" || list[2].Name != "zed" {
		t.Fatalf("clients are not sorted by name: %v", list)
	}
}

func TestUpdateClientErrors(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	client, _, err := store.CreateClient("bruno")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	blank := "  "
	if _, err := store.UpdateClient(client.ClientID, &blank, nil); !errors.Is(err, ErrNoName) {
		t.Fatalf("empty rename: want ErrNoName, got %v", err)
	}
	if _, err := store.UpdateClient("nope", nil, nil); !errors.Is(err, ErrUnknownClient) {
		t.Fatalf("unknown client: want ErrUnknownClient, got %v", err)
	}
}

func TestHashRoundTrip(t *testing.T) {
	h, err := hashSecret("a-secret")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if h.Hash == "" || h.Salt == "" {
		t.Fatal("the hash or the salt is empty")
	}
	if !h.verify("a-secret") {
		t.Fatal("the secret does not verify against its hash")
	}
	if h.verify("another-secret") {
		t.Fatal("a wrong secret verified against the hash")
	}
}

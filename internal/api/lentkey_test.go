package api

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestSetLentKeyStoresMetadataNotValue(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	client, _, err := store.CreateClient("laptop")
	if err != nil {
		t.Fatal(err)
	}

	const value = "sk-ant-oat01-secret-tail9999"
	view, err := store.SetLentKey(client.ClientID, "token", value)
	if err != nil {
		t.Fatal(err)
	}
	if view.LentKey == nil || view.LentKey.Type != "token" {
		t.Fatalf("the view must carry the lent-key type, got %+v", view.LentKey)
	}
	if view.LentKey.Last4 != "9999" {
		t.Errorf("last4 = %q, want 9999", view.LentKey.Last4)
	}
	if view.LentKey.IssuedAt.IsZero() {
		t.Error("issued_at must be set")
	}

	data, err := os.ReadFile(clientPath(apiDir(dir), client.ClientID))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), value) {
		t.Error("the credential value must never be written to disk")
	}
	if !strings.Contains(string(data), "\"last4\": \"9999\"") {
		t.Errorf("the record must store last4, got: %s", data)
	}
}

func TestSetLentKeyByName(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.CreateClient("studio"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetLentKey("studio", "key", "sk-ant-api03-abcd"); err != nil {
		t.Fatalf("a client must be found by name: %v", err)
	}
	if _, err := store.SetLentKey("missing", "key", "x"); !errors.Is(err, ErrUnknownClient) {
		t.Errorf("an unknown client must error, got %v", err)
	}
}

func TestClearLentKey(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	client, _, err := store.CreateClient("laptop")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetLentKey(client.ClientID, "token", "sk-ant-oat01-zzzz"); err != nil {
		t.Fatal(err)
	}
	view, had, err := store.ClearLentKey(client.ClientID)
	if err != nil {
		t.Fatal(err)
	}
	if !had {
		t.Error("clear must report the key was there")
	}
	if view.LentKey != nil {
		t.Error("the view must no longer carry a lent key")
	}
	_, had, err = store.ClearLentKey(client.ClientID)
	if err != nil {
		t.Fatal(err)
	}
	if had {
		t.Error("a second clear must report nothing to do")
	}
}

func TestLentKeySurvivesReopen(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	client, _, err := store.CreateClient("laptop")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetLentKey(client.ClientID, "token", "sk-ant-oat01-tail"); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range reopened.ListClients() {
		if c.ClientID == client.ClientID {
			if c.LentKey == nil || c.LentKey.Type != "token" {
				t.Fatalf("the lent key must survive a reopen, got %+v", c.LentKey)
			}
			return
		}
	}
	t.Fatal("the client must be there after a reopen")
}

package api

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestShareLifecycle(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, _, err := store.CreateShare("  ", time.Hour); !errors.Is(err, ErrNoSession) {
		t.Fatalf("empty session: want ErrNoSession, got %v", err)
	}

	share, token, err := store.CreateShare("brave-otter", time.Hour)
	if err != nil {
		t.Fatalf("create share: %v", err)
	}
	if share.Session != "brave-otter" || share.Scope != "view" {
		t.Fatalf("share view: %+v", share)
	}
	id, secret, ok := strings.Cut(token, ".")
	if !ok || id != share.ID {
		t.Fatalf("token %q does not carry the id %q", token, share.ID)
	}

	got, err := store.VerifyShare(id, secret)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if got.Session != "brave-otter" {
		t.Fatalf("verify session: %q", got.Session)
	}
	if _, err := store.VerifyShare(id, "wrong"); !errors.Is(err, ErrBadSecret) {
		t.Fatalf("wrong secret: want ErrBadSecret, got %v", err)
	}
	if _, err := store.VerifyShare("shr_missing", secret); !errors.Is(err, ErrUnknownShare) {
		t.Fatalf("unknown share: want ErrUnknownShare, got %v", err)
	}
	if !store.ShareActive(id) {
		t.Fatal("a fresh share is not active")
	}

	if err := store.RevokeShare(id); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if store.ShareActive(id) {
		t.Fatal("a revoked share is still active")
	}
	if _, err := store.VerifyShare(id, secret); !errors.Is(err, ErrUnknownShare) {
		t.Fatalf("verify after revoke: want ErrUnknownShare, got %v", err)
	}
	if err := store.RevokeShare(id); !errors.Is(err, ErrUnknownShare) {
		t.Fatalf("second revoke: want ErrUnknownShare, got %v", err)
	}
}

func TestShareExpiry(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	share, token, err := store.CreateShare("brave-otter", -time.Second)
	if err != nil {
		t.Fatalf("create share: %v", err)
	}
	if !share.ExpiresAt.IsZero() {
		t.Fatal("a non-positive ttl set an expiry")
	}

	// A share with an expiry in the past fails the verify and reads inactive.
	past, tokenPast, err := store.CreateShare("brave-otter", time.Hour)
	if err != nil {
		t.Fatalf("create share: %v", err)
	}
	store.mu.Lock()
	store.shares[past.ID].ExpiresAt = time.Now().Add(-time.Hour)
	store.mu.Unlock()
	idPast, secretPast, _ := strings.Cut(tokenPast, ".")
	if _, err := store.VerifyShare(idPast, secretPast); !errors.Is(err, ErrShareExpired) {
		t.Fatalf("expired share: want ErrShareExpired, got %v", err)
	}
	if store.ShareActive(idPast) {
		t.Fatal("an expired share reads active")
	}

	// The no-expiry share still verifies.
	id, secret, _ := strings.Cut(token, ".")
	if _, err := store.VerifyShare(id, secret); err != nil {
		t.Fatalf("no-expiry verify: %v", err)
	}
}

func TestShareListAndReload(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, _, err := store.CreateShare("one", time.Hour); err != nil {
		t.Fatalf("create one: %v", err)
	}
	if _, _, err := store.CreateShare("two", time.Hour); err != nil {
		t.Fatalf("create two: %v", err)
	}
	if got := len(store.ListShares()); got != 2 {
		t.Fatalf("list: want 2, got %d", got)
	}

	reopened, err := Open(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	list := reopened.ListShares()
	if len(list) != 2 {
		t.Fatalf("reload: want 2 shares, got %d", len(list))
	}
	for _, sh := range list {
		if sh.Session == "" || sh.ID == "" {
			t.Fatalf("reloaded share missing fields: %+v", sh)
		}
	}
}

package api

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// shareRecord is one read-only share on disk. It opens one session to whoever
// holds the token, until it expires or the host revokes it. The secret hash is
// inline, so one file holds one share. See docs/peers.md.
type shareRecord struct {
	ID        string    `json:"id"`
	Session   string    `json:"session"`
	Secret    hashed    `json:"secret_hash"`
	Scope     string    `json:"scope"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// Share is the public view of one share. It never holds the secret.
type Share struct {
	ID        string    `json:"id"`
	Session   string    `json:"session"`
	Scope     string    `json:"scope"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

func (r shareRecord) view() Share {
	return Share{
		ID:        r.ID,
		Session:   r.Session,
		Scope:     r.Scope,
		CreatedAt: r.CreatedAt,
		ExpiresAt: r.ExpiresAt,
	}
}

// expired reports whether the share has an expiry that is in the past.
func (r shareRecord) expired(now time.Time) bool {
	return !r.ExpiresAt.IsZero() && now.After(r.ExpiresAt)
}

// CreateShare mints a read-only share for a session. A positive ttl sets the
// expiry; a zero or negative ttl leaves the share with no expiry. It returns the
// share and the token `<id>.<secret>`, and the token is shown only here.
func (s *Store) CreateShare(session string, ttl time.Duration) (Share, string, error) {
	session = strings.TrimSpace(session)
	if session == "" {
		return Share{}, "", ErrNoSession
	}
	id, err := newSecret(6)
	if err != nil {
		return Share{}, "", err
	}
	id = "shr_" + id
	secret, hash, err := makeSecret(32)
	if err != nil {
		return Share{}, "", err
	}
	record := &shareRecord{
		ID:        id,
		Session:   session,
		Secret:    hash,
		Scope:     "view",
		CreatedAt: time.Now(),
	}
	if ttl > 0 {
		record.ExpiresAt = record.CreatedAt.Add(ttl)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.shares[id] = record
	if err := s.writeShare(record); err != nil {
		delete(s.shares, id)
		return Share{}, "", err
	}
	return record.view(), id + "." + secret, nil
}

// VerifyShare reports the share the token opens, or an error. It fails for an
// unknown share, a wrong secret, or an expired share. The caller reads the
// session from the returned share.
func (s *Store) VerifyShare(id, secret string) (Share, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.shares[id]
	if !ok {
		return Share{}, fmt.Errorf("%w: %s", ErrUnknownShare, id)
	}
	if !record.Secret.verify(secret) {
		return Share{}, ErrBadSecret
	}
	if record.expired(time.Now()) {
		return Share{}, fmt.Errorf("%w: %s", ErrShareExpired, id)
	}
	return record.view(), nil
}

// ShareActive reports whether a share is still there and not expired, without
// the secret. A live stream re-checks it, so a revoked or expired share ends the
// stream. See docs/peers.md.
func (s *Store) ShareActive(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.shares[id]
	return ok && !record.expired(time.Now())
}

// RevokeShare removes a share from memory and from disk, so its token stops
// working at once.
func (s *Store) RevokeShare(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.shares[id]; !ok {
		return fmt.Errorf("%w: %s", ErrUnknownShare, id)
	}
	delete(s.shares, id)
	if err := os.Remove(sharePath(s.dir, id)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ListShares gives the public view of every share, in create order.
func (s *Store) ListShares() []Share {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Share, 0, len(s.shares))
	for _, record := range s.shares {
		out = append(out, record.view())
	}
	sortShares(out)
	return out
}

func (s *Store) writeShare(record *shareRecord) error {
	if err := os.MkdirAll(sharesDir(s.dir), 0o700); err != nil {
		return err
	}
	return writeJSON(sharePath(s.dir, record.ID), record)
}

func sortShares(list []Share) {
	for i := 1; i < len(list); i++ {
		for j := i; j > 0 && list[j-1].CreatedAt.After(list[j].CreatedAt); j-- {
			list[j-1], list[j] = list[j], list[j-1]
		}
	}
}

package api

import (
	"os"
	"time"
)

// admin is the admin secret record on disk.
type admin struct {
	hashed
	CreatedAt time.Time `json:"created_at"`
	RotatedAt time.Time `json:"rotated_at,omitempty"`
}

// HasAdmin reports whether an admin secret exists, so the API is on.
func (s *Store) HasAdmin() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.admin != nil
}

// CreateAdmin makes the admin secret. It fails when one already exists, because
// rotate is the way to replace it. It returns the plaintext once.
func (s *Store) CreateAdmin() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.admin != nil {
		return "", ErrAdminExists
	}
	secret, hash, err := makeSecret(32)
	if err != nil {
		return "", err
	}
	s.admin = &admin{hashed: hash, CreatedAt: time.Now()}
	if err := s.writeAdmin(); err != nil {
		s.admin = nil
		return "", err
	}
	return secret, nil
}

// RotateAdmin replaces the admin secret. It returns the new plaintext once.
func (s *Store) RotateAdmin() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.admin == nil {
		return "", ErrNoAdmin
	}
	secret, hash, err := makeSecret(32)
	if err != nil {
		return "", err
	}
	prev := *s.admin
	s.admin.hashed = hash
	s.admin.RotatedAt = time.Now()
	if err := s.writeAdmin(); err != nil {
		*s.admin = prev
		return "", err
	}
	return secret, nil
}

// RevokeAdmin removes the admin secret, so the API is off until a later create.
func (s *Store) RevokeAdmin() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.admin == nil {
		return ErrNoAdmin
	}
	s.admin = nil
	if err := os.Remove(adminPath(s.dir)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// VerifyAdmin reports whether secret is the admin secret.
func (s *Store) VerifyAdmin(secret string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.admin != nil && s.admin.verify(secret)
}

func (s *Store) writeAdmin() error {
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}
	return writeJSON(adminPath(s.dir), s.admin)
}

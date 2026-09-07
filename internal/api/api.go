// Package api holds the credentials that let a program outside the multiplexer
// reach the session API: one admin secret, and one record for each client. It
// stores only a hash of each secret, and it persists the records under the
// state directory. See docs/mcp/api.md.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	ErrNoAdmin        = errors.New("api: no admin secret")
	ErrAdminExists    = errors.New("api: an admin secret already exists")
	ErrUnknownClient  = errors.New("api: unknown client")
	ErrClientDisabled = errors.New("api: the client is disabled")
	ErrBadSecret      = errors.New("api: the secret does not match")
	ErrNoName         = errors.New("api: a client needs a name")
)

// Store holds the admin secret and the clients, and it keeps them on disk. Every
// method is safe for concurrent use.
type Store struct {
	dir string

	mu      sync.Mutex
	admin   *admin
	clients map[string]*clientRecord
}

// Open reads the store under root, and it makes the empty maps. It reads the
// admin secret and every client file that is there.
func Open(root string) (*Store, error) {
	s := &Store{dir: apiDir(root), clients: map[string]*clientRecord{}}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func apiDir(root string) string    { return filepath.Join(root, "api") }
func adminPath(dir string) string  { return filepath.Join(dir, "admin.json") }
func clientsDir(dir string) string { return filepath.Join(dir, "clients") }
func clientPath(dir, id string) string {
	return filepath.Join(clientsDir(dir), id+".json")
}

func (s *Store) load() error {
	if data, err := os.ReadFile(adminPath(s.dir)); err == nil {
		var a admin
		if err := json.Unmarshal(data, &a); err != nil {
			return fmt.Errorf("api: read admin: %w", err)
		}
		s.admin = &a
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("api: read admin: %w", err)
	}

	entries, err := os.ReadDir(clientsDir(s.dir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("api: read clients: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		data, err := os.ReadFile(clientPath(s.dir, id))
		if err != nil {
			continue
		}
		var c clientRecord
		if err := json.Unmarshal(data, &c); err != nil {
			continue
		}
		if c.ClientID == "" {
			c.ClientID = id
		}
		record := c
		s.clients[record.ClientID] = &record
	}
	return nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

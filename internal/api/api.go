// Package api holds the credentials that let a program outside the multiplexer
// reach the session API: one admin secret, and one record for each client. It
// stores only a hash of each secret, and it persists the records under the
// state directory. See docs/mcp/api.md.
package api

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	ErrNoAdmin        = errors.New("api: no admin secret")
	ErrAdminExists    = errors.New("api: an admin secret already exists")
	ErrUnknownClient  = errors.New("api: unknown client")
	ErrClientDisabled = errors.New("api: the client is disabled")
	ErrBadSecret      = errors.New("api: the secret does not match")
	ErrNoName         = errors.New("api: a client needs a name")
)

const algo = "sha256"

// hashed holds a salted hash of one secret. The plaintext is shown once, at
// create and at rotate, and it is never stored.
type hashed struct {
	Algo string `json:"algo"`
	Salt string `json:"salt"`
	Hash string `json:"hash"`
}

func hashSecret(secret string) (hashed, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return hashed{}, fmt.Errorf("api: salt: %w", err)
	}
	sum := sha256.Sum256(append(salt, secret...))
	return hashed{Algo: algo, Salt: hex.EncodeToString(salt), Hash: hex.EncodeToString(sum[:])}, nil
}

func (h hashed) verify(secret string) bool {
	salt, err := hex.DecodeString(h.Salt)
	if err != nil {
		return false
	}
	sum := sha256.Sum256(append(salt, secret...))
	want, err := hex.DecodeString(h.Hash)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(sum[:], want) == 1
}

func newSecret(n int) (string, error) {
	raw := make([]byte, n)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("api: secret: %w", err)
	}
	return hex.EncodeToString(raw), nil
}

// admin is the admin secret record on disk.
type admin struct {
	hashed
	CreatedAt time.Time `json:"created_at"`
	RotatedAt time.Time `json:"rotated_at,omitempty"`
}

// clientRecord is one client record on disk. The secret hash is inline, so one
// file holds one client.
type clientRecord struct {
	ClientID  string    `json:"client_id"`
	Name      string    `json:"name"`
	Secret    hashed    `json:"secret_hash"`
	CreatedAt time.Time `json:"created_at"`
	RotatedAt time.Time `json:"rotated_at,omitempty"`
	Disabled  bool      `json:"disabled,omitempty"`
}

// Client is the public view of one client. It never holds the secret.
type Client struct {
	ClientID  string    `json:"client_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	RotatedAt time.Time `json:"rotated_at,omitempty"`
	Disabled  bool      `json:"disabled"`
}

func (c clientRecord) view() Client {
	return Client{
		ClientID:  c.ClientID,
		Name:      c.Name,
		CreatedAt: c.CreatedAt,
		RotatedAt: c.RotatedAt,
		Disabled:  c.Disabled,
	}
}

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

// CreateClient makes a client with a name. It returns the client and its secret,
// and the secret is shown only here.
func (s *Store) CreateClient(name string) (Client, string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Client{}, "", ErrNoName
	}
	id, err := newSecret(16)
	if err != nil {
		return Client{}, "", err
	}
	secret, hash, err := makeSecret(32)
	if err != nil {
		return Client{}, "", err
	}
	record := &clientRecord{ClientID: id, Name: name, Secret: hash, CreatedAt: time.Now()}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[id] = record
	if err := s.writeClient(record); err != nil {
		delete(s.clients, id)
		return Client{}, "", err
	}
	return record.view(), secret, nil
}

// UpdateClient changes the name, the disabled flag, or both. A nil field stays
// as it is.
func (s *Store) UpdateClient(id string, name *string, disabled *bool) (Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.clients[id]
	if !ok {
		return Client{}, fmt.Errorf("%w: %s", ErrUnknownClient, id)
	}
	prev := *record
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return Client{}, ErrNoName
		}
		record.Name = trimmed
	}
	if disabled != nil {
		record.Disabled = *disabled
	}
	if err := s.writeClient(record); err != nil {
		*record = prev
		return Client{}, err
	}
	return record.view(), nil
}

// RotateClient issues a new secret for a client, and returns it once. The client
// id does not change.
func (s *Store) RotateClient(id string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.clients[id]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrUnknownClient, id)
	}
	secret, hash, err := makeSecret(32)
	if err != nil {
		return "", err
	}
	prev := *record
	record.Secret = hash
	record.RotatedAt = time.Now()
	if err := s.writeClient(record); err != nil {
		*record = prev
		return "", err
	}
	return secret, nil
}

// RevokeClient removes a client from memory and from disk.
func (s *Store) RevokeClient(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.clients[id]; !ok {
		return fmt.Errorf("%w: %s", ErrUnknownClient, id)
	}
	delete(s.clients, id)
	if err := os.Remove(clientPath(s.dir, id)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ListClients gives the public view of every client, in name order.
func (s *Store) ListClients() []Client {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Client, 0, len(s.clients))
	for _, record := range s.clients {
		out = append(out, record.view())
	}
	sortClients(out)
	return out
}

// VerifyClient reports the client that owns the pair, or an error. It fails for
// an unknown client, a wrong secret, or a disabled client.
func (s *Store) VerifyClient(id, secret string) (Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.clients[id]
	if !ok {
		return Client{}, fmt.Errorf("%w: %s", ErrUnknownClient, id)
	}
	if !record.Secret.verify(secret) {
		return Client{}, ErrBadSecret
	}
	if record.Disabled {
		return Client{}, ErrClientDisabled
	}
	return record.view(), nil
}

// Client reads one client by id, so a caller can name it in the interface.
func (s *Store) Client(id string) (Client, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.clients[id]
	if !ok {
		return Client{}, false
	}
	return record.view(), true
}

func (s *Store) writeAdmin() error {
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}
	return writeJSON(adminPath(s.dir), s.admin)
}

func (s *Store) writeClient(record *clientRecord) error {
	if err := os.MkdirAll(clientsDir(s.dir), 0o700); err != nil {
		return err
	}
	return writeJSON(clientPath(s.dir, record.ClientID), record)
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func makeSecret(n int) (string, hashed, error) {
	secret, err := newSecret(n)
	if err != nil {
		return "", hashed{}, err
	}
	hash, err := hashSecret(secret)
	if err != nil {
		return "", hashed{}, err
	}
	return secret, hash, nil
}

func sortClients(list []Client) {
	for i := 1; i < len(list); i++ {
		for j := i; j > 0 && list[j-1].Name > list[j].Name; j-- {
			list[j-1], list[j] = list[j], list[j-1]
		}
	}
}

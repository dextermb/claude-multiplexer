package api

import (
	"fmt"
	"os"
	"strings"
	"time"
)

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

func (s *Store) writeClient(record *clientRecord) error {
	if err := os.MkdirAll(clientsDir(s.dir), 0o700); err != nil {
		return err
	}
	return writeJSON(clientPath(s.dir, record.ClientID), record)
}

func sortClients(list []Client) {
	for i := 1; i < len(list); i++ {
		for j := i; j > 0 && list[j-1].Name > list[j].Name; j-- {
			list[j-1], list[j] = list[j], list[j-1]
		}
	}
}

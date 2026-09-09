package api

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// clientRecord is one client record on disk. The secret hash is inline, so one
// file holds one client. LentKey holds only the metadata of a lent Claude
// credential; the value is never written. See docs/peers/hoisted.md.
type clientRecord struct {
	ClientID  string    `json:"client_id"`
	Name      string    `json:"name"`
	Secret    hashed    `json:"secret_hash"`
	CreatedAt time.Time `json:"created_at"`
	RotatedAt time.Time `json:"rotated_at,omitempty"`
	Disabled  bool      `json:"disabled,omitempty"`
	LentKey   *LentKey  `json:"lent_key,omitempty"`
}

// LentKey is the metadata of a Claude credential a host lent to a client, so the
// client may run a hoisted session. It never holds the credential value. See
// docs/peers/hoisted.md.
type LentKey struct {
	Type     string    `json:"type"`
	Last4    string    `json:"last4"`
	IssuedAt time.Time `json:"issued_at"`
}

// Client is the public view of one client. It never holds the secret.
type Client struct {
	ClientID  string    `json:"client_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	RotatedAt time.Time `json:"rotated_at,omitempty"`
	Disabled  bool      `json:"disabled"`
	LentKey   *LentKey  `json:"lent_key,omitempty"`
}

func (c clientRecord) view() Client {
	view := Client{
		ClientID:  c.ClientID,
		Name:      c.Name,
		CreatedAt: c.CreatedAt,
		RotatedAt: c.RotatedAt,
		Disabled:  c.Disabled,
	}
	if c.LentKey != nil {
		lk := *c.LentKey
		view.LentKey = &lk
	}
	return view
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

// SetLentKey records the metadata of a Claude credential lent to a client, found
// by client id or by name. The value only computes the last four characters; it
// is never stored. See docs/peers/hoisted.md.
func (s *Store) SetLentKey(nameOrID, ktype, value string) (Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.find(nameOrID)
	if !ok {
		return Client{}, fmt.Errorf("%w: %s", ErrUnknownClient, nameOrID)
	}
	prev := record.LentKey
	record.LentKey = &LentKey{Type: ktype, Last4: last4(value), IssuedAt: time.Now()}
	if err := s.writeClient(record); err != nil {
		record.LentKey = prev
		return Client{}, err
	}
	return record.view(), nil
}

// ClearLentKey removes the lent-key metadata from a client, found by id or name,
// and reports whether it had one. It does not stop the credential at Anthropic,
// and it does not retract a copy the borrower already holds.
func (s *Store) ClearLentKey(nameOrID string) (Client, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.find(nameOrID)
	if !ok {
		return Client{}, false, fmt.Errorf("%w: %s", ErrUnknownClient, nameOrID)
	}
	if record.LentKey == nil {
		return record.view(), false, nil
	}
	prev := record.LentKey
	record.LentKey = nil
	if err := s.writeClient(record); err != nil {
		record.LentKey = prev
		return Client{}, false, err
	}
	return record.view(), true, nil
}

// find looks up a client by its id first, then by an exact name match.
func (s *Store) find(nameOrID string) (*clientRecord, bool) {
	if record, ok := s.clients[nameOrID]; ok {
		return record, true
	}
	for _, record := range s.clients {
		if record.Name == nameOrID {
			return record, true
		}
	}
	return nil, false
}

// last4 gives the last four characters of a value, or the whole value when it is
// shorter, so a list can show which credential a client holds without the value.
func last4(value string) string {
	r := []rune(value)
	if len(r) <= 4 {
		return string(r)
	}
	return string(r[len(r)-4:])
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

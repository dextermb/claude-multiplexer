package api

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
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

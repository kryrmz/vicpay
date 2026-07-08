// Package pii encrypts personally identifiable information at the application
// layer with AES-256-GCM and provides a keyed HMAC "blind index" for equality
// lookups on encrypted columns. Doing this in Go (rather than via a Postgres
// session GUC) keeps confidentiality independent of the connection, so it is
// safe under a transactional pooler like PgBouncer.
//
// Two independent subkeys are derived from the master key with HKDF: one for
// encryption and one for the blind index. This avoids reusing a single key
// across cryptographic domains, an audit finding in the predecessor project.
package pii

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"

	"golang.org/x/crypto/hkdf"
)

// ErrCiphertextTooShort is returned when a ciphertext is malformed.
var ErrCiphertextTooShort = errors.New("pii: ciphertext too short")

// Cipher encrypts/decrypts PII and computes blind indexes.
type Cipher struct {
	aead    cipher.AEAD
	indexer []byte // HMAC key for the blind index
}

// NewCipher derives subkeys from a >=32-byte master key.
func NewCipher(masterKey []byte) (*Cipher, error) {
	if len(masterKey) < 32 {
		return nil, errors.New("pii: master key must be at least 32 bytes")
	}
	encKey, err := derive(masterKey, "vicpay/pii/encryption")
	if err != nil {
		return nil, err
	}
	idxKey, err := derive(masterKey, "vicpay/pii/blind-index")
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cipher{aead: aead, indexer: idxKey}, nil
}

// Encrypt returns nonce||ciphertext for the given plaintext.
func (c *Cipher) Encrypt(plaintext string) ([]byte, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return c.aead.Seal(nonce, nonce, []byte(plaintext), nil), nil
}

// Decrypt reverses Encrypt.
func (c *Cipher) Decrypt(blob []byte) (string, error) {
	ns := c.aead.NonceSize()
	if len(blob) < ns {
		return "", ErrCiphertextTooShort
	}
	nonce, ciphertext := blob[:ns], blob[ns:]
	plaintext, err := c.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// BlindIndex returns a deterministic HMAC-SHA256 of value, for indexed equality
// lookups (e.g. finding a user by phone) without decrypting stored ciphertext.
func (c *Cipher) BlindIndex(value string) []byte {
	mac := hmac.New(sha256.New, c.indexer)
	mac.Write([]byte(value))
	return mac.Sum(nil)
}

func derive(master []byte, info string) ([]byte, error) {
	r := hkdf.New(sha256.New, master, nil, []byte(info))
	key := make([]byte, 32)
	if _, err := io.ReadFull(r, key); err != nil {
		return nil, err
	}
	return key, nil
}

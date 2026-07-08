// Package hash implements password hashing with Argon2id using the OWASP 2024
// recommended parameters, plus constant-time verification and a dummy-verify
// helper to blunt user-enumeration timing attacks.
package hash

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Params holds the Argon2id cost parameters. Defaults follow OWASP 2024.
type Params struct {
	Memory      uint32 // KiB
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// DefaultParams is the OWASP 2024 baseline: 128 MiB, t=4, p=2.
var DefaultParams = Params{
	Memory:      128 * 1024,
	Iterations:  4,
	Parallelism: 2,
	SaltLength:  16,
	KeyLength:   32,
}

// ErrInvalidHash is returned when a stored encoded hash cannot be parsed.
var ErrInvalidHash = errors.New("hash: invalid encoded hash")

// ErrIncompatibleVersion is returned when the Argon2 version differs.
var ErrIncompatibleVersion = errors.New("hash: incompatible argon2 version")

// Hash derives an encoded PHC-format Argon2id hash for the given password.
func Hash(password string) (string, error) {
	return HashWithParams(password, DefaultParams)
}

// HashWithParams is Hash with explicit parameters (useful in tests).
func HashWithParams(password string, p Params) (string, error) {
	salt := make([]byte, p.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("hash: read salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLength)
	b64 := base64.RawStdEncoding
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.Memory, p.Iterations, p.Parallelism,
		b64.EncodeToString(salt), b64.EncodeToString(key),
	), nil
}

// Verify reports whether password matches the encoded hash in constant time.
func Verify(password, encoded string) (bool, error) {
	p, salt, key, err := decode(encoded)
	if err != nil {
		return false, err
	}
	if len(key) == 0 || len(key) > 4096 {
		return false, ErrInvalidHash
	}
	keyLen := uint32(len(key)) // #nosec G115 -- len(key) is bounded to [1,4096] above
	other := argon2.IDKey([]byte(password), salt, p.Iterations, p.Memory, p.Parallelism, keyLen)
	return subtle.ConstantTimeCompare(key, other) == 1, nil
}

// DummyVerify performs a hash computation and discards the result. Call it on
// the login path when the user is not found so that the response time does not
// leak whether an account exists.
func DummyVerify(password string) {
	p := DefaultParams
	salt := make([]byte, p.SaltLength)
	_ = argon2.IDKey([]byte(password), salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLength)
}

func decode(encoded string) (Params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return Params{}, nil, nil, ErrInvalidHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return Params{}, nil, nil, ErrInvalidHash
	}
	if version != argon2.Version {
		return Params{}, nil, nil, ErrIncompatibleVersion
	}
	var p Params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.Memory, &p.Iterations, &p.Parallelism); err != nil {
		return Params{}, nil, nil, ErrInvalidHash
	}
	b64 := base64.RawStdEncoding
	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return Params{}, nil, nil, ErrInvalidHash
	}
	key, err := b64.DecodeString(parts[5])
	if err != nil {
		return Params{}, nil, nil, ErrInvalidHash
	}
	return p, salt, key, nil
}

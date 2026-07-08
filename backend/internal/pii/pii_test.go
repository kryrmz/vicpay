package pii

import (
	"bytes"
	"testing"
)

var testKey = []byte("a-32-byte-master-key-for-testing!")

func TestEncryptDecryptRoundTrip(t *testing.T) {
	c, err := NewCipher(testKey)
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	blob, err := c.Encrypt("+50688881234")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	got, err := c.Decrypt(blob)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if got != "+50688881234" {
		t.Fatalf("round trip mismatch: %q", got)
	}
}

func TestEncryptIsNondeterministic(t *testing.T) {
	c, _ := NewCipher(testKey)
	a, _ := c.Encrypt("secret")
	b, _ := c.Encrypt("secret")
	if bytes.Equal(a, b) {
		t.Fatal("two encryptions of the same value must differ (random nonce)")
	}
}

func TestBlindIndexIsDeterministicAndKeyed(t *testing.T) {
	c, _ := NewCipher(testKey)
	if !bytes.Equal(c.BlindIndex("+50688881234"), c.BlindIndex("+50688881234")) {
		t.Fatal("blind index must be deterministic for lookups")
	}
	other, _ := NewCipher([]byte("a-different-32-byte-master-key!!!"))
	if bytes.Equal(c.BlindIndex("+50688881234"), other.BlindIndex("+50688881234")) {
		t.Fatal("blind index must depend on the key")
	}
}

func TestDecryptRejectsShortInput(t *testing.T) {
	c, _ := NewCipher(testKey)
	if _, err := c.Decrypt([]byte("x")); err != ErrCiphertextTooShort {
		t.Fatalf("expected ErrCiphertextTooShort, got %v", err)
	}
}

func TestNewCipherRejectsShortKey(t *testing.T) {
	if _, err := NewCipher([]byte("short")); err == nil {
		t.Fatal("expected error for short master key")
	}
}

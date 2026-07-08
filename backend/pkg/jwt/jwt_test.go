package jwt

import (
	"testing"
	"time"
)

func newTestManager() *Manager {
	return NewManager("test-secret-at-least-32-bytes-long!!", "vicpay-test", 15*time.Minute)
}

func TestAccessAndRefreshNotInterchangeable(t *testing.T) {
	m := newTestManager()

	access, err := m.IssueAccess("user-1", "jti-a")
	if err != nil {
		t.Fatalf("issue access: %v", err)
	}
	if _, err := m.Parse(access, Access); err != nil {
		t.Fatalf("access should parse as access: %v", err)
	}
	// An access token must NOT be accepted where a refresh token is expected.
	if _, err := m.Parse(access, Refresh); err != ErrWrongType {
		t.Fatalf("access accepted as refresh: %v", err)
	}

	refresh, err := m.IssueRefresh("user-1", "jti-r", "fam-1", "", 7*24*time.Hour)
	if err != nil {
		t.Fatalf("issue refresh: %v", err)
	}
	c, err := m.Parse(refresh, Refresh)
	if err != nil {
		t.Fatalf("refresh should parse: %v", err)
	}
	if c.FamilyID != "fam-1" {
		t.Fatalf("family id lost: %q", c.FamilyID)
	}
	if _, err := m.Parse(refresh, Access); err != ErrWrongType {
		t.Fatalf("refresh accepted as access: %v", err)
	}
}

func TestExpiredTokenRejected(t *testing.T) {
	m := NewManager("test-secret-at-least-32-bytes-long!!", "vicpay-test", -1*time.Minute)
	tok, _ := m.IssueAccess("user-1", "jti")
	if _, err := m.Parse(tok, Access); err == nil {
		t.Fatal("expired token must be rejected")
	}
}

func TestWrongSecretRejected(t *testing.T) {
	a := newTestManager()
	b := NewManager("a-completely-different-secret-value-32b", "vicpay-test", time.Minute)
	tok, _ := a.IssueAccess("user-1", "jti")
	if _, err := b.Parse(tok, Access); err == nil {
		t.Fatal("token signed with another secret must be rejected")
	}
}

func TestFingerprintStable(t *testing.T) {
	fp := Fingerprint("abc")
	if fp != Fingerprint("abc") {
		t.Fatal("fingerprint must be deterministic")
	}
	if fp == Fingerprint("abd") {
		t.Fatal("different tokens must fingerprint differently")
	}
}

package otp

import (
	"context"
	"testing"
	"time"
)

// memStore is an in-memory Store for tests.
type memStore struct{ items map[string]*Challenge }

func newMemStore() *memStore { return &memStore{items: map[string]*Challenge{}} }

func (m *memStore) Save(_ context.Context, c *Challenge) error {
	cp := *c
	m.items[c.ID] = &cp
	return nil
}

func (m *memStore) Active(_ context.Context, recipient string, purpose Purpose) (*Challenge, error) {
	var best *Challenge
	for _, c := range m.items {
		if c.Recipient == recipient && c.Purpose == purpose && c.ConsumedAt == nil {
			if best == nil || c.ExpiresAt.After(best.ExpiresAt) {
				best = c
			}
		}
	}
	return best, nil
}

func (m *memStore) IncrementAttempts(_ context.Context, id string) error {
	if c, ok := m.items[id]; ok {
		c.Attempts++
	}
	return nil
}

func (m *memStore) Consume(_ context.Context, id string, at time.Time) error {
	if c, ok := m.items[id]; ok {
		c.ConsumedAt = &at
	}
	return nil
}

// captureSender records the last code so the test can submit it. A real Sender
// would never expose the code.
type captureSender struct{ lastCode string }

func (s *captureSender) Send(_ context.Context, _ string, code string) error {
	s.lastCode = code
	return nil
}

func TestIssueAndVerifyHappyPath(t *testing.T) {
	store := newMemStore()
	sender := &captureSender{}
	svc := NewService(store, sender, DefaultConfig)
	ctx := context.Background()

	if _, err := svc.Issue(ctx, "chal-1", "+50688881234", PurposePhoneVerify); err != nil {
		t.Fatalf("issue: %v", err)
	}
	if len(sender.lastCode) != DefaultConfig.CodeLength {
		t.Fatalf("expected %d-digit code, got %q", DefaultConfig.CodeLength, sender.lastCode)
	}
	if err := svc.Verify(ctx, "+50688881234", PurposePhoneVerify, sender.lastCode); err != nil {
		t.Fatalf("verify: %v", err)
	}
	// Single-use: a second verify with the same code must fail.
	if err := svc.Verify(ctx, "+50688881234", PurposePhoneVerify, sender.lastCode); err != ErrNotFound {
		t.Fatalf("expected consumed challenge, got %v", err)
	}
}

func TestVerifyWrongCodeCountsAttempts(t *testing.T) {
	store := newMemStore()
	sender := &captureSender{}
	svc := NewService(store, sender, Config{CodeLength: 6, TTL: time.Minute, MaxAttempts: 2})
	ctx := context.Background()
	_, _ = svc.Issue(ctx, "chal-1", "+50688881234", PurposePhoneVerify)

	if err := svc.Verify(ctx, "+50688881234", PurposePhoneVerify, "000000"); err != ErrCodeInvalid {
		t.Fatalf("expected ErrCodeInvalid, got %v", err)
	}
	if err := svc.Verify(ctx, "+50688881234", PurposePhoneVerify, "000000"); err != ErrCodeInvalid {
		t.Fatalf("expected ErrCodeInvalid, got %v", err)
	}
	// Attempts exhausted, even the correct code is now refused.
	if err := svc.Verify(ctx, "+50688881234", PurposePhoneVerify, sender.lastCode); err != ErrTooMany {
		t.Fatalf("expected ErrTooMany, got %v", err)
	}
}

func TestVerifyExpired(t *testing.T) {
	store := newMemStore()
	sender := &captureSender{}
	svc := NewService(store, sender, Config{CodeLength: 6, TTL: time.Minute, MaxAttempts: 5})
	base := time.Now()
	svc.now = func() time.Time { return base }
	ctx := context.Background()
	_, _ = svc.Issue(ctx, "chal-1", "+50688881234", PurposePhoneVerify)

	svc.now = func() time.Time { return base.Add(2 * time.Minute) }
	if err := svc.Verify(ctx, "+50688881234", PurposePhoneVerify, sender.lastCode); err != ErrExpired {
		t.Fatalf("expected ErrExpired, got %v", err)
	}
}

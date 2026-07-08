package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/vicpay/backend/internal/auth"
	"github.com/vicpay/backend/internal/authstore"
	"github.com/vicpay/backend/internal/otp"
	"github.com/vicpay/backend/internal/pii"
	"github.com/vicpay/backend/internal/testutil"
	pkgjwt "github.com/vicpay/backend/pkg/jwt"
)

// captureSender records the last OTP code so the test can complete verification.
type captureSender struct{ code string }

func (s *captureSender) Send(_ context.Context, _ string, code string) error {
	s.code = code
	return nil
}

func newService(t *testing.T) (*auth.Service, *captureSender) {
	t.Helper()
	pool := testutil.SetupDB(t)
	cipher, err := pii.NewCipher([]byte("a-32-byte-master-key-for-testing!"))
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	jwtMgr := pkgjwt.NewManager("test-secret-at-least-32-bytes-long!!", "vicpay-test", 15*time.Minute)
	sender := &captureSender{}
	otpSvc := otp.NewService(authstore.NewOTPStore(pool), sender, otp.DefaultConfig)
	svc := auth.NewService(auth.NewRepository(pool), cipher, jwtMgr, otpSvc, 7*24*time.Hour, 30*time.Minute)
	return svc, sender
}

func TestRegisterVerifyLoginFlow(t *testing.T) {
	svc, sender := newService(t)
	ctx := context.Background()

	id, err := svc.Register(ctx, "+50688881234", "hunter2go1234")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if sender.code == "" {
		t.Fatal("register should have dispatched an OTP code")
	}

	// Login before verification is refused.
	if _, err := svc.Login(ctx, "+50688881234", "hunter2go1234"); err != auth.ErrPhoneNotVerified {
		t.Fatalf("expected ErrPhoneNotVerified, got %v", err)
	}

	// Wrong OTP is rejected.
	if _, err := svc.VerifyPhone(ctx, id, "000000"); err == nil {
		t.Fatal("verify with wrong code must fail")
	}

	session, err := svc.VerifyPhone(ctx, id, sender.code)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if session.AccessToken == "" || session.RefreshToken() == "" {
		t.Fatal("verify should mint a session with both tokens")
	}

	// Now login works.
	if _, err := svc.Login(ctx, "+50688881234", "hunter2go1234"); err != nil {
		t.Fatalf("login after verify: %v", err)
	}
	// Wrong password is rejected.
	if _, err := svc.Login(ctx, "+50688881234", "wrongpassword1"); err != auth.ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestDuplicatePhoneRejected(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()
	if _, err := svc.Register(ctx, "+50688887777", "hunter2go1234"); err != nil {
		t.Fatalf("first register: %v", err)
	}
	if _, err := svc.Register(ctx, "+50688887777", "hunter2go1234"); err != auth.ErrPhoneTaken {
		t.Fatalf("expected ErrPhoneTaken, got %v", err)
	}
}

func TestRefreshRotationDetectsReuse(t *testing.T) {
	svc, sender := newService(t)
	ctx := context.Background()

	id, _ := svc.Register(ctx, "+50688889999", "hunter2go1234")
	session, err := svc.VerifyPhone(ctx, id, sender.code)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	original := session.RefreshToken()

	// A first refresh rotates successfully.
	rotated, err := svc.Refresh(ctx, original)
	if err != nil {
		t.Fatalf("first refresh: %v", err)
	}
	if rotated.RefreshToken() == original {
		t.Fatal("refresh must issue a new token")
	}

	// Replaying the now-rotated token is detected as reuse.
	if _, err := svc.Refresh(ctx, original); err != auth.ErrTokenReuse {
		t.Fatalf("expected ErrTokenReuse on replay, got %v", err)
	}
	// The reuse must have revoked the whole family: the rotated token is dead too.
	if _, err := svc.Refresh(ctx, rotated.RefreshToken()); err == nil {
		t.Fatal("after reuse detection the whole family must be revoked")
	}
}

package auth

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/vicpay/backend/internal/otp"
	"github.com/vicpay/backend/internal/pii"
	"github.com/vicpay/backend/pkg/hash"
	pkgjwt "github.com/vicpay/backend/pkg/jwt"
	"github.com/vicpay/backend/pkg/validator"
)

// Service implements the authentication flows.
type Service struct {
	repo        *Repository
	cipher      *pii.Cipher
	jwt         *pkgjwt.Manager
	otp         *otp.Service
	refreshTTL  time.Duration
	idleTimeout time.Duration
	now         func() time.Time
}

// NewService wires the auth service.
func NewService(repo *Repository, cipher *pii.Cipher, jwtMgr *pkgjwt.Manager, otpSvc *otp.Service, refreshTTL, idleTimeout time.Duration) *Service {
	return &Service{
		repo: repo, cipher: cipher, jwt: jwtMgr, otp: otpSvc,
		refreshTTL: refreshTTL, idleTimeout: idleTimeout, now: time.Now,
	}
}

// Register creates a Level 0 user (unverified) and dispatches a phone OTP. It
// returns the new user id; no session is issued until the phone is verified.
func (s *Service) Register(ctx context.Context, phone, password string) (string, error) {
	phone = strings.TrimSpace(phone)
	if !validator.PhoneE164(phone) {
		return "", ErrInvalidPhone
	}
	if !validator.Password(password) {
		return "", ErrWeakPassword
	}
	idx := s.cipher.BlindIndex(phone)
	if existing, err := s.repo.FindByPhoneIndex(ctx, idx); err != nil {
		return "", err
	} else if existing != nil {
		return "", ErrPhoneTaken
	}
	cipherBlob, err := s.cipher.Encrypt(phone)
	if err != nil {
		return "", err
	}
	pwHash, err := hash.Hash(password)
	if err != nil {
		return "", err
	}
	id, err := s.repo.CreateUser(ctx, idx, cipherBlob, pwHash)
	if err != nil {
		return "", err
	}
	if _, err := s.otp.Issue(ctx, uuid.NewString(), phone, otp.PurposePhoneVerify); err != nil {
		return "", err
	}
	return id, nil
}

// ResendCode re-issues a phone-verification OTP for a pending user.
func (s *Service) ResendCode(ctx context.Context, userID string) error {
	u, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return err
	}
	if u == nil {
		return ErrUserNotFound
	}
	phone, err := s.cipher.Decrypt(u.phoneCipher)
	if err != nil {
		return err
	}
	_, err = s.otp.Issue(ctx, uuid.NewString(), phone, otp.PurposePhoneVerify)
	return err
}

// VerifyPhone checks the OTP for a pending user and, on success, marks the phone
// verified and issues the first session.
func (s *Service) VerifyPhone(ctx context.Context, userID, code string) (*Session, error) {
	u, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrUserNotFound
	}
	phone, err := s.cipher.Decrypt(u.phoneCipher)
	if err != nil {
		return nil, err
	}
	if err := s.otp.Verify(ctx, phone, otp.PurposePhoneVerify, code); err != nil {
		return nil, ErrInvalidToken
	}
	if err := s.repo.SetPhoneVerified(ctx, userID); err != nil {
		return nil, err
	}
	u.phoneVerified = true
	return s.mintSession(ctx, u)
}

// Login authenticates a verified user by phone and password.
func (s *Service) Login(ctx context.Context, phone, password string) (*Session, error) {
	idx := s.cipher.BlindIndex(strings.TrimSpace(phone))
	u, err := s.repo.FindByPhoneIndex(ctx, idx)
	if err != nil {
		return nil, err
	}
	if u == nil {
		// Spend comparable time to blunt account-enumeration timing attacks.
		hash.DummyVerify(password)
		return nil, ErrInvalidCredentials
	}
	ok, err := hash.Verify(password, u.passwordHash)
	if err != nil || !ok {
		return nil, ErrInvalidCredentials
	}
	if !u.phoneVerified {
		return nil, ErrPhoneNotVerified
	}
	return s.mintSession(ctx, u)
}

// Refresh rotates a refresh token: it detects reuse (revoking the whole family),
// enforces idle and absolute session windows, and issues a fresh token pair.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (*Session, error) {
	claims, err := s.jwt.Parse(refreshToken, pkgjwt.Refresh)
	if err != nil {
		return nil, ErrInvalidToken
	}
	row, err := s.repo.GetRefresh(ctx, claims.ID)
	if err != nil {
		return nil, err
	}
	if row == nil || row.revokedAt != nil {
		return nil, ErrInvalidToken
	}
	now := s.now()
	if row.usedAt != nil {
		// A rotated token presented again: the family may be compromised.
		_ = s.repo.RevokeFamily(ctx, row.familyID, now)
		return nil, ErrTokenReuse
	}
	if now.After(row.familyOrigin.Add(s.refreshTTL)) || now.After(row.issuedAt.Add(s.idleTimeout)) {
		_ = s.repo.RevokeFamily(ctx, row.familyID, now)
		return nil, ErrSessionExpired
	}
	if err := s.repo.MarkRefreshUsed(ctx, row.jti, now); err != nil {
		return nil, err
	}
	u, err := s.repo.GetUser(ctx, row.userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrUserNotFound
	}
	return s.rotate(ctx, u, row.familyID, row.jti, row.familyOrigin)
}

// Logout revokes the family of the presented refresh token.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	claims, err := s.jwt.Parse(refreshToken, pkgjwt.Refresh)
	if err != nil {
		return nil // already unusable
	}
	row, err := s.repo.GetRefresh(ctx, claims.ID)
	if err != nil || row == nil {
		return err
	}
	return s.repo.RevokeFamily(ctx, row.familyID, s.now())
}

// Me returns the caller's profile.
func (s *Service) Me(ctx context.Context, userID string) (*Profile, error) {
	u, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrUserNotFound
	}
	p := s.profile(u)
	return &p, nil
}

// mintSession starts a brand-new rotation family.
func (s *Service) mintSession(ctx context.Context, u *userRow) (*Session, error) {
	familyID := uuid.NewString()
	origin := s.now()
	return s.rotate(ctx, u, familyID, "", origin)
}

// rotate issues an access+refresh pair within a family and persists the refresh.
func (s *Service) rotate(ctx context.Context, u *userRow, familyID, parentJTI string, origin time.Time) (*Session, error) {
	access, err := s.jwt.IssueAccess(u.id, uuid.NewString())
	if err != nil {
		return nil, err
	}
	refreshJTI := uuid.NewString()
	refresh, err := s.jwt.IssueRefresh(u.id, refreshJTI, familyID, parentJTI, s.refreshTTL)
	if err != nil {
		return nil, err
	}
	var parentPtr *string
	if parentJTI != "" {
		parentPtr = &parentJTI
	}
	now := s.now()
	if err := s.repo.InsertRefresh(ctx, refreshRow{
		jti:          refreshJTI,
		userID:       u.id,
		familyID:     familyID,
		parentJTI:    parentPtr,
		tokenHash:    pkgjwt.Fingerprint(refresh),
		familyOrigin: origin,
		expiresAt:    now.Add(s.refreshTTL),
	}); err != nil {
		return nil, err
	}
	return &Session{Profile: s.profile(u), AccessToken: access, refreshToken: refresh}, nil
}

func (s *Service) profile(u *userRow) Profile {
	masked := "unknown"
	if phone, err := s.cipher.Decrypt(u.phoneCipher); err == nil {
		masked = maskPhone(phone)
	}
	return Profile{ID: u.id, PhoneMasked: masked, KYCLevel: u.kycLevel, Verified: u.phoneVerified}
}

// maskPhone keeps the country prefix and last two digits, e.g. +506****34.
func maskPhone(phone string) string {
	if len(phone) < 6 {
		return "****"
	}
	return phone[:4] + strings.Repeat("*", len(phone)-6) + phone[len(phone)-2:]
}

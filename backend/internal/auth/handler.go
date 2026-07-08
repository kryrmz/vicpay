package auth

import (
	"errors"
	"net/http"

	"github.com/vicpay/backend/internal/middleware"
	"github.com/vicpay/backend/pkg/response"
)

// Handler exposes the auth endpoints over HTTP.
type Handler struct {
	svc     *Service
	cookies *CookieWriter
}

// NewHandler builds a Handler.
func NewHandler(svc *Service, cookies *CookieWriter) *Handler {
	return &Handler{svc: svc, cookies: cookies}
}

type registerReq struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type verifyReq struct {
	PendingUserID string `json:"pendingUserId"`
	Code          string `json:"code"`
}

type resendReq struct {
	PendingUserID string `json:"pendingUserId"`
}

type loginReq struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

// Register creates a Level 0 user and dispatches a phone OTP.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if !response.Decode(w, r, &req) {
		return
	}
	id, err := h.svc.Register(r.Context(), req.Phone, req.Password)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, map[string]string{"pendingUserId": id})
}

// ResendCode re-issues the phone OTP for a pending user.
func (h *Handler) ResendCode(w http.ResponseWriter, r *http.Request) {
	var req resendReq
	if !response.Decode(w, r, &req) {
		return
	}
	if err := h.svc.ResendCode(r.Context(), req.PendingUserID); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// VerifyPhone verifies the OTP and starts the first session.
func (h *Handler) VerifyPhone(w http.ResponseWriter, r *http.Request) {
	var req verifyReq
	if !response.Decode(w, r, &req) {
		return
	}
	session, err := h.svc.VerifyPhone(r.Context(), req.PendingUserID, req.Code)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	h.writeSession(w, session, http.StatusOK)
}

// Login authenticates a verified user.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if !response.Decode(w, r, &req) {
		return
	}
	session, err := h.svc.Login(r.Context(), req.Phone, req.Password)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	h.writeSession(w, session, http.StatusOK)
}

// Refresh rotates the session using the refresh cookie.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	token, ok := h.cookies.Read(r)
	if !ok {
		response.Fail(w, http.StatusUnauthorized, "unauthorized", "no refresh cookie")
		return
	}
	session, err := h.svc.Refresh(r.Context(), token)
	if err != nil {
		h.cookies.Clear(w)
		writeAuthError(w, err)
		return
	}
	h.writeSession(w, session, http.StatusOK)
}

// Logout revokes the session and clears the cookie.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if token, ok := h.cookies.Read(r); ok {
		_ = h.svc.Logout(r.Context(), token)
	}
	h.cookies.Clear(w)
	w.WriteHeader(http.StatusNoContent)
}

// Me returns the authenticated user's profile.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserID(r.Context())
	if !ok {
		response.Fail(w, http.StatusUnauthorized, "unauthorized", "not authenticated")
		return
	}
	profile, err := h.svc.Me(r.Context(), userID)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"user": profile})
}

// writeSession sets the refresh cookie and returns the access token + profile.
func (h *Handler) writeSession(w http.ResponseWriter, s *Session, status int) {
	h.cookies.Set(w, s.RefreshToken())
	response.JSON(w, status, map[string]any{"user": s.Profile, "accessToken": s.AccessToken})
}

// writeAuthError maps domain errors to stable HTTP responses without leaking
// which factor failed (to limit account enumeration).
func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidPhone):
		response.Fail(w, http.StatusBadRequest, "invalid_phone", "phone must be in E.164 format")
	case errors.Is(err, ErrWeakPassword):
		response.Fail(w, http.StatusBadRequest, "weak_password", "password does not meet the minimum requirements")
	case errors.Is(err, ErrPhoneTaken):
		response.Fail(w, http.StatusConflict, "phone_taken", "that phone is already registered")
	case errors.Is(err, ErrInvalidCredentials), errors.Is(err, ErrInvalidToken):
		response.Fail(w, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
	case errors.Is(err, ErrPhoneNotVerified):
		response.Fail(w, http.StatusForbidden, "phone_unverified", "verify your phone to continue")
	case errors.Is(err, ErrTokenReuse), errors.Is(err, ErrSessionExpired):
		response.Fail(w, http.StatusUnauthorized, "session_ended", "please sign in again")
	case errors.Is(err, ErrUserNotFound):
		response.Fail(w, http.StatusNotFound, "not_found", "user not found")
	default:
		response.Fail(w, http.StatusInternalServerError, "internal", "internal server error")
	}
}

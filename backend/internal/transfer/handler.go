package transfer

import (
	"errors"
	"net/http"

	"github.com/vicpay/backend/internal/middleware"
	"github.com/vicpay/backend/pkg/response"
)

// Handler exposes money-movement endpoints for the authenticated user.
type Handler struct {
	svc         *Service
	demoEnabled bool // gates the demo top-up (non-production only)
}

// NewHandler builds a Handler. demoEnabled should be false in production.
func NewHandler(svc *Service, demoEnabled bool) *Handler {
	return &Handler{svc: svc, demoEnabled: demoEnabled}
}

type transferReq struct {
	ToPhone        string `json:"toPhone"`
	ToUserID       string `json:"toUserId"`
	AmountMinor    int64  `json:"amountMinor"`
	Currency       string `json:"currency"`
	IdempotencyKey string `json:"idempotencyKey"`
}

type topUpReq struct {
	AmountMinor    int64  `json:"amountMinor"`
	Currency       string `json:"currency"`
	IdempotencyKey string `json:"idempotencyKey"`
}

// Transfer moves money from the caller to another user by phone.
func (h *Handler) Transfer(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserID(r.Context())
	if !ok {
		response.Fail(w, http.StatusUnauthorized, "unauthorized", "not authenticated")
		return
	}
	var req transferReq
	if !response.Decode(w, r, &req) {
		return
	}
	// A QR payment carries the payee's user id; a manual send carries a phone.
	var (
		res *Result
		err error
	)
	if req.ToUserID != "" {
		res, err = h.svc.TransferToUser(r.Context(), userID, req.ToUserID, req.AmountMinor, req.Currency, req.IdempotencyKey)
	} else {
		res, err = h.svc.Transfer(r.Context(), userID, req.ToPhone, req.AmountMinor, req.Currency, req.IdempotencyKey)
	}
	if err != nil {
		writeTransferError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, res)
}

// TopUp credits the caller's wallet (demo only).
func (h *Handler) TopUp(w http.ResponseWriter, r *http.Request) {
	if !h.demoEnabled {
		response.Fail(w, http.StatusForbidden, "disabled", "demo top-up is disabled")
		return
	}
	userID, ok := middleware.UserID(r.Context())
	if !ok {
		response.Fail(w, http.StatusUnauthorized, "unauthorized", "not authenticated")
		return
	}
	var req topUpReq
	if !response.Decode(w, r, &req) {
		return
	}
	res, err := h.svc.TopUp(r.Context(), userID, req.AmountMinor, req.Currency, req.IdempotencyKey)
	if err != nil {
		writeTransferError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, res)
}

func writeTransferError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidAmount), errors.Is(err, ErrUnsupportedCurrency), errors.Is(err, ErrInvalidRecipient), errors.Is(err, ErrSelfTransfer):
		response.Fail(w, http.StatusBadRequest, "invalid_transfer", err.Error())
	case errors.Is(err, ErrRecipientNotFound):
		response.Fail(w, http.StatusNotFound, "recipient_not_found", "no account with that phone")
	case errors.Is(err, ErrRecipientUnverified):
		response.Fail(w, http.StatusConflict, "recipient_unverified", "the recipient has not verified their phone")
	case errors.Is(err, ErrInsufficientFunds):
		response.Fail(w, http.StatusUnprocessableEntity, "insufficient_funds", "not enough balance")
	case errors.Is(err, ErrLimitExceeded):
		response.Fail(w, http.StatusUnprocessableEntity, "limit_exceeded", "this exceeds your KYC limit; verify your identity to raise it")
	default:
		response.Fail(w, http.StatusInternalServerError, "internal", "could not complete the transfer")
	}
}

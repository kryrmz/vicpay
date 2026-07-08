package wallet

import (
	"net/http"

	"github.com/vicpay/backend/internal/middleware"
	"github.com/vicpay/backend/pkg/response"
)

// Handler serves wallet read endpoints for the authenticated user.
type Handler struct{ svc *Service }

// NewHandler builds a Handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Balances returns the caller's wallet balances.
func (h *Handler) Balances(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserID(r.Context())
	if !ok {
		response.Fail(w, http.StatusUnauthorized, "unauthorized", "not authenticated")
		return
	}
	balances, err := h.svc.Balances(r.Context(), userID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "internal", "could not load balances")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"wallets": balances})
}

// Transactions returns the caller's recent transactions.
func (h *Handler) Transactions(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserID(r.Context())
	if !ok {
		response.Fail(w, http.StatusUnauthorized, "unauthorized", "not authenticated")
		return
	}
	txs, err := h.svc.Transactions(r.Context(), userID, 50)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "internal", "could not load transactions")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"transactions": txs})
}

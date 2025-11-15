package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
	"naimuBack/internal/services"
)

type TopHandler struct {
	Service        *services.TopService
	InvoiceRepo    *repositories.InvoiceRepo
	PaymentService *services.AirbapayService
}

type topActivationPaymentRequest struct {
	models.TopActivationRequest
	Amount      float64 `json:"amount"`
	Description string  `json:"description,omitempty"`
}

func (h *TopHandler) Activate(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Service == nil || h.InvoiceRepo == nil || h.PaymentService == nil {
		http.Error(w, "top payments not configured", http.StatusInternalServerError)
		return
	}

	userID, ok := r.Context().Value("user_id").(int)
	if !ok || userID == 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req topActivationPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Amount <= 0 {
		http.Error(w, "invalid amount", http.StatusBadRequest)
		return
	}

	if err := h.Service.EnsureActivationAllowed(r.Context(), userID, req.TopActivationRequest); err != nil {
		switch {
		case errors.Is(err, models.ErrInvalidTopType),
			errors.Is(err, models.ErrInvalidTopDuration),
			errors.Is(err, models.ErrInvalidTopID):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, repositories.ErrListingNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, services.ErrTopForbidden):
			http.Error(w, err.Error(), http.StatusForbidden)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	description := strings.TrimSpace(req.Description)
	if description == "" {
		description = fmt.Sprintf("top promotion %s #%d", req.Type, req.ID)
	}

	invID, err := h.InvoiceRepo.CreateInvoice(r.Context(), userID, req.Amount, description)
	if err != nil {
		http.Error(w, "create invoice: "+err.Error(), http.StatusInternalServerError)
		return
	}

	payload, err := json.Marshal(req.TopActivationRequest)
	if err != nil {
		http.Error(w, "failed to encode top payload", http.StatusInternalServerError)
		return
	}
	if _, err := h.InvoiceRepo.AddTarget(r.Context(), invID, invoiceTargetTopActivation, int64(req.ID), payload); err != nil {
		http.Error(w, "store invoice target: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := h.PaymentService.CreatePaymentLink(r.Context(), invID, req.Amount, description)
	if err != nil {
		_ = h.InvoiceRepo.UpdateStatus(r.Context(), invID, "error")
		http.Error(w, "create payment link: "+err.Error(), airbapayErrorStatus(err))
		return
	}

	if err := h.InvoiceRepo.UpdateStatus(r.Context(), invID, strings.ToLower(resp.Status)); err != nil {
		fmt.Println("airbapay: failed to update invoice status:", err)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"inv_id":      invID,
		"order_id":    resp.OrderID,
		"invoice_id":  resp.InvoiceID,
		"payment_url": resp.PaymentURL,
		"status":      resp.Status,
	})
}

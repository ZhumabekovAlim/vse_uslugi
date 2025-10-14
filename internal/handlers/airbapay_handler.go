package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"naimuBack/internal/repositories"
	"naimuBack/internal/services"
)

type AirbapayHandler struct {
	Service     *services.AirbapayService
	InvoiceRepo *repositories.InvoiceRepo
}

func NewAirbapayHandler(s *services.AirbapayService, r *repositories.InvoiceRepo) *AirbapayHandler {
	return &AirbapayHandler{Service: s, InvoiceRepo: r}
}

func (h *AirbapayHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil || h.InvoiceRepo == nil {
		http.Error(w, "airbapay not initialized", http.StatusInternalServerError)
		return
	}

	var req struct {
		UserID      int     `json:"user_id"`
		Amount      float64 `json:"amount"`
		Description string  `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	invID, err := h.InvoiceRepo.CreateInvoice(r.Context(), req.UserID, req.Amount, req.Description)
	if err != nil {
		http.Error(w, "create invoice: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := h.Service.CreatePaymentLink(r.Context(), invID, req.Amount, req.Description)
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

func airbapayErrorStatus(err error) int {
	var apiErr *services.AirbapayError
	if errors.As(err, &apiErr) {
		if apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
			return apiErr.StatusCode
		}
	}
	return http.StatusBadGateway
}

func (h *AirbapayHandler) Callback(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil || h.InvoiceRepo == nil {
		http.Error(w, "airbapay not initialized", http.StatusInternalServerError)
		return
	}

	payload, err := h.Service.ParseCallback(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(payload.InvoiceID) == "" {
		http.Error(w, "missing invoice_id", http.StatusBadRequest)
		return
	}
	if !h.Service.ValidateCallbackSignature(payload) {
		http.Error(w, "invalid callback signature", http.StatusBadRequest)
		return
	}

	// invoice_id мы сами задавали как наш invID → можно использовать его
	invID, err := strconv.Atoi(payload.InvoiceID)
	if err != nil {
		http.Error(w, "invalid invoice_id", http.StatusBadRequest)
		return
	}

	status := strings.ToLower(payload.Status)
	switch status {
	case "success", "succeeded", "paid", "done", "approved":
		if err := h.InvoiceRepo.MarkPaid(r.Context(), invID); err != nil {
			http.Error(w, "mark paid: "+err.Error(), http.StatusInternalServerError)
			return
		}
	case "failure", "failed", "cancelled", "rejected", "error":
		if err := h.InvoiceRepo.UpdateStatus(r.Context(), invID, "failed"); err != nil {
			http.Error(w, "mark failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		if err := h.InvoiceRepo.UpdateStatus(r.Context(), invID, status); err != nil {
			http.Error(w, "update status: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":     "ok",
		"invoice_id": payload.InvoiceID,
	})
}
func (h *AirbapayHandler) SuccessRedirect(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (h *AirbapayHandler) FailureRedirect(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "failure"})
}

func (h *AirbapayHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	if h.InvoiceRepo == nil {
		http.Error(w, "airbapay not initialized", http.StatusInternalServerError)
		return
	}

	userIDStr := r.URL.Query().Get(":user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	invoices, err := h.InvoiceRepo.GetByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, "get invoices: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(invoices)
}

package handlers

import (
	"encoding/json"
	"net/http"

	services "naimuBack/internal/services"
)

// RobokassaHandler handles payment requests and callbacks.
type RobokassaHandler struct {
	Service *services.RobokassaService
}

// CreatePayment generates a Robokassa payment URL for the provided invoice.
func (h *RobokassaHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InvoiceID   int     `json:"invoice_id"`
		Amount      float64 `json:"amount"`
		Description string  `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	url, err := h.Service.GeneratePayURL(req.InvoiceID, req.Amount, req.Description)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": url})
}

// Result handles Robokassa payment notifications.
func (h *RobokassaHandler) Result(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	outSum := r.FormValue("OutSum")
	invID := r.FormValue("InvId")
	signature := r.FormValue("SignatureValue")
	if !h.Service.VerifyResult(outSum, invID, signature) {
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}
	w.Write([]byte("OK" + invID))
}

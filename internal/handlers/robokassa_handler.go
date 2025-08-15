package handlers

import (
	"encoding/json"
	"naimuBack/internal/repositories"
	"net/http"
	"strconv"

	services "naimuBack/internal/services"
)

type RobokassaHandler struct {
	Service     *services.RobokassaService
	InvoiceRepo *repositories.InvoiceRepo
}

// POST /robokassa/pay
// { "invoice_id": 678678, "amount": 100.00, "description": "Товары для животных" }
func (h *RobokassaHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Amount      float64 `json:"amount"`
		Description string  `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 1) генерим inv_id на бэке (вставкой в БД)
	invID, err := h.InvoiceRepo.CreateInvoice(r.Context(), req.Amount, req.Description)
	if err != nil {
		http.Error(w, "create invoice: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 2) строим URL оплаты
	payURL, err := h.Service.GeneratePayURL(invID, req.Amount, req.Description)
	if err != nil {
		http.Error(w, "build pay url: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"inv_id": invID,
		"url":    payURL,
	})
}

// POST /robokassa/result  (application/x-www-form-urlencoded)
// В Robokassa обычно приходят как минимум: OutSum, InvId, SignatureValue, IsTest (для теста)
func (h *RobokassaHandler) Result(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	outSum := r.FormValue("OutSum")
	invID := r.FormValue("InvId")
	signature := r.FormValue("SignatureValue")
	isTest := r.FormValue("IsTest") == "1"
	if r.FormValue("IsTest") == "" && h.Service.IsTest {
		isTest = true
	}

	if !h.Service.VerifyResult(outSum, invID, signature, isTest) {
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}

	// отметить оплату
	if err := h.InvoiceRepo.MarkPaid(r.Context(), atoi(invID)); err != nil {
		http.Error(w, "mark paid: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_, _ = w.Write([]byte("OK" + invID))
}

func atoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

package handlers

import (
	"encoding/json"
	"net/http"

	services "naimuBack/internal/services"
)

type RobokassaHandler struct {
	Service *services.RobokassaService
}

// POST /robokassa/pay
// { "invoice_id": 678678, "amount": 100.00, "description": "Товары для животных" }
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
	_ = json.NewEncoder(w).Encode(map[string]string{"url": url})
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

	// определяем тестовый ли это колбэк
	isTestParam := r.FormValue("IsTest")
	isTest := isTestParam == "1"
	// если вдруг Robokassa не прислёт IsTest, можешь подстраховаться глобальным флагом:
	if isTestParam == "" && h.Service.IsTest {
		isTest = true
	}

	if !h.Service.VerifyResult(outSum, invID, signature, isTest) {
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}

	// TODO: отметить оплату как успешную (в тесте НЕ списывать деньги/услуги, а просто логировать)
	_, _ = w.Write([]byte("OK" + invID))
}

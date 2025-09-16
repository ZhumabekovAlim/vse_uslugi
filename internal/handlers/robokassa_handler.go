package handlers

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"naimuBack/internal/repositories"
	"naimuBack/internal/services"
)

type RobokassaHandler struct {
	Service     *services.RobokassaService
	InvoiceRepo *repositories.InvoiceRepo
}

func NewRobokassaHandler(s *services.RobokassaService, r *repositories.InvoiceRepo) *RobokassaHandler {
	return &RobokassaHandler{Service: s, InvoiceRepo: r}
}

func (h *RobokassaHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil || h.InvoiceRepo == nil {
		http.Error(w, "robokassa not initialized", http.StatusInternalServerError)
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
	url, err := h.Service.GeneratePayURL(invID, req.Amount, req.Description)
	if err != nil {
		http.Error(w, "build pay url: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"inv_id": invID,
		"url":    url,
	})
}

func (h *RobokassaHandler) Result(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil || h.InvoiceRepo == nil {
		http.Error(w, "robokassa not initialized", http.StatusInternalServerError)
		return
	}
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
	expectedRaw := fmt.Sprintf("%s:%s:%s", outSum, invID, h.Service.Pass2(isTest))
	expectedSig := fmt.Sprintf("%x", md5.Sum([]byte(expectedRaw)))
	fmt.Println("[ROBOKASSA RESULT] raw:", expectedRaw)
	fmt.Println("[ROBOKASSA RESULT] expected:", strings.ToUpper(expectedSig), "got:", strings.ToUpper(signature))

	if !h.Service.VerifyResult(outSum, invID, signature, isTest) {
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}
	id, _ := strconv.Atoi(invID)
	if err := h.InvoiceRepo.MarkPaid(r.Context(), id); err != nil {
		http.Error(w, "mark paid: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK" + invID))
}

func (h *RobokassaHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	if h.InvoiceRepo == nil {
		http.Error(w, "robokassa not initialized", http.StatusInternalServerError)
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
